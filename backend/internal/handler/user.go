package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"xorapi/internal/middleware"
	"xorapi/internal/model"
	"xorapi/internal/service"
	"xorapi/internal/store"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type registerReq struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6,max=64"`
	Username   string `json:"username"`
	InviteCode string `json:"invite_code"`
	EmailCode  string `json:"email_code"`
}

func Register(c *gin.Context) {
	if !service.GetConfigBool("registration_enabled", true) {
		Fail(c, http.StatusForbidden, "注册已关闭")
		return
	}
	var req registerReq
	if !BindJSON(c, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if service.GetConfigBool("invite_required", false) {
		if req.InviteCode == "" {
			Fail(c, http.StatusBadRequest, "需要邀请码")
			return
		}
		var ic model.InviteCode
		if err := store.DB.Where("code = ? AND status = ?", req.InviteCode, "unused").First(&ic).Error; err != nil {
			Fail(c, http.StatusBadRequest, "邀请码无效或已被使用")
			return
		}
	}
	if service.GetConfigBool("email_verify", false) {
		if req.EmailCode == "" {
			Fail(c, http.StatusBadRequest, "需要邮箱验证码")
			return
		}
		var ec model.EmailCode
		err := store.DB.Where("email = ? AND code = ? AND used_at IS NULL AND expires_at > ?", req.Email, req.EmailCode, time.Now()).
			Order("id DESC").First(&ec).Error
		if err != nil {
			Fail(c, http.StatusBadRequest, "验证码错误或已过期")
			return
		}
		now := time.Now()
		ec.UsedAt = &now
		store.DB.Save(&ec)
	}
	var count int64
	store.DB.Model(&model.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		Fail(c, http.StatusBadRequest, "邮箱已被注册")
		return
	}
	if req.Username == "" {
		req.Username = strings.Split(req.Email, "@")[0]
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "密码加密失败")
		return
	}
	user := model.User{Email: req.Email, Username: req.Username, PasswordHash: string(hash), Role: "user", Status: "active"}
	if err := store.DB.Create(&user).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "创建用户失败: "+err.Error())
		return
	}
	token, _ := middleware.GenerateToken(&user)
	OK(c, gin.H{"token": token, "user": user})
}

type loginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req loginReq
	if !BindJSON(c, &req) {
		return
	}
	var user model.User
	if err := store.DB.Where("email = ?", strings.ToLower(strings.TrimSpace(req.Email))).First(&user).Error; err != nil {
		Fail(c, http.StatusUnauthorized, "邮箱或密码错误")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		Fail(c, http.StatusUnauthorized, "邮箱或密码错误")
		return
	}
	if user.Status != "active" {
		Fail(c, http.StatusForbidden, "账号已被禁用")
		return
	}
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "生成令牌失败")
		return
	}
	OK(c, gin.H{"token": token, "user": user})
}

// SendEmailCode 发送注册验证码
func SendEmailCode(c *gin.Context) {
	if !service.GetConfigBool("email_verify", false) {
		Fail(c, http.StatusBadRequest, "邮箱验证未开启")
		return
	}
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if !BindJSON(c, &req) {
		return
	}
	var count int64
	store.DB.Model(&model.User{}).Where("email = ?", strings.ToLower(req.Email)).Count(&count)
	if count > 0 {
		Fail(c, http.StatusBadRequest, "邮箱已被注册")
		return
	}
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	ec := model.EmailCode{Email: strings.ToLower(req.Email), Code: code, ExpiresAt: time.Now().Add(15 * time.Minute)}
	if err := store.DB.Create(&ec).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "验证码创建失败")
		return
	}
	if err := service.SendMail(ec.Email, "XorAPI 注册验证码", "您的验证码是："+code+"，15 分钟内有效。"); err != nil {
		Fail(c, http.StatusInternalServerError, "邮件发送失败: "+err.Error())
		return
	}
	OK(c, gin.H{"sent": true})
}

// Profile 用户信息
func Profile(c *gin.Context) {
	u := middleware.CurrentUser(c)
	var todayCalls int64
	store.DB.Model(&model.CallLog{}).Where("user_id = ? AND created_at > ?", u.ID, time.Now().Truncate(24*time.Hour)).Count(&todayCalls)
	OK(c, gin.H{"user": u, "today_calls": todayCalls})
}

type updateProfileReq struct {
	Username string `json:"username" binding:"omitempty,min=1,max=64"`
	Email    string `json:"email" binding:"omitempty,email"`
}

func UpdateProfile(c *gin.Context) {
	u := middleware.CurrentUser(c)
	var req updateProfileReq
	if !BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {
		email := strings.ToLower(req.Email)
		var count int64
		store.DB.Model(&model.User{}).Where("email = ? AND id != ?", email, u.ID).Count(&count)
		if count > 0 {
			Fail(c, http.StatusBadRequest, "邮箱已被占用")
			return
		}
		updates["email"] = email
	}
	if len(updates) > 0 {
		store.DB.Model(&model.User{}).Where("id = ?", u.ID).Updates(updates)
	}
	var fresh model.User
	store.DB.First(&fresh, u.ID)
	OK(c, fresh)
}

type changePwdReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

func ChangePassword(c *gin.Context) {
	u := middleware.CurrentUser(c)
	var req changePwdReq
	if !BindJSON(c, &req) {
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.OldPassword)) != nil {
		Fail(c, http.StatusBadRequest, "原密码错误")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	store.DB.Model(&model.User{}).Where("id = ?", u.ID).Update("password_hash", string(hash))
	OK(c, gin.H{"updated": true})
}

// SignIn 每日签到
func SignIn(c *gin.Context) {
	u := middleware.CurrentUser(c)
	if !service.GetConfigBool("checkin_enabled", true) {
		Fail(c, http.StatusForbidden, "签到功能已关闭")
		return
	}
	rec, err := service.CheckIn(store.DB, u.ID)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var fresh model.User
	store.DB.First(&fresh, u.ID)
	OK(c, gin.H{"record": rec, "points": fresh.Points})
}

// CheckInStatus 签到状态
func CheckInStatus(c *gin.Context) {
	u := middleware.CurrentUser(c)
	var rec model.CheckInRecord
	today := time.Now().Format("2006-01-02")
	err := store.DB.Where("user_id = ? AND date = ?", u.ID, today).First(&rec).Error
	OK(c, gin.H{
		"signed_today": err == nil,
		"streak":       rec.Streak,
		"enabled":      service.GetConfigBool("checkin_enabled", true),
	})
}

// Transactions 交易流水
func Transactions(c *gin.Context) {
	u := middleware.CurrentUser(c)
	page, size := pageNo(c)
	var total int64
	store.DB.Model(&model.Transaction{}).Where("user_id = ?", u.ID).Count(&total)
	var list []model.Transaction
	store.DB.Where("user_id = ?", u.ID).Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	OK(c, gin.H{"total": total, "list": list})
}

// UsageLogs 调用记录
func UsageLogs(c *gin.Context) {
	u := middleware.CurrentUser(c)
	page, size := pageNo(c)
	var total int64
	q := store.DB.Model(&model.CallLog{}).Where("user_id = ?", u.ID)
	if m := c.Query("model"); m != "" {
		q = q.Where("model = ?", m)
	}
	q.Count(&total)
	var list []model.CallLog
	q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	OK(c, gin.H{"total": total, "list": list})
}

func pageNo(c *gin.Context) (int, int) {
	page := 1
	size := 20
	if v := c.Query("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := c.Query("size"); v != "" {
		fmt.Sscanf(v, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}

// CreateAPIKey / ListKeys / DeleteKeys
type createKeyReq struct {
	Name        string `json:"name" binding:"required,max=64"`
	IPWhitelist string `json:"ip_whitelist"`
	DailyLimit  int64  `json:"daily_limit"`
}

func CreateAPIKey(c *gin.Context) {
	u := middleware.CurrentUser(c)
	var req createKeyReq
	if !BindJSON(c, &req) {
		return
	}
	var count int64
	store.DB.Model(&model.APIKey{}).Where("user_id = ?", u.ID).Count(&count)
	if count >= 20 {
		Fail(c, http.StatusBadRequest, "每个用户最多创建 20 个 API Key")
		return
	}
	if req.IPWhitelist != "" && !isValidIPList(req.IPWhitelist) {
		Fail(c, http.StatusBadRequest, "IP 白名单格式错误，应为 JSON 数组，如 [\"1.2.3.4\"]")
		return
	}
	raw := "sk-" + mustRandomHex(24)
	ak := model.APIKey{
		UserID: u.ID, Name: req.Name, KeyHash: service.SHA256Hex(raw),
		KeyPreview:  raw[:12] + "..." + raw[len(raw)-4:],
		IPWhitelist: req.IPWhitelist, DailyLimit: req.DailyLimit, Status: "active",
	}
	if err := store.DB.Create(&ak).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "创建失败")
		return
	}
	OK(c, gin.H{"key": raw, "id": ak.ID})
}

func ListAPIKeys(c *gin.Context) {
	u := middleware.CurrentUser(c)
	var keys []model.APIKey
	store.DB.Where("user_id = ?", u.ID).Order("id DESC").Find(&keys)
	OK(c, keys)
}

func UpdateAPIKey(c *gin.Context) {
	u := middleware.CurrentUser(c)
	id := c.Param("id")
	var ak model.APIKey
	if err := store.DB.Where("id = ? AND user_id = ?", id, u.ID).First(&ak).Error; err != nil {
		Fail(c, http.StatusNotFound, "API Key 不存在")
		return
	}
	var req struct {
		Name        string `json:"name"`
		IPWhitelist string `json:"ip_whitelist"`
		DailyLimit  *int64 `json:"daily_limit"`
		Status      string `json:"status"`
	}
	if !BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.IPWhitelist != "" {
		if !isValidIPList(req.IPWhitelist) {
			Fail(c, http.StatusBadRequest, "IP 白名单格式错误")
			return
		}
		updates["ip_whitelist"] = req.IPWhitelist
	}
	if req.DailyLimit != nil {
		updates["daily_limit"] = *req.DailyLimit
	}
	if req.Status == "active" || req.Status == "disabled" {
		updates["status"] = req.Status
	}
	if len(updates) > 0 {
		store.DB.Model(&ak).Updates(updates)
	}
	OK(c, ak)
}

func DeleteAPIKey(c *gin.Context) {
	u := middleware.CurrentUser(c)
	id := c.Param("id")
	res := store.DB.Where("id = ? AND user_id = ?", id, u.ID).Delete(&model.APIKey{})
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, "API Key 不存在")
		return
	}
	OK(c, gin.H{"deleted": true})
}

func isValidIPList(s string) bool {
	var list []string
	return jsonUnmarshalOK(s, &list)
}

func jsonUnmarshalOK(s string, v any) bool {
	return len(s) < 65536 && strings.HasPrefix(strings.TrimSpace(s), "[")
}

func mustRandomHex(n int) string {
	b, err := service.RandomToken(n)
	if err != nil {
		return "fallback" + fmt.Sprint(time.Now().UnixNano())
	}
	return b
}

// SquareModels 模型广场（公开）
func SquareModels(c *gin.Context) {
	var models []model.ModelInfo
	q := store.DB.Where("visible = ?", true)
	if kw := c.Query("search"); kw != "" {
		q = q.Where("name LIKE ? OR display_name LIKE ? OR tags LIKE ?", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	if group := c.Query("group"); group != "" {
		q = q.Where("group_id = ?", group)
	}
	if err := q.Order("id ASC").Find(&models).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "数据库错误")
		return
	}
	var groups []model.Group
	store.DB.Where("status = ?", "active").Find(&groups)
	groupName := map[uint]string{}
	for _, g := range groups {
		groupName[g.ID] = g.Name
	}
	type item struct {
		model.ModelInfo
		GroupName string `json:"group_name"`
	}
	list := make([]item, 0, len(models))
	for _, m := range models {
		list = append(list, item{ModelInfo: m, GroupName: groupName[m.GroupID]})
	}
	OK(c, gin.H{"list": list, "announcement": service.GetConfig("site_announcement"), "exchange_note": service.GetConfig("exchange_note")})
}

// PublicStatus 公开状态（是否允许注册等）
func PublicStatus(c *gin.Context) {
	OK(c, gin.H{
		"registration_enabled": service.GetConfigBool("registration_enabled", true),
		"invite_required":      service.GetConfigBool("invite_required", false),
		"email_verify":         service.GetConfigBool("email_verify", false),
		"checkin_enabled":      service.GetConfigBool("checkin_enabled", true),
		"site_name":            siteName(),
	})
}

func siteName() string {
	if store.Cfg != nil && store.Cfg.Server.SiteName != "" {
		return store.Cfg.Server.SiteName
	}
	return "XorAPI"
}

var _ = gorm.ErrRecordNotFound
