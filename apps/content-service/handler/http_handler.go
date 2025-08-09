package handler

import (
	"goim-social/api/rest"
	"goim-social/apps/content-service/converter"
	"goim-social/apps/content-service/service"
	tracecontext "goim-social/pkg/context"
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
		api.POST("/user_content", h.GetUserContent) // 获取用户内容列表
		api.POST("/stats", h.GetContentStats)       // 获取内容统计

		// 标签管理
		api.POST("/tag/create", h.CreateTag) // 创建标签
		api.POST("/tag/list", h.GetTags)     // 获取标签列表

		// 话题管理
		api.POST("/topic/create", h.CreateTopic) // 创建话题
		api.POST("/topic/list", h.GetTopics)     // 获取话题列表

		// 评论管理
		api.POST("/comment/create", h.CreateComment)      // 创建评论
		api.POST("/comment/delete", h.DeleteComment)      // 删除评论
		api.POST("/comment/list", h.GetComments)          // 获取评论列表
		api.POST("/comment/replies", h.GetCommentReplies) // 获取评论回复

		// 互动管理
		api.POST("/interaction/do", h.DoInteraction)          // 执行互动（点赞/收藏/分享等）
		api.POST("/interaction/undo", h.UndoInteraction)      // 取消互动
		api.POST("/interaction/check", h.CheckInteraction)    // 检查互动状态
		api.POST("/interaction/stats", h.GetInteractionStats) // 获取互动统计

		// 聚合查询
		api.POST("/detail", h.GetContentDetail)     // 获取内容详情（包含评论和互动）
		api.POST("/feed", h.GetContentFeed)         // 获取内容流
		api.POST("/trending", h.GetTrendingContent) // 获取热门内容
	}

	// 为了向后兼容，也注册独立的评论和互动路由
	comment := r.Group("/api/v1/comment")
	{
		comment.POST("/create", h.CreateComment)
		comment.POST("/delete", h.DeleteComment)
		comment.POST("/list", h.GetComments)
		comment.POST("/replies", h.GetCommentReplies)
	}

	interaction := r.Group("/api/v1/interaction")
	{
		interaction.POST("/do", h.DoInteraction)
		interaction.POST("/undo", h.UndoInteraction)
		interaction.POST("/check", h.CheckInteraction)
		interaction.POST("/stats", h.GetInteractionStats)
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

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Create content failed",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorId),
			logger.F("title", req.Title))
	} else {
		message = "创建成功"
		h.logger.Info(ctx, "Create content successful",
			logger.F("contentID", content.ID),
			logger.F("authorID", req.AuthorId),
			logger.F("title", req.Title))
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

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)
	ctx = tracecontext.WithContentID(ctx, req.ContentId)

	content, err := h.svc.GetContent(ctx, req.ContentId, req.UserId)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get content failed",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId),
			logger.F("userID", req.UserId))
	} else {
		message = "获取成功"
		h.logger.Info(ctx, "Get content successful",
			logger.F("contentID", req.ContentId),
			logger.F("userID", req.UserId),
			logger.F("contentTitle", content.Title))
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

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.AuthorId)
	ctx = tracecontext.WithContentID(ctx, req.ContentId)

	content, err := h.svc.PublishContent(ctx, req.ContentId, req.AuthorId)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Publish content failed",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId),
			logger.F("authorID", req.AuthorId))
	} else {
		message = "发布成功"
		h.logger.Info(ctx, "Publish content successful",
			logger.F("contentID", req.ContentId),
			logger.F("authorID", req.AuthorId),
			logger.F("contentTitle", content.Title))
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

// ==================== 评论相关处理函数 ====================

// CreateComment 创建评论
func (h *HTTPHandler) CreateComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create comment request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorCreateCommentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 构建创建评论参数
	params := service.CreateCommentParams{
		TargetID:        req.TargetId,
		TargetType:      h.converter.TargetTypeToString(req.TargetType),
		UserID:          req.UserId,
		UserName:        req.UserName,
		UserAvatar:      req.UserAvatar,
		Content:         req.Content,
		ParentID:        req.ParentId,
		ReplyToUserID:   req.ReplyToUserId,
		ReplyToUserName: req.ReplyToUserName,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.GetHeader("User-Agent"),
	}

	comment, err := h.svc.CreateComment(ctx, params)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Create comment failed", logger.F("error", err.Error()))
	} else {
		message = "评论成功"
	}

	res := h.converter.BuildCreateCommentResponse(err == nil, message, comment)
	httpx.WriteObject(c, res, err)
}

// DeleteComment 删除评论
func (h *HTTPHandler) DeleteComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete comment request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorDeleteCommentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.DeleteComment(ctx, req.CommentId, req.UserId)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Delete comment failed", logger.F("error", err.Error()))
	} else {
		message = "删除成功"
	}

	res := h.converter.BuildDeleteCommentResponse(err == nil, message)
	httpx.WriteObject(c, res, err)
}

// GetComments 获取评论列表
func (h *HTTPHandler) GetComments(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetCommentsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comments request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetCommentsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	comments, total, err := h.svc.GetComments(ctx, req.TargetId, h.converter.TargetTypeToString(req.TargetType), req.ParentId, req.SortBy, req.SortOrder, req.Page, req.PageSize)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get comments failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetCommentsResponse(err == nil, message, comments, total)
	httpx.WriteObject(c, res, err)
}

// GetCommentReplies 获取评论回复
func (h *HTTPHandler) GetCommentReplies(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetCommentRepliesRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comment replies request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetCommentRepliesResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	replies, total, err := h.svc.GetCommentReplies(ctx, req.CommentId, req.SortBy, req.SortOrder, req.Page, req.PageSize)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get comment replies failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetCommentRepliesResponse(err == nil, message, replies, total)
	httpx.WriteObject(c, res, err)
}

// ==================== 互动相关处理函数 ====================

// DoInteraction 执行互动操作
func (h *HTTPHandler) DoInteraction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DoInteractionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid do interaction request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorDoInteractionResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	interaction, err := h.svc.DoInteraction(ctx, req.UserId, req.TargetId, h.converter.TargetTypeToString(req.TargetType), h.converter.InteractionTypeToString(req.InteractionType), req.Metadata)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Do interaction failed", logger.F("error", err.Error()))
	} else {
		message = "操作成功"
	}

	res := h.converter.BuildDoInteractionResponse(err == nil, message, interaction)
	httpx.WriteObject(c, res, err)
}

// UndoInteraction 取消互动操作
func (h *HTTPHandler) UndoInteraction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.UndoInteractionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid undo interaction request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorUndoInteractionResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.UndoInteraction(ctx, req.UserId, req.TargetId, h.converter.TargetTypeToString(req.TargetType), h.converter.InteractionTypeToString(req.InteractionType))

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Undo interaction failed", logger.F("error", err.Error()))
	} else {
		message = "取消成功"
	}

	res := h.converter.BuildUndoInteractionResponse(err == nil, message)
	httpx.WriteObject(c, res, err)
}

// CheckInteraction 检查互动状态
func (h *HTTPHandler) CheckInteraction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CheckInteractionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid check interaction request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorCheckInteractionResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	exists, interaction, err := h.svc.CheckInteraction(ctx, req.UserId, req.TargetId, h.converter.TargetTypeToString(req.TargetType), h.converter.InteractionTypeToString(req.InteractionType))

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Check interaction failed", logger.F("error", err.Error()))
	} else {
		message = "查询成功"
	}

	res := h.converter.BuildCheckInteractionResponse(err == nil, message, exists, interaction)
	httpx.WriteObject(c, res, err)
}

// GetInteractionStats 获取互动统计
func (h *HTTPHandler) GetInteractionStats(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetInteractionStatsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get interaction stats request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetInteractionStatsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	stats, err := h.svc.GetInteractionStats(ctx, req.TargetId, h.converter.TargetTypeToString(req.TargetType))

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get interaction stats failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetInteractionStatsResponse(err == nil, message, stats)
	httpx.WriteObject(c, res, err)
}

// ==================== 聚合查询处理函数 ====================

// GetContentDetail 获取内容详情（包含评论和互动）
func (h *HTTPHandler) GetContentDetail(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetContentDetailRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content detail request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetContentDetailResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	detail, err := h.svc.GetContentDetail(ctx, req.ContentId, req.UserId, req.CommentLimit)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get content detail failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetContentDetailResponse(err == nil, message, detail)
	httpx.WriteObject(c, res, err)
}

// GetContentFeed 获取内容流
func (h *HTTPHandler) GetContentFeed(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetContentFeedRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get content feed request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetContentFeedResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	feedItems, total, err := h.svc.GetContentFeed(ctx, req.UserId, req.ContentType, req.SortBy, req.Page, req.PageSize)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get content feed failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetContentFeedResponse(err == nil, message, feedItems, total)
	httpx.WriteObject(c, res, err)
}

// GetTrendingContent 获取热门内容
func (h *HTTPHandler) GetTrendingContent(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetTrendingContentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get trending content request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetTrendingContentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	trendingItems, err := h.svc.GetTrendingContent(ctx, req.TimeRange, req.ContentType, req.Limit)

	var message string
	if err != nil {
		message = err.Error()
		h.logger.Error(ctx, "Get trending content failed", logger.F("error", err.Error()))
	} else {
		message = "获取成功"
	}

	res := h.converter.BuildGetTrendingContentResponse(err == nil, message, trendingItems)
	httpx.WriteObject(c, res, err)
}
