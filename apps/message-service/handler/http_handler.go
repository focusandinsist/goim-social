package handler

import (
	"github.com/gin-gonic/gin"

	"websocket-server/api/rest"
	"websocket-server/apps/message-service/service"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/utils"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	service *service.Service
	logger  logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(service *service.Service, logger logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 注册HTTP路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/messages")
	{
		api.POST("/history", h.GetHistory)         // 获取历史消息
		api.POST("/unread", h.GetUnreadMessages)   // 获取未读消息
		api.POST("/mark-read", h.MarkMessagesRead) // 标记消息已读
		api.POST("/send", h.SendMessage)           // 特殊场景下的短连接消息，如测试、某些网络环境下的备用通道
	}
}

// SendMessage 发送消息
func (h *HTTPHandler) SendMessage(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.SendMessageRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid send message request", logger.F("error", err.Error()))
		res := &rest.SendMessageResponse{
			MessageId: 0,
			AckId:     "",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 调用service层发送消息
	messageID, ackID, err := h.service.SendMessage(ctx, &req)
	res := &rest.SendMessageResponse{
		MessageId: messageID,
		AckId:     ackID,
	}
	if err != nil {
		h.logger.Error(ctx, "Send message failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetHistory 获取历史消息
func (h *HTTPHandler) GetHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get history request", logger.F("error", err.Error()))
		res := &rest.GetHistoryResponse{
			Messages: []*rest.WSMessage{},
			Total:    0,
			Page:     req.Page,
			Size:     req.Size,
		}
		utils.WriteObject(c, res, err)
		return
	}

	msgs, total, err := h.service.GetMessageHistory(ctx, req.UserId, req.GroupId, int(req.Page), int(req.Size))
	if err != nil {
		h.logger.Error(ctx, "Get history failed", logger.F("error", err.Error()))
		utils.WriteObject(c, nil, err)
		return
	}

	// 转换为proto消息格式
	var wsMessages []*rest.WSMessage
	for _, msg := range msgs {
		wsMessages = append(wsMessages, &rest.WSMessage{
			MessageId:   msg.MessageID,
			From:        msg.From,
			To:          msg.To,
			GroupId:     msg.GroupID,
			Content:     msg.Content,
			Timestamp:   msg.Timestamp,
			MessageType: int32(msg.MessageType),
			AckId:       msg.AckID,
		})
	}

	res := &rest.GetHistoryResponse{
		Messages: wsMessages,
		Total:    int32(total),
		Page:     req.Page,
		Size:     req.Size,
	}

	utils.WriteObject(c, res, err)
}

// GetUnreadMessages 获取未读消息
func (h *HTTPHandler) GetUnreadMessages(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUnreadMessagesRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get unread messages request", logger.F("error", err.Error()))
		res := &rest.GetUnreadMessagesResponse{
			Success:  false,
			Message:  "Invalid request format",
			Messages: []*rest.WSMessage{},
			Total:    0,
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 调用service层获取未读消息
	messages, err := h.service.GetUnreadMessages(ctx, req.UserId)
	
	// 转换为proto消息格式
	var wsMessages []*rest.WSMessage
	if err == nil {
		for _, msg := range messages {
			wsMessages = append(wsMessages, &rest.WSMessage{
				MessageId:   msg.MessageID,
				From:        msg.From,
				To:          msg.To,
				GroupId:     msg.GroupID,
				Content:     msg.Content,
				Timestamp:   msg.Timestamp,
				MessageType: int32(msg.MessageType),
				AckId:       msg.AckID,
			})
		}
	}

	res := &rest.GetUnreadMessagesResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取未读消息成功"
		}(),
		Messages: wsMessages,
		Total:    int32(len(wsMessages)),
	}
	if err != nil {
		h.logger.Error(ctx, "Get unread messages failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// MarkMessagesRead 标记消息已读
func (h *HTTPHandler) MarkMessagesRead(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.MarkMessagesReadRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid mark messages read request", logger.F("error", err.Error()))
		res := &rest.MarkMessagesReadResponse{
			Success:   false,
			Message:   "Invalid request format",
			FailedIds: []int64{},
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 调用service层标记消息已读
	failedIDs, err := h.service.MarkMessagesAsRead(ctx, req.UserId, req.MessageIds)

	res := &rest.MarkMessagesReadResponse{
		Success: err == nil && len(failedIDs) == 0,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			if len(failedIDs) > 0 {
				return "部分消息标记已读失败"
			}
			return "标记已读成功"
		}(),
		FailedIds: failedIDs,
	}
	if err != nil {
		h.logger.Error(ctx, "Mark messages read failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}
