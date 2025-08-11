package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/pkg/logger"
)

// createTagImpl 创建标签实现
func (h *GRPCHandler) createTagImpl(ctx context.Context, req *rest.CreateTagRequest) (*rest.CreateTagResponse, error) {
	tag, err := h.svc.CreateTag(ctx, req.Name)
	if err != nil {
		h.logger.Error(ctx, "Failed to create tag via gRPC",
			logger.F("error", err.Error()),
			logger.F("name", req.Name))
		return h.converter.BuildErrorCreateTagResponse(err.Error()), nil
	}

	return h.converter.BuildCreateTagResponse(true, "创建成功", tag), nil
}

// getTagsImpl 获取标签列表实现
func (h *GRPCHandler) getTagsImpl(ctx context.Context, req *rest.GetTagsRequest) (*rest.GetTagsResponse, error) {
	tags, total, err := h.svc.GetTags(ctx, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Failed to get tags via gRPC",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
		return h.converter.BuildErrorGetTagsResponse(err.Error()), nil
	}

	return h.converter.BuildGetTagsResponse(true, "获取成功", tags, total), nil
}

// createTopicImpl 创建话题实现
func (h *GRPCHandler) createTopicImpl(ctx context.Context, req *rest.CreateTopicRequest) (*rest.CreateTopicResponse, error) {
	topic, err := h.svc.CreateTopic(ctx, req.Name, req.Description, req.CoverImage)
	if err != nil {
		h.logger.Error(ctx, "Failed to create topic via gRPC",
			logger.F("error", err.Error()),
			logger.F("name", req.Name))
		return h.converter.BuildErrorCreateTopicResponse(err.Error()), nil
	}

	return h.converter.BuildCreateTopicResponse(true, "创建成功", topic), nil
}

// getTopicsImpl 获取话题列表实现
func (h *GRPCHandler) getTopicsImpl(ctx context.Context, req *rest.GetTopicsRequest) (*rest.GetTopicsResponse, error) {
	topics, total, err := h.svc.GetTopics(ctx, req.Keyword, req.HotOnly, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Failed to get topics via gRPC",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
		return h.converter.BuildErrorGetTopicsResponse(err.Error()), nil
	}

	return h.converter.BuildGetTopicsResponse(true, "获取成功", topics, total), nil
}
