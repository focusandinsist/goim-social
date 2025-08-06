package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/message-service/converter"
	"goim-social/apps/message-service/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
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
		res := h.converter.BuildSendMessageResponse(0, "")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	// SendMessageRequest没有From字段，这里暂时使用To字段
	ctx = tracecontext.WithUserID(ctx, req.To)
	if req.GroupId > 0 {
		ctx = tracecontext.WithGroupID(ctx, req.GroupId)
	}

	// 调用service层发送消息
	messageID, ackID, err := h.service.SendMessage(ctx, &req)
	res := h.converter.BuildSendMessageResponse(messageID, ackID)
	if err != nil {
		h.logger.Error(ctx, "Send message failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// GetHistory 获取历史消息
func (h *HTTPHandler) GetHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get history request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetHistoryResponse(req.Page, req.Size)
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)
	if req.GroupId > 0 {
		ctx = tracecontext.WithGroupID(ctx, req.GroupId)
	}

	msgs, total, err := h.service.GetMessageHistory(ctx, req.UserId, req.GroupId, int(req.Page), int(req.Size))
	if err != nil {
		h.logger.Error(ctx, "Get history failed", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetHistoryResponse(req.Page, req.Size)
		httpx.WriteObject(c, res, err)
		return
	}

	res := h.converter.BuildGetHistoryResponse(msgs, total, req.Page, req.Size)
	httpx.WriteObject(c, res, err)
}

// GetUnreadMessages 获取未读消息
func (h *HTTPHandler) GetUnreadMessages(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUnreadMessagesRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get unread messages request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetUnreadMessagesResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	// 调用service层获取未读消息
	messages, err := h.service.GetUnreadMessages(ctx, req.UserId)

	var res *rest.GetUnreadMessagesResponse
	if err != nil {
		h.logger.Error(ctx, "Get unread messages failed", logger.F("error", err.Error()))
		res = h.converter.BuildErrorGetUnreadMessagesResponse(err.Error())
	} else {
		res = h.converter.BuildSuccessGetUnreadMessagesResponse(messages)
	}

	httpx.WriteObject(c, res, err)
}

// MarkMessagesRead 标记消息已读
func (h *HTTPHandler) MarkMessagesRead(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.MarkMessagesReadRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid mark messages read request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorMarkMessagesReadResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 调用service层标记消息已读
	failedIDs, err := h.service.MarkMessagesAsRead(ctx, req.UserId, req.MessageIds)

	var res *rest.MarkMessagesReadResponse
	if err != nil {
		h.logger.Error(ctx, "Mark messages read failed", logger.F("error", err.Error()))
		res = h.converter.BuildErrorMarkMessagesReadResponse(err.Error())
	} else {
		res = h.converter.BuildSuccessMarkMessagesReadResponse(failedIDs)
	}

	httpx.WriteObject(c, res, err)
}
