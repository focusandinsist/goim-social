package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// CreateContent 创建内容
func (h *HTTPHandler) CreateContent(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.CreateContentRequest
		resp *rest.CreateContentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create content request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorCreateContentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.AuthorId)

	// 转换媒体文件
	mediaFiles := h.converter.MediaFileProtoToModels(req.MediaFiles)

	// 转换内容类型
	contentType := h.converter.ContentTypeFromProto(req.Type)

	content, err := h.svc.CreateContent(
		ctx,
		req.AuthorId,
		req.Title,
		req.Content,
		contentType,
		mediaFiles,
		req.TagIds,
		req.TopicIds,
		req.TemplateData,
		req.SaveAsDraft,
	)
	if err != nil {
		h.logger.Error(ctx, "Create content failed", logger.F("error", err.Error()), logger.F("authorID", req.AuthorId))
		resp = h.converter.BuildErrorCreateContentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Create content successful", logger.F("contentID", content.ID), logger.F("authorID", req.AuthorId))
		resp = h.converter.BuildCreateContentResponse(true, "创建内容成功", content)
	}

	httpx.WriteObject(c, resp, err)
}

// UpdateContent 更新内容
func (h *HTTPHandler) UpdateContent(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.UpdateContentRequest
		resp *rest.UpdateContentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid update content request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorUpdateContentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.AuthorId)
	ctx = tracecontext.WithContentID(ctx, req.ContentId)

	// 转换媒体文件
	mediaFiles := h.converter.MediaFileProtoToModels(req.MediaFiles)

	// 转换内容类型
	contentType := h.converter.ContentTypeFromProto(req.Type)

	content, err := h.svc.UpdateContent(
		ctx,
		req.ContentId,
		req.AuthorId,
		req.Title,
		req.Content,
		contentType,
		mediaFiles,
		req.TagIds,
		req.TopicIds,
		req.TemplateData,
	)
	if err != nil {
		h.logger.Error(ctx, "Update content failed", logger.F("error", err.Error()), logger.F("contentID", req.ContentId))
		resp = h.converter.BuildErrorUpdateContentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Update content successful", logger.F("contentID", content.ID))
		resp = h.converter.BuildUpdateContentResponse(true, "更新内容成功", content)
	}

	httpx.WriteObject(c, resp, err)
}

// GetContent 获取内容详情
func (h *HTTPHandler) GetContent(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetContentRequest
		resp *rest.GetContentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetContentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithContentID(ctx, req.ContentId)
	if req.UserId > 0 {
		ctx = tracecontext.WithUserID(ctx, req.UserId)
	}

	content, err := h.svc.GetContent(ctx, req.ContentId, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "Get content failed", logger.F("error", err.Error()), logger.F("contentID", req.ContentId))
		resp = h.converter.BuildErrorGetContentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get content successful", logger.F("contentID", content.ID))
		resp = h.converter.BuildGetContentResponse(true, "获取内容成功", content)
	}

	httpx.WriteObject(c, resp, err)
}

// DeleteContent 删除内容
func (h *HTTPHandler) DeleteContent(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.DeleteContentRequest
		resp *rest.DeleteContentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete content request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorDeleteContentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.AuthorId)
	ctx = tracecontext.WithContentID(ctx, req.ContentId)

	err = h.svc.DeleteContent(ctx, req.ContentId, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Delete content failed", logger.F("error", err.Error()), logger.F("contentID", req.ContentId))
		resp = h.converter.BuildErrorDeleteContentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Delete content successful", logger.F("contentID", req.ContentId))
		resp = h.converter.BuildDeleteContentResponse(true, "删除内容成功")
	}

	httpx.WriteObject(c, resp, err)
}

// PublishContent 发布内容
func (h *HTTPHandler) PublishContent(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.PublishContentRequest
		resp *rest.PublishContentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid publish content request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorPublishContentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.AuthorId)
	ctx = tracecontext.WithContentID(ctx, req.ContentId)

	content, err := h.svc.PublishContent(ctx, req.ContentId, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Publish content failed", logger.F("error", err.Error()), logger.F("contentID", req.ContentId))
		resp = h.converter.BuildErrorPublishContentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Publish content successful", logger.F("contentID", req.ContentId))
		resp = h.converter.BuildPublishContentResponse(true, "发布内容成功", content)
	}

	httpx.WriteObject(c, resp, err)
}

// ChangeContentStatus 变更内容状态
func (h *HTTPHandler) ChangeContentStatus(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.ChangeContentStatusRequest
		resp *rest.ChangeContentStatusResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid change content status request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorChangeContentStatusResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.OperatorId)
	ctx = tracecontext.WithContentID(ctx, req.ContentId)

	// 转换状态枚举
	statusStr := h.converter.ContentStatusFromProto(req.NewStatus)

	content, err := h.svc.ChangeContentStatus(ctx, req.ContentId, req.OperatorId, statusStr, req.Reason)
	if err != nil {
		h.logger.Error(ctx, "Change content status failed", logger.F("error", err.Error()), logger.F("contentID", req.ContentId))
		resp = h.converter.BuildErrorChangeContentStatusResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Change content status successful", logger.F("contentID", req.ContentId), logger.F("status", statusStr))
		resp = h.converter.BuildChangeContentStatusResponse(true, "变更内容状态成功", content)
	}

	httpx.WriteObject(c, resp, err)
}

// GetUserContent 获取用户内容列表
func (h *HTTPHandler) GetUserContent(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetUserContentRequest
		resp *rest.GetUserContentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user content request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetUserContentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.AuthorId)

	// 转换状态枚举
	statusStr := h.converter.ContentStatusFromProto(req.Status)

	contents, total, err := h.svc.GetUserContent(ctx, req.AuthorId, statusStr, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Get user content failed", logger.F("error", err.Error()), logger.F("authorID", req.AuthorId))
		resp = h.converter.BuildErrorGetUserContentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get user content successful", logger.F("authorID", req.AuthorId), logger.F("count", len(contents)))
		resp = h.converter.BuildGetUserContentResponse(true, "获取用户内容成功", contents, total, req.Page, req.PageSize)
	}

	httpx.WriteObject(c, resp, err)
}

// GetContentStats 获取内容统计
func (h *HTTPHandler) GetContentStats(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetContentStatsRequest
		resp *rest.GetContentStatsResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content stats request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetContentStatsResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	if req.AuthorId > 0 {
		ctx = tracecontext.WithUserID(ctx, req.AuthorId)
	}

	stats, err := h.svc.GetContentStats(ctx, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Get content stats failed", logger.F("error", err.Error()), logger.F("authorID", req.AuthorId))
		resp = h.converter.BuildErrorGetContentStatsResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get content stats successful", logger.F("authorID", req.AuthorId))
		resp = h.converter.BuildGetContentStatsResponse(true, "获取内容统计成功", stats)
	}

	httpx.WriteObject(c, resp, err)
}
