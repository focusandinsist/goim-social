package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/apps/logic-service/internal/converter"
	"goim-social/apps/logic-service/internal/service"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    log,
	}
}

// RegisterRoutes 注册HTTP路由,Logic服务是内部服务，只提供健康检查和测试接口
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/logic")
	{
		api.POST("/health", h.HealthCheck) // 健康检查
		api.POST("/route", h.RouteMessage) // 消息路由测试
	}
}
