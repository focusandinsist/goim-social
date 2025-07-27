package handler

import (
	"websocket-server/api/rest"
	"websocket-server/apps/content-service/model"
	"websocket-server/apps/content-service/service"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/utils"

	"github.com/gin-gonic/gin"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	svc    *service.Service
	logger logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:    svc,
		logger: log,
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
		res := &rest.CreateContentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换媒体文件
	var mediaFiles []model.ContentMediaFile
	for _, mf := range req.MediaFiles {
		mediaFiles = append(mediaFiles, model.ContentMediaFile{
			URL:      mf.Url,
			Filename: mf.Filename,
			Size:     mf.Size,
			MimeType: mf.MimeType,
			Width:    mf.Width,
			Height:   mf.Height,
			Duration: mf.Duration,
		})
	}

	// 转换内容类型
	contentType := convertContentTypeFromProto(req.Type)

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

	res := &rest.CreateContentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "创建成功"
		}(),
		Content: func() *rest.Content {
			if err != nil {
				return nil
			}
			return convertContentToProto(content)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Create content failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// UpdateContent 更新内容
func (h *HTTPHandler) UpdateContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.UpdateContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid update content request", logger.F("error", err.Error()))
		res := &rest.UpdateContentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换媒体文件
	var mediaFiles []model.ContentMediaFile
	for _, mf := range req.MediaFiles {
		mediaFiles = append(mediaFiles, model.ContentMediaFile{
			URL:      mf.Url,
			Filename: mf.Filename,
			Size:     mf.Size,
			MimeType: mf.MimeType,
			Width:    mf.Width,
			Height:   mf.Height,
			Duration: mf.Duration,
		})
	}

	// 转换内容类型
	contentType := convertContentTypeFromProto(req.Type)

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

	res := &rest.UpdateContentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "更新成功"
		}(),
		Content: func() *rest.Content {
			if err != nil {
				return nil
			}
			return convertContentToProto(content)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Update content failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetContent 获取内容详情
func (h *HTTPHandler) GetContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content request", logger.F("error", err.Error()))
		res := &rest.GetContentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	content, err := h.svc.GetContent(ctx, req.ContentId, req.UserId)
	res := &rest.GetContentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Content: func() *rest.Content {
			if err != nil {
				return nil
			}
			return convertContentToProto(content)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Get content failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// DeleteContent 删除内容
func (h *HTTPHandler) DeleteContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete content request", logger.F("error", err.Error()))
		res := &rest.DeleteContentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err := h.svc.DeleteContent(ctx, req.ContentId, req.AuthorId)
	res := &rest.DeleteContentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "删除成功"
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Delete content failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// PublishContent 发布内容
func (h *HTTPHandler) PublishContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.PublishContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid publish content request", logger.F("error", err.Error()))
		res := &rest.PublishContentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	content, err := h.svc.PublishContent(ctx, req.ContentId, req.AuthorId)
	res := &rest.PublishContentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "发布成功"
		}(),
		Content: func() *rest.Content {
			if err != nil {
				return nil
			}
			return convertContentToProto(content)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Publish content failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// ChangeContentStatus 变更内容状态
func (h *HTTPHandler) ChangeContentStatus(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ChangeContentStatusRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid change content status request", logger.F("error", err.Error()))
		res := &rest.ChangeContentStatusResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换状态枚举
	newStatus := convertContentStatusFromProto(req.NewStatus)

	content, err := h.svc.ChangeContentStatus(ctx, req.ContentId, req.OperatorId, newStatus, req.Reason)
	res := &rest.ChangeContentStatusResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "状态变更成功"
		}(),
		Content: func() *rest.Content {
			if err != nil {
				return nil
			}
			return convertContentToProto(content)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Change content status failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// SearchContent 搜索内容
func (h *HTTPHandler) SearchContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.SearchContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid search content request", logger.F("error", err.Error()))
		res := &rest.SearchContentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	contentType := convertContentTypeFromProto(req.Type)
	status := convertContentStatusFromProto(req.Status)

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

	// 转换内容列表
	var protoContents []*rest.Content
	if err == nil {
		for _, content := range contents {
			protoContents = append(protoContents, convertContentToProto(content))
		}
	}

	res := &rest.SearchContentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "搜索成功"
		}(),
		Contents: protoContents,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if err != nil {
		h.logger.Error(ctx, "Search content failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetUserContent 获取用户内容列表
func (h *HTTPHandler) GetUserContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user content request", logger.F("error", err.Error()))
		res := &rest.GetUserContentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换状态枚举
	status := convertContentStatusFromProto(req.Status)

	contents, total, err := h.svc.GetUserContent(ctx, req.AuthorId, status, req.Page, req.PageSize)

	// 转换内容列表
	var protoContents []*rest.Content
	if err == nil {
		for _, content := range contents {
			protoContents = append(protoContents, convertContentToProto(content))
		}
	}

	res := &rest.GetUserContentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Contents: protoContents,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if err != nil {
		h.logger.Error(ctx, "Get user content failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetContentStats 获取内容统计
func (h *HTTPHandler) GetContentStats(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetContentStatsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content stats request", logger.F("error", err.Error()))
		res := &rest.GetContentStatsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	stats, err := h.svc.GetContentStats(ctx, req.AuthorId)
	res := &rest.GetContentStatsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		TotalContents: func() int64 {
			if err != nil {
				return 0
			}
			return stats.TotalContents
		}(),
		PublishedContents: func() int64 {
			if err != nil {
				return 0
			}
			return stats.PublishedContents
		}(),
		DraftContents: func() int64 {
			if err != nil {
				return 0
			}
			return stats.DraftContents
		}(),
		PendingContents: func() int64 {
			if err != nil {
				return 0
			}
			return stats.PendingContents
		}(),
		TotalViews: func() int64 {
			if err != nil {
				return 0
			}
			return stats.TotalViews
		}(),
		TotalLikes: func() int64 {
			if err != nil {
				return 0
			}
			return stats.TotalLikes
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Get content stats failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// CreateTag 创建标签
func (h *HTTPHandler) CreateTag(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateTagRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create tag request", logger.F("error", err.Error()))
		res := &rest.CreateTagResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	tag, err := h.svc.CreateTag(ctx, req.Name)
	res := &rest.CreateTagResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "创建成功"
		}(),
		Tag: func() *rest.ContentTag {
			if err != nil {
				return nil
			}
			return convertTagToProto(tag)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Create tag failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetTags 获取标签列表
func (h *HTTPHandler) GetTags(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetTagsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get tags request", logger.F("error", err.Error()))
		res := &rest.GetTagsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	tags, total, err := h.svc.GetTags(ctx, req.Keyword, req.Page, req.PageSize)

	// 转换标签列表
	var protoTags []*rest.ContentTag
	if err == nil {
		for _, tag := range tags {
			protoTags = append(protoTags, convertTagToProto(tag))
		}
	}

	res := &rest.GetTagsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Tags:  protoTags,
		Total: total,
	}
	if err != nil {
		h.logger.Error(ctx, "Get tags failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// CreateTopic 创建话题
func (h *HTTPHandler) CreateTopic(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateTopicRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create topic request", logger.F("error", err.Error()))
		res := &rest.CreateTopicResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	topic, err := h.svc.CreateTopic(ctx, req.Name, req.Description, req.CoverImage)
	res := &rest.CreateTopicResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "创建成功"
		}(),
		Topic: func() *rest.ContentTopic {
			if err != nil {
				return nil
			}
			return convertTopicToProto(topic)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Create topic failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetTopics 获取话题列表
func (h *HTTPHandler) GetTopics(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetTopicsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get topics request", logger.F("error", err.Error()))
		res := &rest.GetTopicsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	topics, total, err := h.svc.GetTopics(ctx, req.Keyword, req.HotOnly, req.Page, req.PageSize)

	// 转换话题列表
	var protoTopics []*rest.ContentTopic
	if err == nil {
		for _, topic := range topics {
			protoTopics = append(protoTopics, convertTopicToProto(topic))
		}
	}

	res := &rest.GetTopicsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Topics: protoTopics,
		Total:  total,
	}
	if err != nil {
		h.logger.Error(ctx, "Get topics failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}
