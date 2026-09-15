package controller

import (
	"github.com/gin-gonic/gin"
)

// writeJSON 写出 JSON 响应并禁用缓存。
func writeJSON(c *gin.Context, status int, v any) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, v)
}

// writeError 写出统一错误契约 {"error": msg}。
func writeError(c *gin.Context, status int, msg string) {
	writeJSON(c, status, gin.H{"error": msg})
}
