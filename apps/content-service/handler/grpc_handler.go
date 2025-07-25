package handler

import (
	"context"
	"time"

	"websocket-server/api/rest"
	"websocket-server/apps/content-service/model"
	"websocket-server/apps/content-service/service"
	"websocket-server/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedContentServiceServer
	svc    *service.Service
	logger logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:    svc,
		logger: log,
	}
}

// CreateContent 创建内容
func (h *GRPCHandler) CreateContent(ctx context.Context, req *rest.CreateContentRequest) (*rest.CreateContentResponse, error) {
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
	if err != nil {
		h.logger.Error(ctx, "Failed to create content via gRPC",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorId))
		return &rest.CreateContentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.CreateContentResponse{
		Success: true,
		Message: "创建成功",
		Content: convertContentToProto(content),
	}, nil
}

// UpdateContent 更新内容
func (h *GRPCHandler) UpdateContent(ctx context.Context, req *rest.UpdateContentRequest) (*rest.UpdateContentResponse, error) {
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
	if err != nil {
		h.logger.Error(ctx, "Failed to update content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return &rest.UpdateContentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.UpdateContentResponse{
		Success: true,
		Message: "更新成功",
		Content: convertContentToProto(content),
	}, nil
}

// GetContent 获取内容
func (h *GRPCHandler) GetContent(ctx context.Context, req *rest.GetContentRequest) (*rest.GetContentResponse, error) {
	content, err := h.svc.GetContent(ctx, req.ContentId, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "Failed to get content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return &rest.GetContentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.GetContentResponse{
		Success: true,
		Message: "获取成功",
		Content: convertContentToProto(content),
	}, nil
}

// DeleteContent 删除内容
func (h *GRPCHandler) DeleteContent(ctx context.Context, req *rest.DeleteContentRequest) (*rest.DeleteContentResponse, error) {
	err := h.svc.DeleteContent(ctx, req.ContentId, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return &rest.DeleteContentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.DeleteContentResponse{
		Success: true,
		Message: "删除成功",
	}, nil
}

// PublishContent 发布内容
func (h *GRPCHandler) PublishContent(ctx context.Context, req *rest.PublishContentRequest) (*rest.PublishContentResponse, error) {
	content, err := h.svc.PublishContent(ctx, req.ContentId, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to publish content via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return &rest.PublishContentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.PublishContentResponse{
		Success: true,
		Message: "发布成功",
		Content: convertContentToProto(content),
	}, nil
}

// ChangeContentStatus 变更内容状态
func (h *GRPCHandler) ChangeContentStatus(ctx context.Context, req *rest.ChangeContentStatusRequest) (*rest.ChangeContentStatusResponse, error) {
	newStatus := convertContentStatusFromProto(req.NewStatus)

	content, err := h.svc.ChangeContentStatus(ctx, req.ContentId, req.OperatorId, newStatus, req.Reason)
	if err != nil {
		h.logger.Error(ctx, "Failed to change content status via gRPC",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentId))
		return &rest.ChangeContentStatusResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.ChangeContentStatusResponse{
		Success: true,
		Message: "状态变更成功",
		Content: convertContentToProto(content),
	}, nil
}

// SearchContent 搜索内容
func (h *GRPCHandler) SearchContent(ctx context.Context, req *rest.SearchContentRequest) (*rest.SearchContentResponse, error) {
	params := &model.SearchContentParams{
		Keyword:   req.Keyword,
		Type:      convertContentTypeFromProto(req.Type),
		Status:    convertContentStatusFromProto(req.Status),
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
		return &rest.SearchContentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoContents []*rest.Content
	for _, content := range contents {
		protoContents = append(protoContents, convertContentToProto(content))
	}

	return &rest.SearchContentResponse{
		Success:  true,
		Message:  "搜索成功",
		Contents: protoContents,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetUserContent 获取用户内容列表
func (h *GRPCHandler) GetUserContent(ctx context.Context, req *rest.GetUserContentRequest) (*rest.GetUserContentResponse, error) {
	status := convertContentStatusFromProto(req.Status)

	contents, total, err := h.svc.GetUserContent(ctx, req.AuthorId, status, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user content via gRPC",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorId))
		return &rest.GetUserContentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoContents []*rest.Content
	for _, content := range contents {
		protoContents = append(protoContents, convertContentToProto(content))
	}

	return &rest.GetUserContentResponse{
		Success:  true,
		Message:  "获取成功",
		Contents: protoContents,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetContentStats 获取内容统计
func (h *GRPCHandler) GetContentStats(ctx context.Context, req *rest.GetContentStatsRequest) (*rest.GetContentStatsResponse, error) {
	stats, err := h.svc.GetContentStats(ctx, req.AuthorId)
	if err != nil {
		h.logger.Error(ctx, "Failed to get content stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorId))
		return &rest.GetContentStatsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.GetContentStatsResponse{
		Success:           true,
		Message:           "获取成功",
		TotalContents:     stats.TotalContents,
		PublishedContents: stats.PublishedContents,
		DraftContents:     stats.DraftContents,
		PendingContents:   stats.PendingContents,
		TotalViews:        stats.TotalViews,
		TotalLikes:        stats.TotalLikes,
	}, nil
}

// CreateTag 创建标签
func (h *GRPCHandler) CreateTag(ctx context.Context, req *rest.CreateTagRequest) (*rest.CreateTagResponse, error) {
	tag, err := h.svc.CreateTag(ctx, req.Name)
	if err != nil {
		h.logger.Error(ctx, "Failed to create tag via gRPC",
			logger.F("error", err.Error()),
			logger.F("name", req.Name))
		return &rest.CreateTagResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.CreateTagResponse{
		Success: true,
		Message: "创建成功",
		Tag:     convertTagToProto(tag),
	}, nil
}

// GetTags 获取标签列表
func (h *GRPCHandler) GetTags(ctx context.Context, req *rest.GetTagsRequest) (*rest.GetTagsResponse, error) {
	tags, total, err := h.svc.GetTags(ctx, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Failed to get tags via gRPC",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
		return &rest.GetTagsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoTags []*rest.ContentTag
	for _, tag := range tags {
		protoTags = append(protoTags, convertTagToProto(tag))
	}

	return &rest.GetTagsResponse{
		Success: true,
		Message: "获取成功",
		Tags:    protoTags,
		Total:   total,
	}, nil
}

// CreateTopic 创建话题
func (h *GRPCHandler) CreateTopic(ctx context.Context, req *rest.CreateTopicRequest) (*rest.CreateTopicResponse, error) {
	topic, err := h.svc.CreateTopic(ctx, req.Name, req.Description, req.CoverImage)
	if err != nil {
		h.logger.Error(ctx, "Failed to create topic via gRPC",
			logger.F("error", err.Error()),
			logger.F("name", req.Name))
		return &rest.CreateTopicResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.CreateTopicResponse{
		Success: true,
		Message: "创建成功",
		Topic:   convertTopicToProto(topic),
	}, nil
}

// GetTopics 获取话题列表
func (h *GRPCHandler) GetTopics(ctx context.Context, req *rest.GetTopicsRequest) (*rest.GetTopicsResponse, error) {
	topics, total, err := h.svc.GetTopics(ctx, req.Keyword, req.HotOnly, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Failed to get topics via gRPC",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
		return &rest.GetTopicsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoTopics []*rest.ContentTopic
	for _, topic := range topics {
		protoTopics = append(protoTopics, convertTopicToProto(topic))
	}

	return &rest.GetTopicsResponse{
		Success: true,
		Message: "获取成功",
		Topics:  protoTopics,
		Total:   total,
	}, nil
}

// 转换函数

// convertContentToProto 将内容模型转换为protobuf格式
func convertContentToProto(content *model.Content) *rest.Content {
	if content == nil {
		return nil
	}

	// 转换媒体文件
	var mediaFiles []*rest.MediaFile
	for _, mf := range content.MediaFiles {
		mediaFiles = append(mediaFiles, &rest.MediaFile{
			Url:      mf.URL,
			Filename: mf.Filename,
			Size:     mf.Size,
			MimeType: mf.MimeType,
			Width:    mf.Width,
			Height:   mf.Height,
			Duration: mf.Duration,
		})
	}

	// 转换标签
	var tags []*rest.ContentTag
	for _, tag := range content.Tags {
		tags = append(tags, convertTagToProto(&tag))
	}

	// 转换话题
	var topics []*rest.ContentTopic
	for _, topic := range content.Topics {
		topics = append(topics, convertTopicToProto(&topic))
	}

	// 转换发布时间
	var publishedAt string
	if content.PublishedAt != nil {
		publishedAt = content.PublishedAt.Format(time.RFC3339)
	}

	return &rest.Content{
		Id:           content.ID,
		AuthorId:     content.AuthorID,
		Title:        content.Title,
		Content:      content.Content,
		Type:         convertContentTypeToProto(content.Type),
		Status:       convertContentStatusToProto(content.Status),
		MediaFiles:   mediaFiles,
		Tags:         tags,
		Topics:       topics,
		TemplateData: content.TemplateData,
		ViewCount:    content.ViewCount,
		LikeCount:    content.LikeCount,
		CommentCount: content.CommentCount,
		ShareCount:   content.ShareCount,
		CreatedAt:    content.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    content.UpdatedAt.Format(time.RFC3339),
		PublishedAt:  publishedAt,
	}
}

// convertTagToProto 将标签模型转换为protobuf格式
func convertTagToProto(tag *model.ContentTag) *rest.ContentTag {
	if tag == nil {
		return nil
	}
	return &rest.ContentTag{
		Id:         tag.ID,
		Name:       tag.Name,
		UsageCount: tag.UsageCount,
	}
}

// convertTopicToProto 将话题模型转换为protobuf格式
func convertTopicToProto(topic *model.ContentTopic) *rest.ContentTopic {
	if topic == nil {
		return nil
	}
	return &rest.ContentTopic{
		Id:           topic.ID,
		Name:         topic.Name,
		Description:  topic.Description,
		CoverImage:   topic.CoverImage,
		ContentCount: topic.ContentCount,
		IsHot:        topic.IsHot,
	}
}

// convertContentTypeToProto 将内容类型转换为protobuf枚举
func convertContentTypeToProto(contentType string) rest.ContentType {
	switch contentType {
	case model.ContentTypeText:
		return rest.ContentType_CONTENT_TYPE_TEXT
	case model.ContentTypeImage:
		return rest.ContentType_CONTENT_TYPE_IMAGE
	case model.ContentTypeVideo:
		return rest.ContentType_CONTENT_TYPE_VIDEO
	case model.ContentTypeAudio:
		return rest.ContentType_CONTENT_TYPE_AUDIO
	case model.ContentTypeMixed:
		return rest.ContentType_CONTENT_TYPE_MIXED
	case model.ContentTypeTemplate:
		return rest.ContentType_CONTENT_TYPE_TEMPLATE
	default:
		return rest.ContentType_CONTENT_TYPE_UNSPECIFIED
	}
}

// convertContentTypeFromProto 将protobuf枚举转换为内容类型
func convertContentTypeFromProto(contentType rest.ContentType) string {
	switch contentType {
	case rest.ContentType_CONTENT_TYPE_TEXT:
		return model.ContentTypeText
	case rest.ContentType_CONTENT_TYPE_IMAGE:
		return model.ContentTypeImage
	case rest.ContentType_CONTENT_TYPE_VIDEO:
		return model.ContentTypeVideo
	case rest.ContentType_CONTENT_TYPE_AUDIO:
		return model.ContentTypeAudio
	case rest.ContentType_CONTENT_TYPE_MIXED:
		return model.ContentTypeMixed
	case rest.ContentType_CONTENT_TYPE_TEMPLATE:
		return model.ContentTypeTemplate
	default:
		return model.ContentTypeText
	}
}

// convertContentStatusToProto 将内容状态转换为protobuf枚举
func convertContentStatusToProto(status string) rest.ContentStatus {
	switch status {
	case model.ContentStatusDraft:
		return rest.ContentStatus_CONTENT_STATUS_DRAFT
	case model.ContentStatusPending:
		return rest.ContentStatus_CONTENT_STATUS_PENDING
	case model.ContentStatusPublished:
		return rest.ContentStatus_CONTENT_STATUS_PUBLISHED
	case model.ContentStatusRejected:
		return rest.ContentStatus_CONTENT_STATUS_REJECTED
	case model.ContentStatusDeleted:
		return rest.ContentStatus_CONTENT_STATUS_DELETED
	default:
		return rest.ContentStatus_CONTENT_STATUS_UNSPECIFIED
	}
}

// convertContentStatusFromProto 将protobuf枚举转换为内容状态
func convertContentStatusFromProto(status rest.ContentStatus) string {
	switch status {
	case rest.ContentStatus_CONTENT_STATUS_DRAFT:
		return model.ContentStatusDraft
	case rest.ContentStatus_CONTENT_STATUS_PENDING:
		return model.ContentStatusPending
	case rest.ContentStatus_CONTENT_STATUS_PUBLISHED:
		return model.ContentStatusPublished
	case rest.ContentStatus_CONTENT_STATUS_REJECTED:
		return model.ContentStatusRejected
	case rest.ContentStatus_CONTENT_STATUS_DELETED:
		return model.ContentStatusDeleted
	default:
		return ""
	}
}
