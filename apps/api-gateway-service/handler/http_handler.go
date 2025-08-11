package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/apps/api-gateway-service/converter"
	"goim-social/apps/api-gateway-service/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP协议处理器
type HTTPHandler struct {
	svc       *service.Service
	converter *converter.Converter
	log       logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		log:       log,
	}
}

// RegisterRoutes 注册HTTP路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	// 动态路由，所有符合 /api/v1/{service-name}/* 格式的请求都会被动态路由
	api := r.Group("/api/v1")
	{
		api.Any("/*path", h.DynamicRoute) // 动态路由处理器
	}
}

// DynamicRoute 动态路由处理器
func (h *HTTPHandler) DynamicRoute(c *gin.Context) {
	ctx := c.Request.Context()

	// 从认证中间件获取用户ID（如果有的话）
	if userID, exists := c.Get("userID"); exists {
		if uid, ok := userID.(int64); ok {
			ctx = tracecontext.WithUserID(ctx, uid)
			c.Request = c.Request.WithContext(ctx)
		}
	}

	// 记录请求日志
	h.log.Info(ctx, "Dynamic route request",
		logger.F("method", c.Request.Method),
		logger.F("path", c.Request.URL.Path),
		logger.F("query", c.Request.URL.RawQuery))

	// 调用service层的动态路由功能
	err := h.svc.ProxyRequest(c.Writer, c.Request)
	if err != nil {
		h.log.Error(ctx, "Dynamic route failed", logger.F("error", err.Error()))
		// 错误已经在service层处理了，这里不需要再次响应
	}
}
