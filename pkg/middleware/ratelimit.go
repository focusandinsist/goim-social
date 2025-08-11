package middleware

import (
	"github.com/gin-gonic/gin"
)

// RateLimit 简单的限流中间件
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO 这里可以实现基于IP或用户的限流逻辑
		// 暂时跳过实现
		c.Next()
	}
}
