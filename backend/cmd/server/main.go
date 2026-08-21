package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"xorapi/internal/config"
	"xorapi/internal/handler"
	"xorapi/internal/router"
	"xorapi/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	cfgPath := flag.String("c", "config.yaml", "配置文件路径")
	port := flag.Int("port", 0, "覆盖监听端口")
	flag.Parse()

	if envCfg := os.Getenv("XORAPI_CONFIG"); envCfg != "" {
		*cfgPath = envCfg
	}

	handler.ConfigPath = *cfgPath
	handler.DataDir = "./data"

	cfg, err := config.Load(*cfgPath)
	installed := false
	if err == nil {
		store.Cfg = cfg
		if cfg.Server.DataDir != "" {
			handler.DataDir = cfg.Server.DataDir
		}
		installed = handler.IsInstalled()
	}
	if !installed {
		store.Cfg = config.Default()
	}

	if installed {
		if err := store.InitDB(store.Cfg); err != nil {
			log.Fatalf("数据库连接失败: %v（请检查 config.yaml）", err)
		}
		store.InitRedis(store.Cfg)
	}

	listen := fmt.Sprintf(":%d", store.Cfg.Server.Port)
	if *port != 0 {
		listen = fmt.Sprintf(":%d", *port)
	}

	gin.SetMode(gin.ReleaseMode)
	engine := router.New()

	abs, _ := filepath.Abs(*cfgPath)
	log.Printf("XorAPI 启动中... 监听 %s | 配置: %s | 状态: %s",
		listen, abs, map[bool]string{true: "已安装", false: "待安装（/install）"}[installed])
	if err := http.ListenAndServe(listen, engine); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
