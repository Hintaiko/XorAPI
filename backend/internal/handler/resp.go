package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": data})
}

func Fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "msg": msg})
}

func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return false
	}
	return true
}
