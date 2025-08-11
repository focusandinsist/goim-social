package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goim-social/pkg/logger"
)

// Recovery 错误恢复中间件
func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error(c.Request.Context(), "Panic recovered",
					logger.F("error", err),
					logger.F("method", c.Request.Method),
					logger.F("path", c.Request.URL.Path))
				
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "internal server error",
					"error":   "unexpected error occurred",
				})
				c.Abort()
			}
		}()
		
		c.Next()
	}
}
