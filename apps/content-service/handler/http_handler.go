package handler

import (
	"goim-social/api/rest"
	"goim-social/apps/content-service/converter"
	"goim-social/apps/content-service/model"
	"goim-social/apps/content-service/service"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"

	"github.com/gin-gonic/gin"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    log,
	}
}

// RegisterRoutes 注册HTTP路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/content")
	{
		// 内容管理
		api.POST("/create", h.CreateContent)              // 创建内容
		api.POST("/update", h.UpdateContent)              // 更新内容
		api.POST("/get", h.GetContent)                    // 获取内容详情
		api.POST("/delete", h.DeleteContent)              // 删除内容
		api.POST("/publish", h.PublishContent)            // 发布内容
		api.POST("/change_status", h.ChangeContentStatus) // 变更内容状态

		// 内容查询
		api.POST("/search", h.SearchContent)        // 搜索内容
		api.POST("/user_content", h.GetUserContent) // 获取用户内容列表
		api.POST("/stats", h.GetContentStats)       // 获取内容统计

		// 标签管理
		api.POST("/tag/create", h.CreateTag) // 创建标签
		api.POST("/tag/list", h.GetTags)     // 获取标签列表

		// 话题管理
		api.POST("/topic/create", h.CreateTopic) // 创建话题
		api.POST("/topic/list", h.GetTopics)     // 获取话题列表
	}
}

// CreateContent 创建内容
func (h *HTTPHandler) CreateContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create content request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorCreateContentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

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

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Create content failed", logger.F("error", err.Error()))
	} else {
		message = "创建成功"
	}

	res := h.converter.BuildCreateContentResponse(err == nil, message, content)
	httpx.WriteObject(c, res, err)
}

// UpdateContent 更新内容
func (h *HTTPHandler) UpdateContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.UpdateContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid update content request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorUpdateContentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

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

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Update content failed", logger.F("error", err.Error()))
	} else {
		message = "更新成功"
	}

	res := h.converter.BuildUpdateContentResponse(err == nil, message, content)
	httpx.WriteObject(c, res, err)
}

// GetContent 获取内容详情
func (h *HTTPHandler) GetContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetContentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	content, err := h.svc.GetContent(ctx, req.ContentId, req.UserId)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get content failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetContentResponse(err == nil, message, content)
	httpx.WriteObject(c, res, err)
}

// DeleteContent 删除内容
func (h *HTTPHandler) DeleteContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete content request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorDeleteContentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.DeleteContent(ctx, req.ContentId, req.AuthorId)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Delete content failed", logger.F("error", err.Error()))
	} else {
		message = "删除成功"
	}

	res := h.converter.BuildDeleteContentResponse(err == nil, message)
	httpx.WriteObject(c, res, err)
}

// PublishContent 发布内容
func (h *HTTPHandler) PublishContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.PublishContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid publish content request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorPublishContentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	content, err := h.svc.PublishContent(ctx, req.ContentId, req.AuthorId)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Publish content failed", logger.F("error", err.Error()))
	} else {
		message = "发布成功"
	}

	res := h.converter.BuildPublishContentResponse(err == nil, message, content)
	httpx.WriteObject(c, res, err)
}

// ChangeContentStatus 变更内容状态
func (h *HTTPHandler) ChangeContentStatus(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ChangeContentStatusRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid change content status request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorChangeContentStatusResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换状态枚举
	newStatus := h.converter.ContentStatusFromProto(req.NewStatus)

	content, err := h.svc.ChangeContentStatus(ctx, req.ContentId, req.OperatorId, newStatus, req.Reason)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Change content status failed", logger.F("error", err.Error()))
	} else {
		message = "状态变更成功"
	}

	res := h.converter.BuildChangeContentStatusResponse(err == nil, message, content)
	httpx.WriteObject(c, res, err)
}

// SearchContent 搜索内容
func (h *HTTPHandler) SearchContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.SearchContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid search content request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorSearchContentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	contentType := h.converter.ContentTypeFromProto(req.Type)
	status := h.converter.ContentStatusFromProto(req.Status)

	params := &model.SearchContentParams{
		Keyword:   req.Keyword,
		Type:      contentType,
		Status:    status,
		TagIDs:    req.TagIds,
		TopicIDs:  req.TopicIds,
		AuthorID:  req.AuthorId,
		Page:      req.Page,
		PageSize:  req.PageSize,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	contents, total, err := h.svc.SearchContent(ctx, params)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Search content failed", logger.F("error", err.Error()))
	} else {
		message = "搜索成功"
	}

	res := h.converter.BuildSearchContentResponse(err == nil, message, contents, total, req.Page, req.PageSize)
	httpx.WriteObject(c, res, err)
}

// GetUserContent 获取用户内容列表
func (h *HTTPHandler) GetUserContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user content request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetUserContentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换状态枚举
	status := h.converter.ContentStatusFromProto(req.Status)

	contents, total, err := h.svc.GetUserContent(ctx, req.AuthorId, status, req.Page, req.PageSize)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get user content failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetUserContentResponse(err == nil, message, contents, total, req.Page, req.PageSize)
	httpx.WriteObject(c, res, err)
}

// GetContentStats 获取内容统计
func (h *HTTPHandler) GetContentStats(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetContentStatsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content stats request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetContentStatsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	stats, err := h.svc.GetContentStats(ctx, req.AuthorId)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get content stats failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetContentStatsResponse(err == nil, message, stats)
	httpx.WriteObject(c, res, err)
}

// CreateTag 创建标签
func (h *HTTPHandler) CreateTag(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateTagRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create tag request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorCreateTagResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	tag, err := h.svc.CreateTag(ctx, req.Name)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Create tag failed", logger.F("error", err.Error()))
	} else {
		message = "创建成功"
	}

	res := h.converter.BuildCreateTagResponse(err == nil, message, tag)
	httpx.WriteObject(c, res, err)
}

// GetTags 获取标签列表
func (h *HTTPHandler) GetTags(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetTagsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get tags request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetTagsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	tags, total, err := h.svc.GetTags(ctx, req.Keyword, req.Page, req.PageSize)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get tags failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetTagsResponse(err == nil, message, tags, total)
	httpx.WriteObject(c, res, err)
}

// CreateTopic 创建话题
func (h *HTTPHandler) CreateTopic(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateTopicRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create topic request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorCreateTopicResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	topic, err := h.svc.CreateTopic(ctx, req.Name, req.Description, req.CoverImage)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Create topic failed", logger.F("error", err.Error()))
	} else {
		message = "创建成功"
	}

	res := h.converter.BuildCreateTopicResponse(err == nil, message, topic)
	httpx.WriteObject(c, res, err)
}

// GetTopics 获取话题列表
func (h *HTTPHandler) GetTopics(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetTopicsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get topics request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetTopicsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	topics, total, err := h.svc.GetTopics(ctx, req.Keyword, req.HotOnly, req.Page, req.PageSize)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get topics failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetTopicsResponse(err == nil, message, topics, total)
	httpx.WriteObject(c, res, err)
}
