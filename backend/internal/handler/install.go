package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"xorapi/internal/config"
	"xorapi/internal/database"
	"xorapi/internal/model"
	"xorapi/internal/store"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	ConfigPath string
	DataDir    string
	installing atomic.Bool
)

// IsInstalled 配置文件与锁文件同时存在视为已安装
func IsInstalled() bool {
	if ConfigPath == "" {
		return false
	}
	if _, err := os.Stat(ConfigPath); err != nil {
		return false
	}
	lock := filepath.Join(DataDir, "install.lock")
	if _, err := os.Stat(lock); err != nil {
		return false
	}
	return true
}

func InstallStatus(c *gin.Context) {
	OK(c, gin.H{"installed": IsInstalled()})
}

type installReq struct {
	MySQL struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Name     string `json:"name"`
	} `json:"mysql"`
	Redis struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Password string `json:"password"`
		DB       int    `json:"db"`
	} `json:"redis"`
	Admin struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Username string `json:"username"`
	} `json:"admin"`
	SiteName string `json:"site_name"`
}

// InstallPreflight 环境检测
func InstallPreflight(c *gin.Context) {
	var req installReq
	if !BindJSON(c, &req) {
		return
	}
	result := gin.H{
		"mysql": testMySQL(req.MySQL.Host, req.MySQL.Port, req.MySQL.User, req.MySQL.Password, req.MySQL.Name),
		"redis": testRedis(req.Redis.Host, req.Redis.Port, req.Redis.Password, req.Redis.DB),
		"dir":   testDir(),
	}
	OK(c, result)
}

func testMySQL(host string, port int, user, password, dbname string) gin.H {
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 3306
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return gin.H{"ok": false, "msg": err.Error()}
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return gin.H{"ok": false, "msg": "连接失败: " + err.Error()}
	}
	var version string
	if err := db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		return gin.H{"ok": false, "msg": "无法获取 MySQL 版本: " + err.Error()}
	}
	if major, minor, patch, ok := parseMySQLVersion(version); ok && !mySQLVersionOK(major, minor, patch) {
		return gin.H{"ok": false, "msg": "MySQL 版本过低（当前 " + version + "），需要 5.6.4 及以上"}
	}
	if dbname != "" {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", dbname).Scan(&count); err == nil && count == 0 {
			if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS `" + dbname + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
				return gin.H{"ok": false, "msg": "数据库不存在且无权创建: " + err.Error()}
			}
		}
	}
	return gin.H{"ok": true, "msg": "连接成功（MySQL " + version + "）"}
}

// parseMySQLVersion 解析 "5.6.50-log" / "8.0.36" / "10.4.12-MariaDB" 形式的版本号
func parseMySQLVersion(v string) (major, minor, patch int, ok bool) {
	parts := strings.SplitN(v, "-", 2)
	nums := strings.Split(parts[0], ".")
	if len(nums) < 2 {
		return 0, 0, 0, false
	}
	major, err1 := strconv.Atoi(nums[0])
	minor, err2 := strconv.Atoi(nums[1])
	if err1 != nil || err2 != nil {
		return 0, 0, 0, false
	}
	if len(nums) > 2 {
		patch, _ = strconv.Atoi(strings.TrimRight(nums[2], "abcdefghijklmnopqrstuvwxyz"))
	}
	return major, minor, patch, true
}

// mySQLVersionOK 下限 5.6.4（DATETIME 毫秒精度的最低版本）；MariaDB 10.x 天然满足
func mySQLVersionOK(major, minor, patch int) bool {
	if major > 5 {
		return true
	}
	if major == 5 && (minor > 6 || (minor == 6 && patch >= 4)) {
		return true
	}
	return false
}

func testRedis(host string, port int, password string, db int) gin.H {
	if host == "" {
		return gin.H{"ok": false, "msg": "未填写"}
	}
	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", host, port), Password: password, DB: db})
	defer rdb.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return gin.H{"ok": false, "msg": "连接失败: " + err.Error()}
	}
	return gin.H{"ok": true, "msg": "连接成功"}
}

func testDir() gin.H {
	dir := DataDir
	if dir == "" {
		dir = "./data"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return gin.H{"ok": false, "msg": "无法创建数据目录: " + err.Error()}
	}
	probe := filepath.Join(dir, ".write_test")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		return gin.H{"ok": false, "msg": "目录不可写: " + err.Error()}
	}
	os.Remove(probe)
	return gin.H{"ok": true, "msg": "可写"}
}

// InstallExecute 执行安装
func InstallExecute(c *gin.Context) {
	if IsInstalled() {
		Fail(c, http.StatusForbidden, "系统已安装，如需重装请删除 config.yaml 与 data/install.lock")
		return
	}
	if !installing.CompareAndSwap(false, true) {
		Fail(c, http.StatusConflict, "安装正在进行中")
		return
	}
	defer installing.Store(false)

	var req installReq
	if !BindJSON(c, &req) {
		return
	}
	if req.MySQL.Host == "" {
		req.MySQL.Host = "127.0.0.1"
	}
	if req.MySQL.Port == 0 {
		req.MySQL.Port = 3306
	}
	if req.Redis.Port == 0 {
		req.Redis.Port = 6379
	}
	if req.MySQL.Name == "" {
		Fail(c, http.StatusBadRequest, "数据库名不能为空")
		return
	}
	if req.Admin.Email == "" || len(req.Admin.Password) < 6 {
		Fail(c, http.StatusBadRequest, "管理员邮箱必填，密码至少 6 位")
		return
	}

	// 1. 建库
	base, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		req.MySQL.User, req.MySQL.Password, req.MySQL.Host, req.MySQL.Port))
	if err != nil {
		Fail(c, http.StatusBadRequest, "MySQL 连接失败: "+err.Error())
		return
	}
	if _, err := base.Exec("CREATE DATABASE IF NOT EXISTS `" + req.MySQL.Name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		base.Close()
		Fail(c, http.StatusBadRequest, "创建数据库失败（请先手动建库）: "+err.Error())
		return
	}
	base.Close()

	// 2. 迁移
	cfg := config.Default()
	cfg.Database = config.DatabaseConfig{
		Host: req.MySQL.Host, Port: req.MySQL.Port,
		User: req.MySQL.User, Password: req.MySQL.Password, Name: req.MySQL.Name,
	}
	cfg.Redis = config.RedisConfig{
		Host: req.Redis.Host, Port: req.Redis.Port,
		Password: req.Redis.Password, DB: req.Redis.DB,
	}
	if req.SiteName != "" {
		cfg.Server.SiteName = req.SiteName
	}
	cfg.Server.JWTSecret = config.RandomHex(32)
	cfg.Server.EncryptKey = config.RandomHex(32)
	cfg.Server.DataDir = DataDir

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		Fail(c, http.StatusBadRequest, "数据库初始化失败: "+err.Error())
		return
	}
	if err := database.Migrate(db); err != nil {
		Fail(c, http.StatusInternalServerError, "数据表创建失败: "+err.Error())
		return
	}
	if err := database.SeedConfigs(db); err != nil {
		Fail(c, http.StatusInternalServerError, "默认配置写入失败: "+err.Error())
		return
	}

	// 3. 管理员
	if req.Admin.Username == "" {
		req.Admin.Username = "admin"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Admin.Password), bcrypt.DefaultCost)
	admin := model.User{
		Email: req.Admin.Email, Username: req.Admin.Username,
		PasswordHash: string(hash), Role: "admin", Status: "active", EmailVerified: true,
	}
	if err := db.Create(&admin).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "管理员创建失败: "+err.Error())
		return
	}

	// 4. 写配置与锁文件
	if err := cfg.Save(ConfigPath); err != nil {
		Fail(c, http.StatusInternalServerError, "配置文件写入失败: "+err.Error())
		return
	}
	if err := os.MkdirAll(DataDir, 0o755); err != nil {
		Fail(c, http.StatusInternalServerError, "数据目录创建失败: "+err.Error())
		return
	}
	lockPath := filepath.Join(DataDir, "install.lock")
	if err := os.WriteFile(lockPath, []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
		Fail(c, http.StatusInternalServerError, "锁文件写入失败: "+err.Error())
		return
	}

	// 5. 热加载运行时（免重启）
	store.Cfg = cfg
	store.DB = db
	store.InitRedis(cfg)

	OK(c, gin.H{"installed": true})
}
