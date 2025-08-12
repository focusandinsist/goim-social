package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/apps/message-service/internal/converter"
	"goim-social/apps/message-service/internal/service"
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

// RegisterRoutes 注册HTTP路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	// 消息相关路由
	messages := r.Group("/api/v1/messages")
	{
		messages.POST("/history", h.GetHistory)         // 获取历史消息
		messages.POST("/unread", h.GetUnreadMessages)   // 获取未读消息
		messages.POST("/mark-read", h.MarkMessagesRead) // 标记消息已读
		messages.POST("/send", h.SendMessage)           // 特殊场景下的短连接消息，如测试、某些网络环境下的备用通道
	}

	// 历史记录相关路由
	history := r.Group("/api/v1/history")
	{
		history.POST("/record", h.RecordUserAction)        // 记录用户行为
		history.POST("/batch-record", h.BatchRecordAction) // 批量记录用户行为
		history.POST("/user", h.GetUserHistory)            // 获取用户历史记录
		history.POST("/delete", h.DeleteUserHistory)       // 删除用户历史记录
		history.POST("/stats", h.GetUserActionStats)       // 获取用户行为统计
	}
}
