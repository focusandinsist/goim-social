package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/content-service/converter"
	"goim-social/apps/content-service/model"
	"goim-social/apps/content-service/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedContentServiceServer
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    log,
	}
}

// CreateContent 创建内容
func (h *GRPCHandler) CreateContent(ctx context.Context, req *rest.CreateContentRequest) (*rest.CreateContentResponse, error) {
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

// UpdateContent 更新内容
func (h *GRPCHandler) UpdateContent(ctx context.Context, req *rest.UpdateContentRequest) (*rest.UpdateContentResponse, error) {
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

// GetContent 获取内容
func (h *GRPCHandler) GetContent(ctx context.Context, req *rest.GetContentRequest) (*rest.GetContentResponse, error) {
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

// DeleteContent 删除内容
func (h *GRPCHandler) DeleteContent(ctx context.Context, req *rest.DeleteContentRequest) (*rest.DeleteContentResponse, error) {
	err := h.svc.DeleteContent(ctx, req.ContentId, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return h.converter.BuildErrorDeleteContentResponse(err.Error()), nil
	}

	return h.converter.BuildDeleteContentResponse(true, "删除成功"), nil
}

// PublishContent 发布内容
func (h *GRPCHandler) PublishContent(ctx context.Context, req *rest.PublishContentRequest) (*rest.PublishContentResponse, error) {
	content, err := h.svc.PublishContent(ctx, req.ContentId, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to publish content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return h.converter.BuildErrorPublishContentResponse(err.Error()), nil
	}

	return h.converter.BuildPublishContentResponse(true, "发布成功", content), nil
}

// ChangeContentStatus 变更内容状态
func (h *GRPCHandler) ChangeContentStatus(ctx context.Context, req *rest.ChangeContentStatusRequest) (*rest.ChangeContentStatusResponse, error) {
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

// SearchContent 搜索内容
func (h *GRPCHandler) SearchContent(ctx context.Context, req *rest.SearchContentRequest) (*rest.SearchContentResponse, error) {
	params := &model.SearchContentParams{
		Keyword:   req.Keyword,
		Type:      h.converter.ContentTypeFromProto(req.Type),
		Status:    h.converter.ContentStatusFromProto(req.Status),
		TagIDs:    req.TagIds,
		TopicIDs:  req.TopicIds,
		AuthorID:  req.AuthorId,
		Page:      req.Page,
		PageSize:  req.PageSize,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	contents, total, err := h.svc.SearchContent(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to search content via gRPC",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
		return h.converter.BuildErrorSearchContentResponse(err.Error()), nil
	}

	return h.converter.BuildSearchContentResponse(true, "搜索成功", contents, total, req.Page, req.PageSize), nil
}

// GetUserContent 获取用户内容列表
func (h *GRPCHandler) GetUserContent(ctx context.Context, req *rest.GetUserContentRequest) (*rest.GetUserContentResponse, error) {
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

// GetContentStats 获取内容统计
func (h *GRPCHandler) GetContentStats(ctx context.Context, req *rest.GetContentStatsRequest) (*rest.GetContentStatsResponse, error) {
	stats, err := h.svc.GetContentStats(ctx, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to get content stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorId))
		return h.converter.BuildErrorGetContentStatsResponse(err.Error()), nil
	}

	return h.converter.BuildGetContentStatsResponse(true, "获取成功", stats), nil
}

// CreateTag 创建标签
func (h *GRPCHandler) CreateTag(ctx context.Context, req *rest.CreateTagRequest) (*rest.CreateTagResponse, error) {
	tag, err := h.svc.CreateTag(ctx, req.Name)
	if err != nil {
		h.logger.Error(ctx, "Failed to create tag via gRPC",
			logger.F("error", err.Error()),
			logger.F("name", req.Name))
		return h.converter.BuildErrorCreateTagResponse(err.Error()), nil
	}

	return h.converter.BuildCreateTagResponse(true, "创建成功", tag), nil
}

// GetTags 获取标签列表
func (h *GRPCHandler) GetTags(ctx context.Context, req *rest.GetTagsRequest) (*rest.GetTagsResponse, error) {
	tags, total, err := h.svc.GetTags(ctx, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Failed to get tags via gRPC",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
		return h.converter.BuildErrorGetTagsResponse(err.Error()), nil
	}

	return h.converter.BuildGetTagsResponse(true, "获取成功", tags, total), nil
}

// CreateTopic 创建话题
func (h *GRPCHandler) CreateTopic(ctx context.Context, req *rest.CreateTopicRequest) (*rest.CreateTopicResponse, error) {
	topic, err := h.svc.CreateTopic(ctx, req.Name, req.Description, req.CoverImage)
	if err != nil {
		h.logger.Error(ctx, "Failed to create topic via gRPC",
			logger.F("error", err.Error()),
			logger.F("name", req.Name))
		return h.converter.BuildErrorCreateTopicResponse(err.Error()), nil
	}

	return h.converter.BuildCreateTopicResponse(true, "创建成功", topic), nil
}

// GetTopics 获取话题列表
func (h *GRPCHandler) GetTopics(ctx context.Context, req *rest.GetTopicsRequest) (*rest.GetTopicsResponse, error) {
	topics, total, err := h.svc.GetTopics(ctx, req.Keyword, req.HotOnly, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Failed to get topics via gRPC",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
		return h.converter.BuildErrorGetTopicsResponse(err.Error()), nil
	}

	return h.converter.BuildGetTopicsResponse(true, "获取成功", topics, total), nil
}
