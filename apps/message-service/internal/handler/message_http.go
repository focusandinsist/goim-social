package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// SendMessage 发送消息
func (h *HTTPHandler) SendMessage(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.SendMessageRequest
		resp interface{}
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid send message request", logger.F("error", err.Error()))
		resp = h.converter.BuildSendMessageResponse(0, "")
		httpx.WriteObject(c, resp, err)
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
	resp = h.converter.BuildSendMessageResponse(messageID, ackID)
	if err != nil {
		h.logger.Error(ctx, "Send message failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, resp, err)
}

// GetHistory 获取历史消息
func (h *HTTPHandler) GetHistory(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetHistoryRequest
		resp *rest.GetHistoryResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get history request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetHistoryResponse(req.Page, req.Size)
		httpx.WriteObject(c, resp, err)
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
		resp = h.converter.BuildErrorGetHistoryResponse(req.Page, req.Size)
	} else {
		resp = h.converter.BuildGetHistoryResponse(msgs, total, req.Page, req.Size)
	}

	httpx.WriteObject(c, resp, err)
}

// GetUnreadMessages 获取未读消息
func (h *HTTPHandler) GetUnreadMessages(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetUnreadMessagesRequest
		resp *rest.GetUnreadMessagesResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get unread messages request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetUnreadMessagesResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	msgs, err := h.service.GetUnreadMessages(ctx, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "Get unread messages failed", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetUnreadMessagesResponse(err.Error())
	} else {
		resp = h.converter.BuildSuccessGetUnreadMessagesResponse(msgs)
	}

	httpx.WriteObject(c, resp, err)
}

// MarkMessagesRead 标记消息已读
func (h *HTTPHandler) MarkMessagesRead(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.MarkMessagesReadRequest
		resp *rest.MarkMessagesReadResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid mark messages read request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorMarkMessagesReadResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	failedIDs, err := h.service.MarkMessagesAsRead(ctx, req.UserId, req.MessageIds)
	if err != nil {
		h.logger.Error(ctx, "Mark messages read failed", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorMarkMessagesReadResponse(err.Error())
	} else {
		resp = h.converter.BuildSuccessMarkMessagesReadResponse(failedIDs)
	}

	httpx.WriteObject(c, resp, err)
}
