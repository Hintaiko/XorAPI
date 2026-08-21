package router

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"xorapi/internal/handler"
	"xorapi/internal/middleware"
	"xorapi/internal/relay"
	"xorapi/internal/service"
	"xorapi/internal/web"

	"github.com/gin-gonic/gin"
)

func relayHandler(proto string) gin.HandlerFunc {
	return func(c *gin.Context) { relay.HandleRelay(c, proto) }
}

// installGate 安装门禁：未安装时仅放行安装接口与静态页面，安装完成即刻生效（无需重启）
func installGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if handler.IsInstalled() {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/install/") ||
			!strings.HasPrefix(path, "/api") && !strings.HasPrefix(path, "/v1") && !strings.HasPrefix(path, "/anthropic") {
			c.Next()
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "系统未安装，请先完成 /install 安装向导"})
		c.Abort()
	}
}

func New() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), cors(), installGate())
	r.MaxMultipartMemory = 32 << 20

	api := r.Group("/api")

	// 安装接口（始终可用）
	api.GET("/install/status", handler.InstallStatus)
	api.POST("/install/preflight", handler.InstallPreflight)
	api.POST("/install/execute", handler.InstallExecute)

	// 公开接口
	api.GET("/status", handler.PublicStatus)
	api.GET("/square/models", handler.SquareModels)
	api.POST("/auth/register", handler.Register)
	api.POST("/auth/login", handler.Login)
	api.POST("/auth/email-code", handler.SendEmailCode)

	// 用户接口（JWT）
	u := api.Group("", middleware.JWTAuth())
	u.GET("/user/profile", handler.Profile)
	u.PUT("/user/profile", handler.UpdateProfile)
	u.POST("/user/password", handler.ChangePassword)
	u.POST("/user/signin", handler.SignIn)
	u.GET("/user/signin/status", handler.CheckInStatus)
	u.GET("/user/transactions", handler.Transactions)
	u.GET("/user/logs", handler.UsageLogs)
	u.POST("/keys", handler.CreateAPIKey)
	u.GET("/keys", handler.ListAPIKeys)
	u.PUT("/keys/:id", handler.UpdateAPIKey)
	u.DELETE("/keys/:id", handler.DeleteAPIKey)

	// 管理接口
	ad := api.Group("/admin", middleware.JWTAuth(), middleware.AdminRequired())
	ad.GET("/configs", handler.AdminGetConfigs)
	ad.PUT("/configs", handler.AdminUpdateConfigs)
	ad.GET("/users", handler.AdminListUsers)
	ad.PUT("/users/:id", handler.AdminUpdateUser)
	ad.POST("/users/:id/points", handler.AdminAdjustPoints)
	ad.GET("/groups", handler.AdminListGroups)
	ad.POST("/groups", handler.AdminSaveGroup)
	ad.DELETE("/groups/:id", handler.AdminDeleteGroup)
	ad.POST("/channels/:id/test", handler.AdminTestChannel)
	ad.GET("/models", handler.AdminListModels)
	ad.POST("/models", handler.AdminSaveModel)
	ad.DELETE("/models/:id", handler.AdminDeleteModel)
	ad.GET("/invites", handler.AdminListInvites)
	ad.POST("/invites", handler.AdminCreateInvites)
	ad.DELETE("/invites/:id", handler.AdminDeleteInvite)
	ad.GET("/dashboard", handler.AdminDashboard)
	ad.GET("/templates", handler.AdminListTemplates)
	ad.POST("/templates/upload", handler.AdminUploadTemplate)
	ad.PUT("/templates/activate", handler.AdminActivateTemplate)

	// 中继接口（API Key 认证）
	r.POST("/v1/chat/completions", middleware.APIKeyAuth(), relayHandler("openai"))
	r.GET("/v1/models", middleware.APIKeyAuth(), relay.HandleModels)
	r.POST("/anthropic/v1/messages", middleware.APIKeyAuth(), relayHandler("anthropic"))

	registerStatic(r)
	return r
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key, anthropic-version, X-Api-Key, Anthropic-Version")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// registerStatic 静态资源：自定义模板优先，否则使用内嵌前端
func registerStatic(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/v1") || strings.HasPrefix(path, "/anthropic") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "接口不存在"})
			return
		}
		if path == "/" {
			path = "/index.html"
		}
		slug := service.GetConfig("active_template")
		if slug != "" && slug != "default" {
			root := filepath.Join(handler.TemplatesDir(), slug)
			full := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(path, "/")))
			if !strings.HasPrefix(filepath.Clean(full), filepath.Clean(root)) {
				c.String(http.StatusForbidden, "forbidden")
				return
			}
			if st, err := os.Stat(full); err == nil && !st.IsDir() {
				c.File(full)
				return
			}
			index := filepath.Join(root, "index.html")
			if _, err := os.Stat(index); err == nil {
				c.File(index)
				return
			}
		}
		dist, _ := fs.Sub(web.DistFS, "dist")
		rel := strings.TrimPrefix(path, "/")
		if data, err := fs.ReadFile(dist, rel); err == nil {
			c.Data(http.StatusOK, contentType(path), data)
			return
		}
		index, _ := fs.ReadFile(dist, "index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(path, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(path, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(path, ".woff"):
		return "font/woff"
	default:
		return "application/octet-stream"
	}
}
