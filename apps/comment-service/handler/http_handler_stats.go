package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// GetCommentStats 获取评论统计
func (h *HTTPHandler) GetCommentStats(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetCommentStatsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comment stats request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetCommentStatsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换对象类型
	objectType := h.converter.ObjectTypeFromProto(req.ObjectType)

	stats, err := h.svc.GetCommentStats(ctx, req.ObjectId, objectType)

	var res *rest.GetCommentStatsResponse
	if err != nil {
		h.logger.Error(ctx, "Failed to get comment stats",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectId))
		res = h.converter.BuildGetCommentStatsResponse(false, err.Error(), nil)
	} else {
		res = h.converter.BuildGetCommentStatsResponse(true, "获取成功", stats)
	}

	httpx.WriteObject(c, res, err)
}

// GetBatchCommentStats 批量获取评论统计
func (h *HTTPHandler) GetBatchCommentStats(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetBatchCommentStatsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get batch comment stats request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetBatchCommentStatsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换对象类型
	objectType := h.converter.ObjectTypeFromProto(req.ObjectType)

	statsList, err := h.svc.GetBatchCommentStats(ctx, req.ObjectIds, objectType)

	var res *rest.GetBatchCommentStatsResponse
	if err != nil {
		h.logger.Error(ctx, "Failed to get batch comment stats",
			logger.F("error", err.Error()),
			logger.F("objectIDs", req.ObjectIds))
		res = h.converter.BuildGetBatchCommentStatsResponse(false, err.Error(), nil)
	} else {
		res = h.converter.BuildGetBatchCommentStatsResponse(true, "获取成功", statsList)
	}

	httpx.WriteObject(c, res, err)
}
