package handler

import (
	"context"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// createContentImpl 创建内容实现
func (h *GRPCHandler) createContentImpl(ctx context.Context, req *rest.CreateContentRequest) (*rest.CreateContentResponse, error) {
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
		h.logger.Error(ctx, "Failed to create content via gRPC",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorId),
			logger.F("title", req.Title))
		return h.converter.BuildErrorCreateContentResponse(err.Error()), nil
	}

	h.logger.Info(ctx, "Create content via gRPC successful",
		logger.F("contentID", content.ID),
		logger.F("authorID", req.AuthorId),
		logger.F("title", req.Title))

	return h.converter.BuildCreateContentResponse(true, "创建成功", content), nil
}

// updateContentImpl 更新内容实现
func (h *GRPCHandler) updateContentImpl(ctx context.Context, req *rest.UpdateContentRequest) (*rest.UpdateContentResponse, error) {
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
		h.logger.Error(ctx, "Failed to update content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return h.converter.BuildErrorUpdateContentResponse(err.Error()), nil
	}

	return h.converter.BuildUpdateContentResponse(true, "更新成功", content), nil
}

// getContentImpl 获取内容实现
func (h *GRPCHandler) getContentImpl(ctx context.Context, req *rest.GetContentRequest) (*rest.GetContentResponse, error) {
	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)
	ctx = tracecontext.WithContentID(ctx, req.ContentId)

	content, err := h.svc.GetContent(ctx, req.ContentId, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "Failed to get content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId),
			logger.F("userID", req.UserId))
		return h.converter.BuildErrorGetContentResponse(err.Error()), nil
	}

	h.logger.Info(ctx, "Get content via gRPC successful",
		logger.F("contentID", req.ContentId),
		logger.F("userID", req.UserId),
		logger.F("contentTitle", content.Title))

	return h.converter.BuildGetContentResponse(true, "获取成功", content), nil
}

// deleteContentImpl 删除内容实现
func (h *GRPCHandler) deleteContentImpl(ctx context.Context, req *rest.DeleteContentRequest) (*rest.DeleteContentResponse, error) {
	err := h.svc.DeleteContent(ctx, req.ContentId, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return h.converter.BuildErrorDeleteContentResponse(err.Error()), nil
	}

	return h.converter.BuildDeleteContentResponse(true, "删除成功"), nil
}

// publishContentImpl 发布内容实现
func (h *GRPCHandler) publishContentImpl(ctx context.Context, req *rest.PublishContentRequest) (*rest.PublishContentResponse, error) {
	content, err := h.svc.PublishContent(ctx, req.ContentId, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to publish content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return h.converter.BuildErrorPublishContentResponse(err.Error()), nil
	}

	return h.converter.BuildPublishContentResponse(true, "发布成功", content), nil
}

// changeContentStatusImpl 变更内容状态实现
func (h *GRPCHandler) changeContentStatusImpl(ctx context.Context, req *rest.ChangeContentStatusRequest) (*rest.ChangeContentStatusResponse, error) {
	newStatus := h.converter.ContentStatusFromProto(req.NewStatus)

	content, err := h.svc.ChangeContentStatus(ctx, req.ContentId, req.OperatorId, newStatus, req.Reason)
	if err != nil {
		h.logger.Error(ctx, "Failed to change content status via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return h.converter.BuildErrorChangeContentStatusResponse(err.Error()), nil
	}

	return h.converter.BuildChangeContentStatusResponse(true, "状态变更成功", content), nil
}

// getUserContentImpl 获取用户内容列表实现
func (h *GRPCHandler) getUserContentImpl(ctx context.Context, req *rest.GetUserContentRequest) (*rest.GetUserContentResponse, error) {
	status := h.converter.ContentStatusFromProto(req.Status)

	contents, total, err := h.svc.GetUserContent(ctx, req.AuthorId, status, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user content via gRPC",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorId))
		return h.converter.BuildErrorGetUserContentResponse(err.Error()), nil
	}

	return h.converter.BuildGetUserContentResponse(true, "获取成功", contents, total, req.Page, req.PageSize), nil
}

// getContentStatsImpl 获取内容统计实现
func (h *GRPCHandler) getContentStatsImpl(ctx context.Context, req *rest.GetContentStatsRequest) (*rest.GetContentStatsResponse, error) {
	stats, err := h.svc.GetContentStats(ctx, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to get content stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorId))
		return h.converter.BuildErrorGetContentStatsResponse(err.Error()), nil
	}

	return h.converter.BuildGetContentStatsResponse(true, "获取成功", stats), nil
}
