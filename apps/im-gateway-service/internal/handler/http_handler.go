package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/apps/im-gateway-service/internal/converter"
	"goim-social/apps/im-gateway-service/internal/service"
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
	api := r.Group("/api/v1/connect")
	{
		api.POST("/online_status", h.OnlineStatus) // 查询在线状态
	}
}
