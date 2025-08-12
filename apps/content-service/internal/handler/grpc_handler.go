package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/content-service/internal/converter"
	"goim-social/apps/content-service/internal/service"
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
	return h.createContentImpl(ctx, req)
}

// UpdateContent 更新内容
func (h *GRPCHandler) UpdateContent(ctx context.Context, req *rest.UpdateContentRequest) (*rest.UpdateContentResponse, error) {
	return h.updateContentImpl(ctx, req)
}

// GetContent 获取内容
func (h *GRPCHandler) GetContent(ctx context.Context, req *rest.GetContentRequest) (*rest.GetContentResponse, error) {
	return h.getContentImpl(ctx, req)
}

// DeleteContent 删除内容
func (h *GRPCHandler) DeleteContent(ctx context.Context, req *rest.DeleteContentRequest) (*rest.DeleteContentResponse, error) {
	return h.deleteContentImpl(ctx, req)
}

// PublishContent 发布内容
func (h *GRPCHandler) PublishContent(ctx context.Context, req *rest.PublishContentRequest) (*rest.PublishContentResponse, error) {
	return h.publishContentImpl(ctx, req)
}

// ChangeContentStatus 变更内容状态
func (h *GRPCHandler) ChangeContentStatus(ctx context.Context, req *rest.ChangeContentStatusRequest) (*rest.ChangeContentStatusResponse, error) {
	return h.changeContentStatusImpl(ctx, req)
}

// GetUserContent 获取用户内容列表
func (h *GRPCHandler) GetUserContent(ctx context.Context, req *rest.GetUserContentRequest) (*rest.GetUserContentResponse, error) {
	return h.getUserContentImpl(ctx, req)
}

// GetContentStats 获取内容统计
func (h *GRPCHandler) GetContentStats(ctx context.Context, req *rest.GetContentStatsRequest) (*rest.GetContentStatsResponse, error) {
	return h.getContentStatsImpl(ctx, req)
}

// CreateTag 创建标签
func (h *GRPCHandler) CreateTag(ctx context.Context, req *rest.CreateTagRequest) (*rest.CreateTagResponse, error) {
	return h.createTagImpl(ctx, req)
}

// GetTags 获取标签列表
func (h *GRPCHandler) GetTags(ctx context.Context, req *rest.GetTagsRequest) (*rest.GetTagsResponse, error) {
	return h.getTagsImpl(ctx, req)
}

// CreateTopic 创建话题
func (h *GRPCHandler) CreateTopic(ctx context.Context, req *rest.CreateTopicRequest) (*rest.CreateTopicResponse, error) {
	return h.createTopicImpl(ctx, req)
}

// GetTopics 获取话题列表
func (h *GRPCHandler) GetTopics(ctx context.Context, req *rest.GetTopicsRequest) (*rest.GetTopicsResponse, error) {
	return h.getTopicsImpl(ctx, req)
}
