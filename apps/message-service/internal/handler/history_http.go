package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// RecordUserAction 记录用户行为
func (h *HTTPHandler) RecordUserAction(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.RecordUserActionRequest
		resp *rest.RecordUserActionResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid record user action request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorRecordUserActionResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	// 记录用户行为
	err = h.service.RecordUserAction(ctx, req.UserId, req.ActionType.String(), req.ObjectType.String(), req.ObjectId, req.ObjectTitle, req.ObjectUrl, req.Metadata, req.IpAddress, req.UserAgent, req.DeviceInfo, req.Location, req.Duration)
	if err != nil {
		h.logger.Error(ctx, "Record user action failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("actionType", req.ActionType.String()))
		resp = h.converter.BuildErrorRecordUserActionResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Record user action successful",
			logger.F("userID", req.UserId),
			logger.F("actionType", req.ActionType.String()))
		resp = h.converter.BuildSuccessRecordUserActionResponse()
	}

	httpx.WriteObject(c, resp, err)
}

// BatchRecordAction 批量记录用户行为
func (h *HTTPHandler) BatchRecordAction(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.BatchRecordUserActionRequest
		resp *rest.BatchRecordUserActionResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid batch record user action request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorBatchRecordUserActionResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 批量记录用户行为
	records := h.converter.ConvertToHistoryRecords(req.Actions)
	successCount, failedCount, errors := h.service.BatchRecordUserAction(ctx, records)
	resp = h.converter.BuildBatchRecordUserActionResponse(successCount, failedCount, errors)
	httpx.WriteObject(c, resp, nil)
}

// GetUserHistory 获取用户历史记录
func (h *HTTPHandler) GetUserHistory(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetUserHistoryRequest
		resp *rest.GetUserHistoryResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user history request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetUserHistoryResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	// 获取用户历史记录
	// 需要将字符串时间转换为time.Time，这里简化处理
	startTime, _ := time.Parse(time.RFC3339, req.StartTime)
	endTime, _ := time.Parse(time.RFC3339, req.EndTime)
	actions, total, err := h.service.GetUserHistory(ctx, req.UserId, req.ActionType.String(), req.ObjectType.String(), startTime, endTime, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Get user history failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		resp = h.converter.BuildErrorGetUserHistoryResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get user history successful",
			logger.F("userID", req.UserId),
			logger.F("total", total))
		resp = h.converter.BuildGetUserHistoryResponse(actions, total, req.Page, req.PageSize)
	}

	httpx.WriteObject(c, resp, err)
}

// DeleteUserHistory 删除用户历史记录
func (h *HTTPHandler) DeleteUserHistory(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.DeleteHistoryRequest
		resp *rest.DeleteHistoryResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete user history request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorDeleteHistoryResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	// 删除用户历史记录
	// 将int64数组转换为string数组
	recordIDs := make([]string, len(req.RecordIds))
	for i, id := range req.RecordIds {
		recordIDs[i] = fmt.Sprintf("%d", id)
	}
	deletedCount, err := h.service.DeleteUserHistory(ctx, req.UserId, recordIDs)
	if err != nil {
		h.logger.Error(ctx, "Delete user history failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		resp = h.converter.BuildErrorDeleteHistoryResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Delete user history successful",
			logger.F("userID", req.UserId),
			logger.F("deletedCount", deletedCount))
		resp = h.converter.BuildDeleteHistoryResponse(deletedCount)
	}

	httpx.WriteObject(c, resp, err)
}

// GetUserActionStats 获取用户行为统计
func (h *HTTPHandler) GetUserActionStats(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetUserActionStatsRequest
		resp *rest.GetUserActionStatsResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user action stats request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetUserActionStatsResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	// 获取用户行为统计
	startTime, _ := time.Parse(time.RFC3339, req.StartTime)
	endTime, _ := time.Parse(time.RFC3339, req.EndTime)
	stats, err := h.service.GetUserActionStats(ctx, req.UserId, req.ActionType.String(), startTime, endTime, req.GroupBy)
	if err != nil {
		h.logger.Error(ctx, "Get user action stats failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		resp = h.converter.BuildErrorGetUserActionStatsResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get user action stats successful",
			logger.F("userID", req.UserId))
		resp = h.converter.BuildGetUserActionStatsResponse(stats)
	}

	httpx.WriteObject(c, resp, err)
}
