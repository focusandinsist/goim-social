package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/apps/user-service/internal/converter"
	"goim-social/apps/user-service/internal/service"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	service   *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(service *service.Service, logger logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		service:   service,
		converter: converter.NewConverter(),
		logger:    logger,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/users")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		api.POST("/get", h.GetUserByID)
	}
}
