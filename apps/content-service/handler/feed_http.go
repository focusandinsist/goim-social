package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// GetContentDetail 获取内容详情（包含评论和互动）
func (h *HTTPHandler) GetContentDetail(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetContentDetailRequest
		resp *rest.GetContentDetailResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content detail request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetContentDetailResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithContentID(ctx, req.ContentId)
	if req.UserId > 0 {
		ctx = tracecontext.WithUserID(ctx, req.UserId)
	}

	detail, err := h.svc.GetContentDetail(ctx, req.ContentId, req.UserId, req.CommentLimit)
	if err != nil {
		h.logger.Error(ctx, "Get content detail failed", logger.F("error", err.Error()), logger.F("contentID", req.ContentId))
		resp = h.converter.BuildErrorGetContentDetailResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get content detail successful", logger.F("contentID", req.ContentId))
		resp = h.converter.BuildGetContentDetailResponse(true, "获取内容详情成功", detail)
	}

	httpx.WriteObject(c, resp, err)
}

// GetContentFeed 获取内容流
func (h *HTTPHandler) GetContentFeed(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetContentFeedRequest
		resp *rest.GetContentFeedResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content feed request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetContentFeedResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	feed, total, err := h.svc.GetContentFeed(ctx, req.UserId, req.ContentType, req.SortBy, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Get content feed failed", logger.F("error", err.Error()), logger.F("userID", req.UserId))
		resp = h.converter.BuildErrorGetContentFeedResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get content feed successful", logger.F("userID", req.UserId), logger.F("count", len(feed)))
		resp = h.converter.BuildGetContentFeedResponse(true, "获取内容流成功", feed, total)
	}

	httpx.WriteObject(c, resp, err)
}

// GetTrendingContent 获取热门内容
func (h *HTTPHandler) GetTrendingContent(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetTrendingContentRequest
		resp *rest.GetTrendingContentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get trending content request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetTrendingContentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	trending, err := h.svc.GetTrendingContent(ctx, req.TimeRange, req.ContentType, req.Limit)
	if err != nil {
		h.logger.Error(ctx, "Get trending content failed", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetTrendingContentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get trending content successful", logger.F("count", len(trending)))
		resp = h.converter.BuildGetTrendingContentResponse(true, "获取热门内容成功", trending)
	}

	httpx.WriteObject(c, resp, err)
}
