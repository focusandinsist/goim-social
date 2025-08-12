package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// DoInteraction 执行互动（点赞/收藏/分享等）
func (h *HTTPHandler) DoInteraction(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.DoInteractionRequest
		resp *rest.DoInteractionResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid do interaction request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorDoInteractionResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)
	ctx = tracecontext.WithContentID(ctx, req.TargetId)

	interaction, err := h.svc.DoInteraction(ctx, req.UserId, req.TargetId, req.TargetType.String(), req.InteractionType.String(), req.Metadata)
	if err != nil {
		h.logger.Error(ctx, "Do interaction failed", logger.F("error", err.Error()), logger.F("targetID", req.TargetId), logger.F("type", req.InteractionType.String()))
		resp = h.converter.BuildErrorDoInteractionResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Do interaction successful", logger.F("targetID", req.TargetId), logger.F("type", req.InteractionType.String()))
		resp = h.converter.BuildDoInteractionResponse(true, "互动操作成功", interaction)
	}

	httpx.WriteObject(c, resp, err)
}

// UndoInteraction 取消互动
func (h *HTTPHandler) UndoInteraction(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.UndoInteractionRequest
		resp *rest.UndoInteractionResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid undo interaction request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorUndoInteractionResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)
	ctx = tracecontext.WithContentID(ctx, req.TargetId)

	err = h.svc.UndoInteraction(ctx, req.UserId, req.TargetId, req.TargetType.String(), req.InteractionType.String())
	if err != nil {
		h.logger.Error(ctx, "Undo interaction failed", logger.F("error", err.Error()), logger.F("targetID", req.TargetId), logger.F("type", req.InteractionType.String()))
		resp = h.converter.BuildErrorUndoInteractionResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Undo interaction successful", logger.F("targetID", req.TargetId), logger.F("type", req.InteractionType.String()))
		resp = h.converter.BuildUndoInteractionResponse(true, "取消互动成功")
	}

	httpx.WriteObject(c, resp, err)
}

// CheckInteraction 检查互动状态
func (h *HTTPHandler) CheckInteraction(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.CheckInteractionRequest
		resp *rest.CheckInteractionResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid check interaction request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorCheckInteractionResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)
	ctx = tracecontext.WithContentID(ctx, req.TargetId)

	hasInteraction, interaction, err := h.svc.CheckInteraction(ctx, req.UserId, req.TargetId, req.TargetType.String(), req.InteractionType.String())
	if err != nil {
		h.logger.Error(ctx, "Check interaction failed", logger.F("error", err.Error()), logger.F("targetID", req.TargetId), logger.F("type", req.InteractionType.String()))
		resp = h.converter.BuildErrorCheckInteractionResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Check interaction successful", logger.F("targetID", req.TargetId), logger.F("type", req.InteractionType.String()), logger.F("hasInteraction", hasInteraction))
		resp = h.converter.BuildCheckInteractionResponse(true, "检查互动状态成功", hasInteraction, interaction)
	}

	httpx.WriteObject(c, resp, err)
}

// GetInteractionStats 获取互动统计
func (h *HTTPHandler) GetInteractionStats(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetInteractionStatsRequest
		resp *rest.GetInteractionStatsResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get interaction stats request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetInteractionStatsResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithContentID(ctx, req.TargetId)

	stats, err := h.svc.GetInteractionStats(ctx, req.TargetId, req.TargetType.String())
	if err != nil {
		h.logger.Error(ctx, "Get interaction stats failed", logger.F("error", err.Error()), logger.F("targetID", req.TargetId))
		resp = h.converter.BuildErrorGetInteractionStatsResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get interaction stats successful", logger.F("targetID", req.TargetId))
		resp = h.converter.BuildGetInteractionStatsResponse(true, "获取互动统计成功", stats)
	}

	httpx.WriteObject(c, resp, err)
}
