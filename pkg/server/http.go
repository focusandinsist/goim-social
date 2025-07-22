package server

import (
	"context"
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

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	return r
}

// parseDuration 解析时间字符串
func parseDuration(s string, defaultDuration time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return defaultDuration
}

// HTTPServer HTTP服务器接口
type HTTPServer interface {
	GetEngine() *gin.Engine
	RegisterRoutes(registerFunc func(*gin.Engine))
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HTTPServerWrapper Gin HTTP服务器包装器
type HTTPServerWrapper struct {
	engine *gin.Engine
	server *http.Server
	logger kratoslog.Logger
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
		logger: logger,
	}
}

// GetEngine 获取Gin引擎
func (w *HTTPServerWrapper) GetEngine() *gin.Engine {
	return w.engine
}

// RegisterRoutes 注册路由
func (w *HTTPServerWrapper) RegisterRoutes(registerFunc func(*gin.Engine)) {
	registerFunc(w.engine)
}

// Start 启动服务器
func (w *HTTPServerWrapper) Start(ctx context.Context) error {
	w.logger.Log(kratoslog.LevelInfo, "msg", "HTTP server starting", "addr", w.server.Addr)
	return w.server.ListenAndServe()
}

// Stop 停止服务器
func (w *HTTPServerWrapper) Stop(ctx context.Context) error {
	w.logger.Log(kratoslog.LevelInfo, "msg", "HTTP server stopping")
	return w.server.Shutdown(ctx)
}
