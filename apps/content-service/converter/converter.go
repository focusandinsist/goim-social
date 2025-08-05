package converter

import (
	"time"

	"goim-social/api/rest"
	"goim-social/apps/content-service/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// ContentModelToProto 将内容Model转换为Protobuf
func (c *Converter) ContentModelToProto(content *model.Content) *rest.Content {
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
		tags = append(tags, c.TagModelToProto(&tag))
	}

	// 转换话题
	var topics []*rest.ContentTopic
	for _, topic := range content.Topics {
		topics = append(topics, c.TopicModelToProto(&topic))
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
		Type:         c.ContentTypeToProto(content.Type),
		Status:       c.ContentStatusToProto(content.Status),
		MediaFiles:   mediaFiles,
		Tags:         tags,
		Topics:       topics,
		TemplateData: content.TemplateData,
		ViewCount:    content.ViewCount,
		CreatedAt:    content.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    content.UpdatedAt.Format(time.RFC3339),
		PublishedAt:  publishedAt,
	}
}

// ContentModelsToProto 将内容Model列表转换为Protobuf列表
func (c *Converter) ContentModelsToProto(contents []*model.Content) []*rest.Content {
	if contents == nil {
		return []*rest.Content{}
	}

	result := make([]*rest.Content, 0, len(contents))
	for _, content := range contents {
		if protoContent := c.ContentModelToProto(content); protoContent != nil {
			result = append(result, protoContent)
		}
	}
	return result
}

// TagModelToProto 将标签Model转换为Protobuf
func (c *Converter) TagModelToProto(tag *model.ContentTag) *rest.ContentTag {
	if tag == nil {
		return nil
	}
	return &rest.ContentTag{
		Id:         tag.ID,
		Name:       tag.Name,
		UsageCount: tag.UsageCount,
	}
}

// TagModelsToProto 将标签Model列表转换为Protobuf列表
func (c *Converter) TagModelsToProto(tags []*model.ContentTag) []*rest.ContentTag {
	if tags == nil {
		return []*rest.ContentTag{}
	}

	result := make([]*rest.ContentTag, 0, len(tags))
	for _, tag := range tags {
		if protoTag := c.TagModelToProto(tag); protoTag != nil {
			result = append(result, protoTag)
		}
	}
	return result
}

// TopicModelToProto 将话题Model转换为Protobuf
func (c *Converter) TopicModelToProto(topic *model.ContentTopic) *rest.ContentTopic {
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

// TopicModelsToProto 将话题Model列表转换为Protobuf列表
func (c *Converter) TopicModelsToProto(topics []*model.ContentTopic) []*rest.ContentTopic {
	if topics == nil {
		return []*rest.ContentTopic{}
	}

	result := make([]*rest.ContentTopic, 0, len(topics))
	for _, topic := range topics {
		if protoTopic := c.TopicModelToProto(topic); protoTopic != nil {
			result = append(result, protoTopic)
		}
	}
	return result
}

// 枚举转换方法

// ContentTypeToProto 将内容类型转换为protobuf枚举
func (c *Converter) ContentTypeToProto(contentType string) rest.ContentType {
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

// ContentTypeFromProto 将protobuf枚举转换为内容类型
func (c *Converter) ContentTypeFromProto(contentType rest.ContentType) string {
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

// ContentStatusToProto 将内容状态转换为protobuf枚举
func (c *Converter) ContentStatusToProto(status string) rest.ContentStatus {
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

// ContentStatusFromProto 将protobuf枚举转换为内容状态
func (c *Converter) ContentStatusFromProto(status rest.ContentStatus) string {
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

// MediaFileModelsToProto 将媒体文件Model列表转换为Protobuf列表
func (c *Converter) MediaFileModelsToProto(mediaFiles []model.ContentMediaFile) []*rest.MediaFile {
	result := make([]*rest.MediaFile, 0, len(mediaFiles))
	for _, mf := range mediaFiles {
		result = append(result, &rest.MediaFile{
			Url:      mf.URL,
			Filename: mf.Filename,
			Size:     mf.Size,
			MimeType: mf.MimeType,
			Width:    mf.Width,
			Height:   mf.Height,
			Duration: mf.Duration,
		})
	}
	return result
}

// MediaFileProtoToModels 将Protobuf媒体文件列表转换为Model列表
func (c *Converter) MediaFileProtoToModels(mediaFiles []*rest.MediaFile) []model.ContentMediaFile {
	result := make([]model.ContentMediaFile, 0, len(mediaFiles))
	for _, mf := range mediaFiles {
		result = append(result, model.ContentMediaFile{
			URL:      mf.Url,
			Filename: mf.Filename,
			Size:     mf.Size,
			MimeType: mf.MimeType,
			Width:    mf.Width,
			Height:   mf.Height,
			Duration: mf.Duration,
		})
	}
	return result
}

// 响应构建方法

// BuildCreateContentResponse 构建创建内容响应
func (c *Converter) BuildCreateContentResponse(success bool, message string, content *model.Content) *rest.CreateContentResponse {
	return &rest.CreateContentResponse{
		Success: success,
		Message: message,
		Content: c.ContentModelToProto(content),
	}
}

// BuildUpdateContentResponse 构建更新内容响应
func (c *Converter) BuildUpdateContentResponse(success bool, message string, content *model.Content) *rest.UpdateContentResponse {
	return &rest.UpdateContentResponse{
		Success: success,
		Message: message,
		Content: c.ContentModelToProto(content),
	}
}

// BuildGetContentResponse 构建获取内容响应
func (c *Converter) BuildGetContentResponse(success bool, message string, content *model.Content) *rest.GetContentResponse {
	return &rest.GetContentResponse{
		Success: success,
		Message: message,
		Content: c.ContentModelToProto(content),
	}
}

// BuildDeleteContentResponse 构建删除内容响应
func (c *Converter) BuildDeleteContentResponse(success bool, message string) *rest.DeleteContentResponse {
	return &rest.DeleteContentResponse{
		Success: success,
		Message: message,
	}
}

// BuildPublishContentResponse 构建发布内容响应
func (c *Converter) BuildPublishContentResponse(success bool, message string, content *model.Content) *rest.PublishContentResponse {
	return &rest.PublishContentResponse{
		Success: success,
		Message: message,
		Content: c.ContentModelToProto(content),
	}
}

// BuildChangeContentStatusResponse 构建变更内容状态响应
func (c *Converter) BuildChangeContentStatusResponse(success bool, message string, content *model.Content) *rest.ChangeContentStatusResponse {
	return &rest.ChangeContentStatusResponse{
		Success: success,
		Message: message,
		Content: c.ContentModelToProto(content),
	}
}

// BuildSearchContentResponse 构建搜索内容响应
func (c *Converter) BuildSearchContentResponse(success bool, message string, contents []*model.Content, total int64, page, pageSize int32) *rest.SearchContentResponse {
	return &rest.SearchContentResponse{
		Success:  success,
		Message:  message,
		Contents: c.ContentModelsToProto(contents),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildGetUserContentResponse 构建获取用户内容响应
func (c *Converter) BuildGetUserContentResponse(success bool, message string, contents []*model.Content, total int64, page, pageSize int32) *rest.GetUserContentResponse {
	return &rest.GetUserContentResponse{
		Success:  success,
		Message:  message,
		Contents: c.ContentModelsToProto(contents),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildGetContentStatsResponse 构建获取内容统计响应
func (c *Converter) BuildGetContentStatsResponse(success bool, message string, stats *model.ContentStats) *rest.GetContentStatsResponse {
	if !success || stats == nil {
		return &rest.GetContentStatsResponse{
			Success:           success,
			Message:           message,
			TotalContents:     0,
			PublishedContents: 0,
			DraftContents:     0,
			PendingContents:   0,
			TotalViews:        0,
			TotalLikes:        0,
		}
	}

	return &rest.GetContentStatsResponse{
		Success:           success,
		Message:           message,
		TotalContents:     stats.TotalContents,
		PublishedContents: stats.PublishedContents,
		DraftContents:     stats.DraftContents,
		PendingContents:   stats.PendingContents,
		TotalViews:        stats.TotalViews,
		TotalLikes:        stats.TotalLikes,
	}
}

// BuildCreateTagResponse 构建创建标签响应
func (c *Converter) BuildCreateTagResponse(success bool, message string, tag *model.ContentTag) *rest.CreateTagResponse {
	return &rest.CreateTagResponse{
		Success: success,
		Message: message,
		Tag:     c.TagModelToProto(tag),
	}
}

// BuildGetTagsResponse 构建获取标签列表响应
func (c *Converter) BuildGetTagsResponse(success bool, message string, tags []*model.ContentTag, total int64) *rest.GetTagsResponse {
	return &rest.GetTagsResponse{
		Success: success,
		Message: message,
		Tags:    c.TagModelsToProto(tags),
		Total:   total,
	}
}

// BuildCreateTopicResponse 构建创建话题响应
func (c *Converter) BuildCreateTopicResponse(success bool, message string, topic *model.ContentTopic) *rest.CreateTopicResponse {
	return &rest.CreateTopicResponse{
		Success: success,
		Message: message,
		Topic:   c.TopicModelToProto(topic),
	}
}

// BuildGetTopicsResponse 构建获取话题列表响应
func (c *Converter) BuildGetTopicsResponse(success bool, message string, topics []*model.ContentTopic, total int64) *rest.GetTopicsResponse {
	return &rest.GetTopicsResponse{
		Success: success,
		Message: message,
		Topics:  c.TopicModelsToProto(topics),
		Total:   total,
	}
}

// 便捷方法：构建错误响应

// BuildErrorCreateContentResponse 构建创建内容错误响应
func (c *Converter) BuildErrorCreateContentResponse(message string) *rest.CreateContentResponse {
	return c.BuildCreateContentResponse(false, message, nil)
}

// BuildErrorUpdateContentResponse 构建更新内容错误响应
func (c *Converter) BuildErrorUpdateContentResponse(message string) *rest.UpdateContentResponse {
	return c.BuildUpdateContentResponse(false, message, nil)
}

// BuildErrorGetContentResponse 构建获取内容错误响应
func (c *Converter) BuildErrorGetContentResponse(message string) *rest.GetContentResponse {
	return c.BuildGetContentResponse(false, message, nil)
}

// BuildErrorDeleteContentResponse 构建删除内容错误响应
func (c *Converter) BuildErrorDeleteContentResponse(message string) *rest.DeleteContentResponse {
	return c.BuildDeleteContentResponse(false, message)
}

// BuildErrorPublishContentResponse 构建发布内容错误响应
func (c *Converter) BuildErrorPublishContentResponse(message string) *rest.PublishContentResponse {
	return c.BuildPublishContentResponse(false, message, nil)
}

// BuildErrorChangeContentStatusResponse 构建变更内容状态错误响应
func (c *Converter) BuildErrorChangeContentStatusResponse(message string) *rest.ChangeContentStatusResponse {
	return c.BuildChangeContentStatusResponse(false, message, nil)
}

// BuildErrorSearchContentResponse 构建搜索内容错误响应
func (c *Converter) BuildErrorSearchContentResponse(message string) *rest.SearchContentResponse {
	return c.BuildSearchContentResponse(false, message, nil, 0, 0, 0)
}

// BuildErrorGetUserContentResponse 构建获取用户内容错误响应
func (c *Converter) BuildErrorGetUserContentResponse(message string) *rest.GetUserContentResponse {
	return c.BuildGetUserContentResponse(false, message, nil, 0, 0, 0)
}

// BuildErrorGetContentStatsResponse 构建获取内容统计错误响应
func (c *Converter) BuildErrorGetContentStatsResponse(message string) *rest.GetContentStatsResponse {
	return c.BuildGetContentStatsResponse(false, message, nil)
}

// BuildErrorCreateTagResponse 构建创建标签错误响应
func (c *Converter) BuildErrorCreateTagResponse(message string) *rest.CreateTagResponse {
	return c.BuildCreateTagResponse(false, message, nil)
}

// BuildErrorGetTagsResponse 构建获取标签列表错误响应
func (c *Converter) BuildErrorGetTagsResponse(message string) *rest.GetTagsResponse {
	return c.BuildGetTagsResponse(false, message, nil, 0)
}

// BuildErrorCreateTopicResponse 构建创建话题错误响应
func (c *Converter) BuildErrorCreateTopicResponse(message string) *rest.CreateTopicResponse {
	return c.BuildCreateTopicResponse(false, message, nil)
}

// BuildErrorGetTopicsResponse 构建获取话题列表错误响应
func (c *Converter) BuildErrorGetTopicsResponse(message string) *rest.GetTopicsResponse {
	return c.BuildGetTopicsResponse(false, message, nil, 0)
}
