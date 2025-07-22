package handler

import (
	"websocket-server/apps/connect-service/service"
	"websocket-server/pkg/logger"
)

// Handler 连接服务处理器
type Handler struct {
	service *service.Service
	logger  logger.Logger
}

// NewHandler 创建新的处理器实例
func NewHandler(service *service.Service, logger logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}
