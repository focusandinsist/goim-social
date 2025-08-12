package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// CreateTag 创建标签
func (h *HTTPHandler) CreateTag(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.CreateTagRequest
		resp *rest.CreateTagResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create tag request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorCreateTagResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	tag, err := h.svc.CreateTag(ctx, req.Name)
	if err != nil {
		h.logger.Error(ctx, "Create tag failed", logger.F("error", err.Error()), logger.F("name", req.Name))
		resp = h.converter.BuildErrorCreateTagResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Create tag successful", logger.F("tagID", tag.ID), logger.F("name", tag.Name))
		resp = h.converter.BuildCreateTagResponse(true, "创建标签成功", tag)
	}

	httpx.WriteObject(c, resp, err)
}

// GetTags 获取标签列表
func (h *HTTPHandler) GetTags(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetTagsRequest
		resp *rest.GetTagsResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get tags request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetTagsResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	tags, total, err := h.svc.GetTags(ctx, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Get tags failed", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetTagsResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get tags successful", logger.F("count", len(tags)))
		resp = h.converter.BuildGetTagsResponse(true, "获取标签列表成功", tags, total)
	}

	httpx.WriteObject(c, resp, err)
}

// CreateTopic 创建话题
func (h *HTTPHandler) CreateTopic(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.CreateTopicRequest
		resp *rest.CreateTopicResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create topic request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorCreateTopicResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	topic, err := h.svc.CreateTopic(ctx, req.Name, req.Description, req.CoverImage)
	if err != nil {
		h.logger.Error(ctx, "Create topic failed", logger.F("error", err.Error()), logger.F("name", req.Name))
		resp = h.converter.BuildErrorCreateTopicResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Create topic successful", logger.F("topicID", topic.ID), logger.F("name", topic.Name))
		resp = h.converter.BuildCreateTopicResponse(true, "创建话题成功", topic)
	}

	httpx.WriteObject(c, resp, err)
}

// GetTopics 获取话题列表
func (h *HTTPHandler) GetTopics(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetTopicsRequest
		resp *rest.GetTopicsResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get topics request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetTopicsResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	topics, total, err := h.svc.GetTopics(ctx, req.Keyword, req.HotOnly, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Get topics failed", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetTopicsResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get topics successful", logger.F("count", len(topics)))
		resp = h.converter.BuildGetTopicsResponse(true, "获取话题列表成功", topics, total)
	}

	httpx.WriteObject(c, resp, err)
}
