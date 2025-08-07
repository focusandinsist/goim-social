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

// ==================== 历史记录相关处理函数 ====================

// RecordUserAction 记录用户行为
func (h *HTTPHandler) RecordUserAction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.RecordUserActionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid record user action request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorRecordUserActionResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 从context获取用户信息
	userID := tracecontext.GetUserID(ctx)
	if userID == 0 {
		userID = req.UserId
	}

	// 调用service层记录用户行为
	err := h.service.RecordUserAction(
		ctx,
		userID,
		h.converter.ActionTypeToString(req.ActionType),
		h.converter.ObjectTypeToString(req.ObjectType),
		req.ObjectId,
		req.ObjectTitle,
		req.ObjectUrl,
		req.Metadata,
		req.IpAddress,
		req.UserAgent,
		req.DeviceInfo,
		req.Location,
		req.Duration,
	)

	var res *rest.RecordUserActionResponse
	if err != nil {
		h.logger.Error(ctx, "Record user action failed", logger.F("error", err.Error()))
		res = h.converter.BuildErrorRecordUserActionResponse(err.Error())
	} else {
		res = h.converter.BuildSuccessRecordUserActionResponse()
	}

	httpx.WriteObject(c, res, err)
}

// BatchRecordAction 批量记录用户行为
func (h *HTTPHandler) BatchRecordAction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.BatchRecordUserActionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid batch record user action request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorBatchRecordUserActionResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换请求为模型
	records := h.converter.ConvertToHistoryRecords(req.Actions)

	// 调用service层批量记录用户行为
	successCount, failedCount, errors := h.service.BatchRecordUserAction(ctx, records)

	res := h.converter.BuildBatchRecordUserActionResponse(successCount, failedCount, errors)
	httpx.WriteObject(c, res, nil)
}

// GetUserHistory 获取用户历史记录
func (h *HTTPHandler) GetUserHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user history request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetUserHistoryResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 从context获取用户信息
	userID := tracecontext.GetUserID(ctx)
	if userID == 0 {
		userID = req.UserId
	}

	// 解析时间参数
	startTime, endTime, err := h.converter.ParseTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		h.logger.Error(ctx, "Invalid time range", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetUserHistoryResponse("Invalid time range")
		httpx.WriteObject(c, res, err)
		return
	}

	// 调用service层获取用户历史记录
	records, total, err := h.service.GetUserHistory(
		ctx,
		userID,
		h.converter.ActionTypeToString(req.ActionType),
		h.converter.ObjectTypeToString(req.ObjectType),
		startTime,
		endTime,
		req.Page,
		req.PageSize,
	)

	var res *rest.GetUserHistoryResponse
	if err != nil {
		h.logger.Error(ctx, "Get user history failed", logger.F("error", err.Error()))
		res = h.converter.BuildErrorGetUserHistoryResponse(err.Error())
	} else {
		res = h.converter.BuildGetUserHistoryResponse(records, total, req.Page, req.PageSize)
	}

	httpx.WriteObject(c, res, err)
}

// DeleteUserHistory 删除用户历史记录
func (h *HTTPHandler) DeleteUserHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete user history request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorDeleteHistoryResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 从context获取用户信息
	userID := tracecontext.GetUserID(ctx)
	if userID == 0 {
		userID = req.UserId
	}

	// 转换记录ID为字符串数组
	recordIDs := h.converter.ConvertRecordIDs(req.RecordIds)

	// 调用service层删除用户历史记录
	deletedCount, err := h.service.DeleteUserHistory(ctx, userID, recordIDs)

	var res *rest.DeleteHistoryResponse
	if err != nil {
		h.logger.Error(ctx, "Delete user history failed", logger.F("error", err.Error()))
		res = h.converter.BuildErrorDeleteHistoryResponse(err.Error())
	} else {
		res = h.converter.BuildDeleteHistoryResponse(deletedCount)
	}

	httpx.WriteObject(c, res, err)
}

// GetUserActionStats 获取用户行为统计
func (h *HTTPHandler) GetUserActionStats(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserActionStatsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user action stats request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetUserActionStatsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 从context获取用户信息
	userID := tracecontext.GetUserID(ctx)
	if userID == 0 {
		userID = req.UserId
	}

	// 解析时间参数
	startTime, endTime, err := h.converter.ParseTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		h.logger.Error(ctx, "Invalid time range", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetUserActionStatsResponse("Invalid time range")
		httpx.WriteObject(c, res, err)
		return
	}

	// 调用service层获取用户行为统计
	stats, err := h.service.GetUserActionStats(
		ctx,
		userID,
		h.converter.ActionTypeToString(req.ActionType),
		startTime,
		endTime,
		req.GroupBy,
	)

	var res *rest.GetUserActionStatsResponse
	if err != nil {
		h.logger.Error(ctx, "Get user action stats failed", logger.F("error", err.Error()))
		res = h.converter.BuildErrorGetUserActionStatsResponse(err.Error())
	} else {
		res = h.converter.BuildGetUserActionStatsResponse(stats)
	}

	httpx.WriteObject(c, res, err)
}
