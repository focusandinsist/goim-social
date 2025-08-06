package service

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/apps/content-service/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
	"goim-social/pkg/telemetry"
)

// ==================== 聚合查询相关业务逻辑 ====================

// GetContentDetail 获取内容详情（包含评论和互动）
func (s *Service) GetContentDetail(ctx context.Context, contentID, userID int64, commentLimit int32) (*model.ContentDetailResult, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.GetContentDetail")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("content.id", contentID),
		attribute.Int64("content.user_id", userID),
		attribute.Int32("content.comment_limit", commentLimit),
	)

	// 将业务信息添加到context
	if userID > 0 {
		ctx = tracecontext.WithUserID(ctx, userID)
	}

	// 参数验证
	if contentID <= 0 {
		span.SetStatus(codes.Error, "invalid content ID")
		return nil, fmt.Errorf("内容ID无效")
	}

	if commentLimit <= 0 {
		commentLimit = 10 // 默认返回10条评论
	}

	// 聚合查询内容详情
	content, comments, stats, userInteractions, err := s.dao.GetContentWithDetails(ctx, contentID, userID, commentLimit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get content details")
		return nil, fmt.Errorf("获取内容详情失败: %v", err)
	}

	// 增加浏览次数（异步执行，不影响主流程）
	go func() {
		if err := s.dao.IncrementViewCount(context.Background(), contentID); err != nil {
			s.logger.Error(context.Background(), "Failed to increment view count",
				logger.F("contentID", contentID),
				logger.F("error", err.Error()))
		}
	}()

	result := &model.ContentDetailResult{
		Content:          content,
		TopComments:      comments,
		InteractionStats: stats,
		UserInteractions: userInteractions,
	}

	span.SetAttributes(
		attribute.String("content.title", content.Title),
		attribute.Int("content.comment_count", len(comments)),
		attribute.Int64("content.like_count", stats.LikeCount),
	)

	s.logger.Info(ctx, "Content detail retrieved successfully",
		logger.F("contentID", contentID),
		logger.F("userID", userID),
		logger.F("commentCount", len(comments)))

	span.SetStatus(codes.Ok, "content detail retrieved successfully")
	return result, nil
}

// GetContentFeed 获取内容流
func (s *Service) GetContentFeed(ctx context.Context, userID int64, contentType, sortBy string, page, pageSize int32) ([]*model.ContentFeedItem, int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.GetContentFeed")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("feed.user_id", userID),
		attribute.String("feed.content_type", contentType),
		attribute.String("feed.sort_by", sortBy),
		attribute.Int32("feed.page", page),
		attribute.Int32("feed.page_size", pageSize),
	)

	// 将业务信息添加到context
	if userID > 0 {
		ctx = tracecontext.WithUserID(ctx, userID)
	}

	// 参数验证和默认值设置
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > model.MaxBatchSize {
		pageSize = model.DefaultPageSize
	}
	if sortBy == "" {
		sortBy = "time"
	}

	// 获取内容流数据
	contents, stats, userInteractionsMap, err := s.dao.GetContentFeed(ctx, userID, contentType, sortBy, page, pageSize)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get content feed")
		return nil, 0, fmt.Errorf("获取内容流失败: %v", err)
	}

	// 构建内容流项目
	feedItems := make([]*model.ContentFeedItem, len(contents))
	for i, content := range contents {
		// 查找对应的统计数据
		var contentStats *model.InteractionStats
		for _, stat := range stats {
			if stat.TargetID == content.ID && stat.TargetType == model.TargetTypeContent {
				contentStats = stat
				break
			}
		}
		if contentStats == nil {
			contentStats = &model.InteractionStats{
				TargetID:   content.ID,
				TargetType: model.TargetTypeContent,
			}
		}

		// 获取用户互动状态
		var userInteractions map[string]bool
		if userInteractionsMap != nil {
			userInteractions = userInteractionsMap[content.ID]
		}
		if userInteractions == nil {
			userInteractions = make(map[string]bool)
		}

		feedItems[i] = &model.ContentFeedItem{
			Content:          content,
			InteractionStats: contentStats,
			UserInteractions: userInteractions,
			CommentPreview:   3, // 预览3条评论
		}
	}

	// 计算总数（简化实现，实际应该从数据库查询）
	total := int64(len(contents))
	if len(contents) == int(pageSize) {
		total = int64(page * pageSize) // 估算值
	}

	span.SetAttributes(
		attribute.Int("feed.item_count", len(feedItems)),
		attribute.Int64("feed.total", total),
	)

	s.logger.Info(ctx, "Content feed retrieved successfully",
		logger.F("userID", userID),
		logger.F("contentType", contentType),
		logger.F("sortBy", sortBy),
		logger.F("itemCount", len(feedItems)))

	span.SetStatus(codes.Ok, "content feed retrieved successfully")
	return feedItems, total, nil
}

// GetTrendingContent 获取热门内容
func (s *Service) GetTrendingContent(ctx context.Context, timeRange, contentType string, limit int32) ([]*model.ContentFeedItem, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.GetTrendingContent")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.String("trending.time_range", timeRange),
		attribute.String("trending.content_type", contentType),
		attribute.Int32("trending.limit", limit),
	)

	// 参数验证和默认值设置
	if limit <= 0 || limit > model.MaxBatchSize {
		limit = 20 // 默认返回20条热门内容
	}
	if timeRange == "" {
		timeRange = "day" // 默认一天内的热门内容
	}

	// 获取热门内容数据
	contents, stats, err := s.dao.GetTrendingContent(ctx, timeRange, contentType, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get trending content")
		return nil, fmt.Errorf("获取热门内容失败: %v", err)
	}

	// 构建内容流项目
	feedItems := make([]*model.ContentFeedItem, len(contents))
	for i, content := range contents {
		// 查找对应的统计数据
		var contentStats *model.InteractionStats
		for _, stat := range stats {
			if stat.TargetID == content.ID && stat.TargetType == model.TargetTypeContent {
				contentStats = stat
				break
			}
		}
		if contentStats == nil {
			contentStats = &model.InteractionStats{
				TargetID:   content.ID,
				TargetType: model.TargetTypeContent,
			}
		}

		feedItems[i] = &model.ContentFeedItem{
			Content:          content,
			InteractionStats: contentStats,
			UserInteractions: make(map[string]bool), // 热门内容不返回用户互动状态
			CommentPreview:   0,                     // 热门内容不预览评论
		}
	}

	span.SetAttributes(attribute.Int("trending.item_count", len(feedItems)))

	s.logger.Info(ctx, "Trending content retrieved successfully",
		logger.F("timeRange", timeRange),
		logger.F("contentType", contentType),
		logger.F("itemCount", len(feedItems)))

	span.SetStatus(codes.Ok, "trending content retrieved successfully")
	return feedItems, nil
}

// DeleteContentWithRelated 删除内容及其相关数据
func (s *Service) DeleteContentWithRelated(ctx context.Context, contentID, userID int64) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.DeleteContentWithRelated")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("content.id", contentID),
		attribute.Int64("content.user_id", userID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 参数验证
	if contentID <= 0 {
		span.SetStatus(codes.Error, "invalid content ID")
		return fmt.Errorf("内容ID无效")
	}

	// 获取内容信息进行权限检查
	content, err := s.dao.GetContent(ctx, contentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "content not found")
		return fmt.Errorf("内容不存在")
	}

	// 权限检查（只能删除自己的内容）
	if content.AuthorID != userID {
		span.SetStatus(codes.Error, "permission denied")
		return fmt.Errorf("无权限删除此内容")
	}

	// 执行级联删除
	if err := s.dao.DeleteContentWithRelated(ctx, contentID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete content with related data")
		return fmt.Errorf("删除内容失败: %v", err)
	}

	// 清除相关缓存
	go s.clearContentCache(context.Background(), contentID)

	// 发送事件到消息队列
	go s.publishContentEvent(context.Background(), "delete", content)

	s.logger.Info(ctx, "Content with related data deleted successfully",
		logger.F("contentID", contentID),
		logger.F("userID", userID),
		logger.F("title", content.Title))

	span.SetStatus(codes.Ok, "content with related data deleted successfully")
	return nil
}
