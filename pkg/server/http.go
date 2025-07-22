package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	kratoslog "github.com/go-kratos/kratos/v2/log"

	"websocket-server/pkg/config"
)

// NewGinEngine 创建Gin引擎
func NewGinEngine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	return r
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// parseDuration 解析时间字符串
func parseDuration(s string, defaultDuration time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return defaultDuration
}

// HTTPServerWrapper Gin HTTP服务器包装器
type HTTPServerWrapper struct {
	engine *gin.Engine
	server *http.Server
}

// NewHTTPServerWrapper 创建HTTP服务器包装器
func NewHTTPServerWrapper(c *config.Config, logger kratoslog.Logger) *HTTPServerWrapper {
	engine := NewGinEngine()

	// 创建标准HTTP服务器
	server := &http.Server{
		Addr:         c.Server.HTTP.Addr,
		Handler:      engine,
		ReadTimeout:  parseDuration(c.Server.HTTP.Timeout, 30*time.Second),
		WriteTimeout: parseDuration(c.Server.HTTP.Timeout, 30*time.Second),
	}

	return &HTTPServerWrapper{
		engine: engine,
		server: server,
	}
}

// GetEngine 获取Gin引擎
func (w *HTTPServerWrapper) GetEngine() *gin.Engine {
	return w.engine
}

// Start 启动服务器
func (w *HTTPServerWrapper) Start(ctx context.Context) error {
	log.Printf("HTTP server starting on %s", w.server.Addr)
	return w.server.ListenAndServe()
}

// Stop 停止服务器
func (w *HTTPServerWrapper) Stop(ctx context.Context) error {
	log.Println("HTTP server stopping")
	return w.server.Shutdown(ctx)
}
