package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"xorapi/internal/model"
	"xorapi/internal/service"
	"xorapi/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Claims struct {
	UserID uint   `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(u *model.User) (string, error) {
	claims := Claims{
		UserID: u.ID, Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(store.Cfg.Server.JWTSecret))
}

func parseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return []byte(store.Cfg.Server.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

// JWTAuth 管理后台 / 用户中心认证
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录"})
			return
		}
		claims, err := parseToken(strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "登录已过期"})
			return
		}
		var user model.User
		if err := store.DB.First(&user, claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户不存在"})
			return
		}
		if user.Status != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "账号已被禁用"})
			return
		}
		c.Set("user", &user)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil || u.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "需要管理员权限"})
			return
		}
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *model.User {
	if v, ok := c.Get("user"); ok {
		if u, ok := v.(*model.User); ok {
			return u
		}
	}
	return nil
}

// APIKeyAuth 中继接口认证（sk-xxx）
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		key := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if strings.HasPrefix(auth, "Bearer sk-") == false || len(key) < 20 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "无效的 API Key", "type": "invalid_request_error"}})
			return
		}
		hash := service.SHA256Hex(key)
		var ak model.APIKey
		if err := store.DB.Where("key_hash = ?", hash).First(&ak).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "API Key 不存在", "type": "invalid_request_error"}})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "数据库错误", "type": "server_error"}})
			return
		}
		if ak.Status != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "API Key 已被禁用", "type": "invalid_request_error"}})
			return
		}
		var user model.User
		if err := store.DB.First(&user, ak.UserID).Error; err != nil || user.Status != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "账号不可用", "type": "invalid_request_error"}})
			return
		}
		if user.Points <= 0 {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{"error": gin.H{"message": "点数余额不足，请充值或签到", "type": "insufficient_quota"}})
			return
		}
		ip := clientIP(c)
		if !matchWhitelist(ak.IPWhitelist, ip) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "IP 不在白名单内: " + ip, "type": "invalid_request_error"}})
			return
		}
		if ak.DailyLimit > 0 {
			if n := store.IncrDaily("kd", ak.ID, time.Now()); n > ak.DailyLimit {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"message": "已达今日调用上限", "type": "rate_limit_error"}})
				return
			}
		}
		rpm := int64(service.GetConfigInt("relay_rpm", 60))
		if !store.AllowMinute(fmt.Sprintf("k%d", ak.ID), rpm) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"message": "请求过于频繁", "type": "rate_limit_error"}})
			return
		}
		c.Set("apikey", &ak)
		c.Set("user", &user)
		c.Set("client_ip", ip)
		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	if xf := c.GetHeader("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}
	return c.ClientIP()
}

func matchWhitelist(whitelistJSON, ip string) bool {
	if strings.TrimSpace(whitelistJSON) == "" || whitelistJSON == "[]" || whitelistJSON == "null" {
		return true
	}
	var list []string
	if err := json.Unmarshal([]byte(whitelistJSON), &list); err != nil {
		return true
	}
	for _, item := range list {
		if strings.TrimSpace(item) == ip {
			return true
		}
	}
	return false
}
