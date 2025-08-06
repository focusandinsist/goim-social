package service

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/apps/content-service/dao"
	"goim-social/apps/content-service/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/telemetry"
)

// Service 内容服务
type Service struct {
	dao    dao.ContentDAO
	redis  *redis.RedisClient
	kafka  *kafka.Producer
	logger logger.Logger
}

// NewService 创建内容服务实例
func NewService(contentDAO dao.ContentDAO, redis *redis.RedisClient, kafka *kafka.Producer, log logger.Logger) *Service {
	return &Service{
		dao:    contentDAO,
		redis:  redis,
		kafka:  kafka,
		logger: log,
	}
}

// CreateContent 创建内容
func (s *Service) CreateContent(ctx context.Context, authorID int64,
	title, content, contentType string, mediaFiles []model.ContentMediaFile,
	tagIDs, topicIDs []int64, templateData string, saveAsDraft bool) (*model.Content, error) {

	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.CreateContent")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("content.author_id", authorID),
		attribute.String("content.title", title),
		attribute.String("content.type", contentType),
		attribute.Bool("content.save_as_draft", saveAsDraft),
		attribute.Int("content.media_files_count", len(mediaFiles)),
		attribute.Int("content.tag_ids_count", len(tagIDs)),
		attribute.Int("content.topic_ids_count", len(topicIDs)),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, authorID)

	if authorID <= 0 {
		span.SetStatus(codes.Error, "invalid author ID")
		return nil, fmt.Errorf("作者ID无效")
	}
	if title == "" {
		span.SetStatus(codes.Error, "title is empty")
		return nil, fmt.Errorf("标题不能为空")
	}
	if !model.ValidateContentType(contentType) {
		span.SetStatus(codes.Error, "invalid content type")
		return nil, fmt.Errorf("内容类型无效")
	}

	// 确定初始状态
	status := model.ContentStatusPending
	if saveAsDraft {
		status = model.ContentStatusDraft
	}

	// 创建内容对象
	newContent := &model.Content{
		AuthorID:     authorID,
		Title:        title,
		Content:      content,
		Type:         contentType,
		Status:       status,
		TemplateData: templateData,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 如果直接发布，设置发布时间
	if status == model.ContentStatusPublished {
		now := time.Now()
		newContent.PublishedAt = &now
	}

	if err := s.dao.CreateContent(ctx, newContent); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create content")
		return nil, fmt.Errorf("创建内容失败: %v", err)
	}

	// 设置内容ID到context和span
	ctx = tracecontext.WithContentID(ctx, newContent.ID)
	span.SetAttributes(attribute.Int64("content.id", newContent.ID))

	// 添加媒体文件
	if len(mediaFiles) > 0 {
		for i, mediaFile := range mediaFiles {
			mediaFile.ContentID = newContent.ID
			mediaFile.SortOrder = int32(i)
			if err := s.dao.CreateMediaFile(ctx, &mediaFile); err != nil {
				s.logger.Error(ctx, "Failed to create media file",
					logger.F("contentID", newContent.ID),
					logger.F("error", err.Error()))
			}
		}
	}

	// 添加标签关联
	if len(tagIDs) > 0 {
		if err := s.dao.AddContentTags(ctx, newContent.ID, tagIDs); err != nil {
			s.logger.Error(ctx, "Failed to add content tags",
				logger.F("contentID", newContent.ID),
				logger.F("error", err.Error()))
		}
	}

	// 添加话题关联
	if len(topicIDs) > 0 {
		if err := s.dao.AddContentTopics(ctx, newContent.ID, topicIDs); err != nil {
			s.logger.Error(ctx, "Failed to add content topics",
				logger.F("contentID", newContent.ID),
				logger.F("error", err.Error()))
		}
	}

	// 记录状态变更日志
	statusLog := &model.ContentStatusLog{
		ContentID:  newContent.ID,
		FromStatus: "",
		ToStatus:   status,
		OperatorID: authorID,
		Reason:     "内容创建",
		CreatedAt:  time.Now(),
	}
	if err := s.dao.CreateStatusLog(ctx, statusLog); err != nil {
		s.logger.Error(ctx, "Failed to create status log",
			logger.F("contentID", newContent.ID),
			logger.F("error", err.Error()))
	}

	// 获取完整的内容信息
	fullContent, err := s.dao.GetContentWithRelations(ctx, newContent.ID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get full content after creation",
			logger.F("contentID", newContent.ID),
			logger.F("error", err.Error()))
		span.SetStatus(codes.Ok, "content created but failed to get full content")
		return newContent, nil
	}

	s.logger.Info(ctx, "Content created successfully",
		logger.F("contentID", newContent.ID),
		logger.F("title", title),
		logger.F("authorID", authorID),
		logger.F("status", status))

	span.SetStatus(codes.Ok, "content created successfully")
	return fullContent, nil
}

// UpdateContent 更新内容
func (s *Service) UpdateContent(ctx context.Context, contentID, authorID int64, title, content, contentType string,
	mediaFiles []model.ContentMediaFile, tagIDs, topicIDs []int64, templateData string) (*model.Content, error) {

	// 获取现有内容
	existingContent, err := s.dao.GetContent(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("内容不存在: %v", err)
	}

	// 权限验证
	if existingContent.AuthorID != authorID {
		return nil, fmt.Errorf("无权限修改此内容")
	}

	// 状态验证：只有草稿和被拒绝的内容可以修改
	if existingContent.Status != model.ContentStatusDraft && existingContent.Status != model.ContentStatusRejected {
		return nil, fmt.Errorf("当前状态的内容不允许修改")
	}

	// 验证参数
	if title == "" {
		return nil, fmt.Errorf("标题不能为空")
	}
	if !model.ValidateContentType(contentType) {
		return nil, fmt.Errorf("内容类型无效")
	}

	// 更新内容字段
	existingContent.Title = title
	existingContent.Content = content
	existingContent.Type = contentType
	existingContent.TemplateData = templateData
	existingContent.UpdatedAt = time.Now()

	// 如果是被拒绝的内容，修改后重新设为草稿状态
	if existingContent.Status == model.ContentStatusRejected {
		existingContent.Status = model.ContentStatusDraft
	}

	// 更新内容
	if err := s.dao.UpdateContent(ctx, existingContent); err != nil {
		return nil, fmt.Errorf("更新内容失败: %v", err)
	}

	// 更新媒体文件
	if err := s.dao.DeleteMediaFiles(ctx, contentID); err != nil {
		s.logger.Error(ctx, "Failed to delete old media files",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
	}

	for i, mediaFile := range mediaFiles {
		mediaFile.ContentID = contentID
		mediaFile.SortOrder = int32(i)
		if err := s.dao.CreateMediaFile(ctx, &mediaFile); err != nil {
			s.logger.Error(ctx, "Failed to create media file",
				logger.F("contentID", contentID),
				logger.F("error", err.Error()))
		}
	}

	// 更新标签关联
	if err := s.dao.RemoveContentTags(ctx, contentID); err != nil {
		s.logger.Error(ctx, "Failed to remove old content tags",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
	}
	if len(tagIDs) > 0 {
		if err := s.dao.AddContentTags(ctx, contentID, tagIDs); err != nil {
			s.logger.Error(ctx, "Failed to add content tags",
				logger.F("contentID", contentID),
				logger.F("error", err.Error()))
		}
	}

	// 更新话题关联
	if err := s.dao.RemoveContentTopics(ctx, contentID); err != nil {
		s.logger.Error(ctx, "Failed to remove old content topics",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
	}
	if len(topicIDs) > 0 {
		if err := s.dao.AddContentTopics(ctx, contentID, topicIDs); err != nil {
			s.logger.Error(ctx, "Failed to add content topics",
				logger.F("contentID", contentID),
				logger.F("error", err.Error()))
		}
	}

	// 获取完整的内容信息
	fullContent, err := s.dao.GetContentWithRelations(ctx, contentID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get full content after update",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
		return existingContent, nil
	}

	return fullContent, nil
}

// GetContent 获取内容详情
func (s *Service) GetContent(ctx context.Context, contentID, userID int64) (*model.Content, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.GetContent")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("content.id", contentID),
		attribute.Int64("user.id", userID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)
	ctx = tracecontext.WithContentID(ctx, contentID)

	if contentID <= 0 {
		span.SetStatus(codes.Error, "invalid content ID")
		return nil, fmt.Errorf("内容ID无效")
	}

	// 获取内容
	content, err := s.dao.GetContentWithRelations(ctx, contentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "content not found")
		return nil, fmt.Errorf("内容不存在: %v", err)
	}

	span.SetAttributes(
		attribute.String("content.title", content.Title),
		attribute.String("content.status", content.Status),
		attribute.Int64("content.author_id", content.AuthorID),
	)

	// 权限检查：只有作者可以查看草稿和被拒绝的内容
	if (content.Status == model.ContentStatusDraft || content.Status == model.ContentStatusRejected) &&
		content.AuthorID != userID {
		span.SetStatus(codes.Error, "permission denied for draft/rejected content")
		return nil, fmt.Errorf("无权限查看此内容")
	}

	// 只有已发布的内容对所有人可见
	if content.Status != model.ContentStatusPublished && content.AuthorID != userID {
		span.SetStatus(codes.Error, "content not accessible")
		return nil, fmt.Errorf("内容不可访问")
	}

	// 增加浏览次数（异步执行，不影响主流程）
	go func() {
		if err := s.dao.IncrementViewCount(context.Background(), contentID); err != nil {
			s.logger.Error(context.Background(), "Failed to increment view count",
				logger.F("contentID", contentID),
				logger.F("error", err.Error()))
		}
	}()

	s.logger.Info(ctx, "Content retrieved successfully",
		logger.F("contentID", contentID),
		logger.F("userID", userID),
		logger.F("contentTitle", content.Title))

	span.SetStatus(codes.Ok, "content retrieved successfully")
	return content, nil
}

// DeleteContent 删除内容
func (s *Service) DeleteContent(ctx context.Context, contentID, authorID int64) error {
	// 获取内容
	content, err := s.dao.GetContent(ctx, contentID)
	if err != nil {
		return fmt.Errorf("内容不存在: %v", err)
	}

	// 权限验证
	if content.AuthorID != authorID {
		return fmt.Errorf("无权限删除此内容")
	}

	// 记录状态变更日志
	statusLog := &model.ContentStatusLog{
		ContentID:  contentID,
		FromStatus: content.Status,
		ToStatus:   model.ContentStatusDeleted,
		OperatorID: authorID,
		Reason:     "用户删除",
		CreatedAt:  time.Now(),
	}
	if err := s.dao.CreateStatusLog(ctx, statusLog); err != nil {
		s.logger.Error(ctx, "Failed to create status log",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
	}

	// 删除内容（级联删除关联数据）
	return s.dao.DeleteContent(ctx, contentID)
}

// PublishContent 发布内容
func (s *Service) PublishContent(ctx context.Context, contentID, authorID int64) (*model.Content, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.PublishContent")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("content.id", contentID),
		attribute.Int64("author.id", authorID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, authorID)
	ctx = tracecontext.WithContentID(ctx, contentID)

	// 获取内容
	content, err := s.dao.GetContent(ctx, contentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "content not found")
		return nil, fmt.Errorf("内容不存在: %v", err)
	}

	span.SetAttributes(
		attribute.String("content.title", content.Title),
		attribute.String("content.current_status", content.Status),
	)

	// 权限验证
	if content.AuthorID != authorID {
		span.SetStatus(codes.Error, "permission denied")
		return nil, fmt.Errorf("无权限发布此内容")
	}

	// 状态验证
	if !model.CanTransitionStatus(content.Status, model.ContentStatusPublished) {
		span.SetStatus(codes.Error, "invalid status transition")
		return nil, fmt.Errorf("当前状态不允许发布")
	}

	// 更新状态和发布时间
	oldStatus := content.Status
	content.Status = model.ContentStatusPublished
	now := time.Now()
	content.PublishedAt = &now
	content.UpdatedAt = now

	if err := s.dao.UpdateContent(ctx, content); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update content")
		return nil, fmt.Errorf("发布内容失败: %v", err)
	}

	// 记录状态变更日志
	statusLog := &model.ContentStatusLog{
		ContentID:  contentID,
		FromStatus: oldStatus,
		ToStatus:   model.ContentStatusPublished,
		OperatorID: authorID,
		Reason:     "用户发布",
		CreatedAt:  time.Now(),
	}
	if err := s.dao.CreateStatusLog(ctx, statusLog); err != nil {
		s.logger.Error(ctx, "Failed to create status log",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
	}

	// 获取完整内容信息
	fullContent, err := s.dao.GetContentWithRelations(ctx, contentID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get full content after publish",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
		span.SetStatus(codes.Ok, "content published but failed to get full content")
		return content, nil
	}

	s.logger.Info(ctx, "Content published successfully",
		logger.F("contentID", contentID),
		logger.F("authorID", authorID),
		logger.F("title", content.Title))

	span.SetStatus(codes.Ok, "content published successfully")
	return fullContent, nil
}

// ChangeContentStatus 变更内容状态（管理员操作）
func (s *Service) ChangeContentStatus(ctx context.Context, contentID, operatorID int64, newStatus, reason string) (*model.Content, error) {
	// 验证新状态
	if !model.ValidateContentStatus(newStatus) {
		return nil, fmt.Errorf("无效的状态")
	}

	// 获取内容
	content, err := s.dao.GetContent(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("内容不存在: %v", err)
	}

	// 状态转换验证
	if !model.CanTransitionStatus(content.Status, newStatus) {
		return nil, fmt.Errorf("不允许从 %s 转换到 %s", content.Status, newStatus)
	}

	// 更新状态
	oldStatus := content.Status
	content.Status = newStatus
	content.UpdatedAt = time.Now()

	// 如果是发布状态，设置发布时间
	if newStatus == model.ContentStatusPublished && content.PublishedAt == nil {
		now := time.Now()
		content.PublishedAt = &now
	}

	if err := s.dao.UpdateContent(ctx, content); err != nil {
		return nil, fmt.Errorf("更新内容状态失败: %v", err)
	}

	// 记录状态变更日志
	statusLog := &model.ContentStatusLog{
		ContentID:  contentID,
		FromStatus: oldStatus,
		ToStatus:   newStatus,
		OperatorID: operatorID,
		Reason:     reason,
		CreatedAt:  time.Now(),
	}
	if err := s.dao.CreateStatusLog(ctx, statusLog); err != nil {
		s.logger.Error(ctx, "Failed to create status log",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
	}

	// 获取完整内容信息
	fullContent, err := s.dao.GetContentWithRelations(ctx, contentID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get full content after status change",
			logger.F("contentID", contentID),
			logger.F("error", err.Error()))
		return content, nil
	}

	return fullContent, nil
}

// SearchContent 搜索内容
func (s *Service) SearchContent(ctx context.Context, params *model.SearchContentParams) ([]*model.Content, int64, error) {
	// 参数验证和默认值设置
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}
	if params.SortBy == "" {
		params.SortBy = model.SortByCreatedAt
	}
	if params.SortOrder == "" {
		params.SortOrder = model.SortOrderDesc
	}

	// 如果没有指定状态，默认只搜索已发布的内容
	if params.Status == "" && params.AuthorID == 0 {
		params.Status = model.ContentStatusPublished
	}

	return s.dao.SearchContents(ctx, params)
}

// GetUserContent 获取用户内容列表
func (s *Service) GetUserContent(ctx context.Context, authorID int64, status string, page, pageSize int32) ([]*model.Content, int64, error) {
	if authorID <= 0 {
		return nil, 0, fmt.Errorf("用户ID无效")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	return s.dao.GetUserContents(ctx, authorID, status, page, pageSize)
}

// GetContentStats 获取内容统计
func (s *Service) GetContentStats(ctx context.Context, authorID int64) (*model.ContentStats, error) {
	return s.dao.GetContentStats(ctx, authorID)
}

// CreateTag 创建标签
func (s *Service) CreateTag(ctx context.Context, name string) (*model.ContentTag, error) {
	if name == "" {
		return nil, fmt.Errorf("标签名称不能为空")
	}

	// 检查标签是否已存在
	existingTag, err := s.dao.GetTagByName(ctx, name)
	if err == nil && existingTag != nil {
		return existingTag, nil
	}

	tag := &model.ContentTag{
		Name:       name,
		UsageCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.dao.CreateTag(ctx, tag); err != nil {
		return nil, fmt.Errorf("创建标签失败: %v", err)
	}

	return tag, nil
}

// GetTags 获取标签列表
func (s *Service) GetTags(ctx context.Context, keyword string, page, pageSize int32) ([]*model.ContentTag, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	return s.dao.GetTags(ctx, keyword, page, pageSize)
}

// CreateTopic 创建话题
func (s *Service) CreateTopic(ctx context.Context, name, description, coverImage string) (*model.ContentTopic, error) {
	if name == "" {
		return nil, fmt.Errorf("话题名称不能为空")
	}

	// 检查话题是否已存在
	existingTopic, err := s.dao.GetTopicByName(ctx, name)
	if err == nil && existingTopic != nil {
		return existingTopic, nil
	}

	topic := &model.ContentTopic{
		Name:         name,
		Description:  description,
		CoverImage:   coverImage,
		ContentCount: 0,
		IsHot:        false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.dao.CreateTopic(ctx, topic); err != nil {
		return nil, fmt.Errorf("创建话题失败: %v", err)
	}

	return topic, nil
}

// GetTopics 获取话题列表
func (s *Service) GetTopics(ctx context.Context, keyword string, hotOnly bool, page, pageSize int32) ([]*model.ContentTopic, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	return s.dao.GetTopics(ctx, keyword, hotOnly, page, pageSize)
}
