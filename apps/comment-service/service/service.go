package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/api/rest"
	"goim-social/apps/comment-service/dao"
	"goim-social/apps/comment-service/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/telemetry"
)

// Service 评论服务
type Service struct {
	dao      dao.CommentDAO
	redis    *redis.RedisClient
	producer *kafka.Producer
	logger   logger.Logger
}

// NewService 创建评论服务实例
func NewService(dao dao.CommentDAO, redis *redis.RedisClient, kafka *kafka.Producer, logger logger.Logger) *Service {
	return &Service{
		dao:      dao,
		redis:    redis,
		producer: kafka,
		logger:   logger,
	}
}

// CreateComment 创建评论
func (s *Service) CreateComment(ctx context.Context, params *model.CreateCommentParams) (*model.Comment, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "comment.service.CreateComment")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("comment.object_id", params.ObjectID),
		attribute.String("comment.object_type", params.ObjectType),
		attribute.Int64("comment.user_id", params.UserID),
		attribute.String("comment.user_name", params.UserName),
		attribute.Int64("comment.parent_id", params.ParentID),
		attribute.Int("comment.content_length", len(params.Content)),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, params.UserID)

	// 参数验证
	if err := s.validateCreateCommentParams(params); err != nil {
		span.SetStatus(codes.Error, "invalid parameters")
		return nil, err
	}

	// 构建评论对象
	comment := &model.Comment{
		ObjectID:        params.ObjectID,
		ObjectType:      params.ObjectType,
		UserID:          params.UserID,
		UserName:        params.UserName,
		UserAvatar:      params.UserAvatar,
		Content:         strings.TrimSpace(params.Content),
		ParentID:        params.ParentID,
		ReplyToUserID:   params.ReplyToUserID,
		ReplyToUserName: params.ReplyToUserName,
		Status:          model.CommentStatusPending, // 默认待审核
		IPAddress:       params.IPAddress,
		UserAgent:       params.UserAgent,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 处理回复逻辑
	if params.ParentID > 0 {
		parentComment, err := s.dao.GetComment(ctx, params.ParentID)
		if err != nil {
			return nil, fmt.Errorf("父评论不存在: %v", err)
		}

		// 设置根评论ID
		if parentComment.RootID > 0 {
			comment.RootID = parentComment.RootID
		} else {
			comment.RootID = parentComment.ID
		}

		// 检查回复层级
		if comment.GetDepth() >= model.MaxReplyDepth {
			return nil, fmt.Errorf("回复层级过深，最多支持%d层", model.MaxReplyDepth)
		}
	}

	// 创建评论
	if err := s.dao.CreateComment(ctx, comment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create comment")
		return nil, fmt.Errorf("创建评论失败: %v", err)
	}

	// 设置评论ID到span
	span.SetAttributes(attribute.Int64("comment.id", comment.ID))

	// 清除相关缓存
	s.clearCommentCache(ctx, comment.ObjectID, comment.ObjectType)

	// 发送事件
	s.publishEvent(ctx, model.EventCommentCreated, comment)

	s.logger.Info(ctx, "Comment created successfully",
		logger.F("commentID", comment.ID),
		logger.F("userID", comment.UserID),
		logger.F("objectID", comment.ObjectID))

	span.SetStatus(codes.Ok, "comment created successfully")
	return comment, nil
}

// UpdateComment 更新评论
func (s *Service) UpdateComment(ctx context.Context, params *model.UpdateCommentParams) (*model.Comment, error) {
	// 参数验证
	if params.CommentID <= 0 {
		return nil, fmt.Errorf("评论ID无效")
	}
	if params.UserID <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}
	if len(strings.TrimSpace(params.Content)) < model.MinCommentLength {
		return nil, fmt.Errorf("评论内容不能为空")
	}
	if len(params.Content) > model.MaxCommentLength {
		return nil, fmt.Errorf("评论内容过长，最多%d个字符", model.MaxCommentLength)
	}

	// 获取原评论
	comment, err := s.dao.GetComment(ctx, params.CommentID)
	if err != nil {
		return nil, fmt.Errorf("评论不存在: %v", err)
	}

	// 权限检查
	if !comment.CanEdit(params.UserID, false) {
		return nil, fmt.Errorf("无权限编辑此评论")
	}

	// 更新评论内容
	comment.Content = strings.TrimSpace(params.Content)
	comment.UpdatedAt = time.Now()

	if err := s.dao.UpdateComment(ctx, comment); err != nil {
		return nil, fmt.Errorf("更新评论失败: %v", err)
	}

	// 清除缓存
	s.clearCommentCache(ctx, comment.ObjectID, comment.ObjectType)

	// 发送事件
	s.publishEvent(ctx, model.EventCommentUpdated, comment)

	s.logger.Info(ctx, "Comment updated successfully",
		logger.F("commentID", comment.ID),
		logger.F("userID", params.UserID))

	return comment, nil
}

// DeleteComment 删除评论
func (s *Service) DeleteComment(ctx context.Context, params *model.DeleteCommentParams) error {
	// 参数验证
	if params.CommentID <= 0 {
		return fmt.Errorf("评论ID无效")
	}
	if params.UserID <= 0 && !params.IsAdmin {
		return fmt.Errorf("用户ID无效")
	}

	// 获取评论
	comment, err := s.dao.GetComment(ctx, params.CommentID)
	if err != nil {
		return fmt.Errorf("评论不存在: %v", err)
	}

	// 权限检查
	if !comment.CanDelete(params.UserID, params.IsAdmin) {
		return fmt.Errorf("无权限删除此评论")
	}

	// 删除评论
	if err := s.dao.DeleteComment(ctx, params.CommentID); err != nil {
		return fmt.Errorf("删除评论失败: %v", err)
	}

	// 清除缓存
	s.clearCommentCache(ctx, comment.ObjectID, comment.ObjectType)

	// 发送事件
	s.publishEvent(ctx, model.EventCommentDeleted, comment)

	s.logger.Info(ctx, "Comment deleted successfully",
		logger.F("commentID", comment.ID),
		logger.F("userID", params.UserID),
		logger.F("isAdmin", params.IsAdmin))

	return nil
}

// GetComment 获取评论
func (s *Service) GetComment(ctx context.Context, commentID int64) (*model.Comment, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "comment.service.GetComment")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("comment.id", commentID))

	if commentID <= 0 {
		span.SetStatus(codes.Error, "invalid comment ID")
		return nil, fmt.Errorf("评论ID无效")
	}

	comment, err := s.dao.GetComment(ctx, commentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "comment not found")
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("comment.user_id", comment.UserID),
		attribute.String("comment.status", comment.Status),
	)

	span.SetStatus(codes.Ok, "comment retrieved successfully")
	return comment, nil
}

// GetComments 获取评论列表
func (s *Service) GetComments(ctx context.Context, params *model.GetCommentsParams) ([]*model.Comment, int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "comment.service.GetComments")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("comment.object_id", params.ObjectID),
		attribute.String("comment.object_type", params.ObjectType),
		attribute.Int("comment.page", int(params.Page)),
		attribute.Int("comment.page_size", int(params.PageSize)),
	)

	// 参数验证和默认值设置
	if params.ObjectID <= 0 {
		span.SetStatus(codes.Error, "invalid object ID")
		return nil, 0, fmt.Errorf("对象ID无效")
	}
	if params.ObjectType == "" {
		span.SetStatus(codes.Error, "invalid object type")
		return nil, 0, fmt.Errorf("对象类型无效")
	}

	// 设置默认值
	if params.Page <= 0 {
		params.Page = model.DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}
	if params.SortBy == "" {
		params.SortBy = model.SortByTime
	}
	if params.SortOrder == "" {
		params.SortOrder = model.SortOrderDesc
	}
	if params.MaxReplyCount <= 0 {
		params.MaxReplyCount = model.DefaultReplyShow
	}

	comments, total, err := s.dao.GetComments(ctx, params)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get comments")
		return nil, 0, err
	}

	span.SetAttributes(
		attribute.Int("comment.result_count", len(comments)),
		attribute.Int64("comment.total_count", total),
	)

	span.SetStatus(codes.Ok, "comments retrieved successfully")
	return comments, total, nil
}

// GetUserComments 获取用户评论
func (s *Service) GetUserComments(ctx context.Context, params *model.GetUserCommentsParams) ([]*model.Comment, int64, error) {
	if params.UserID <= 0 {
		return nil, 0, fmt.Errorf("用户ID无效")
	}

	// 设置默认值
	if params.Page <= 0 {
		params.Page = model.DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}

	return s.dao.GetUserComments(ctx, params)
}

// ModerateComment 审核评论
func (s *Service) ModerateComment(ctx context.Context, params *model.ModerateCommentParams) (*model.Comment, error) {
	// 参数验证
	if params.CommentID <= 0 {
		return nil, fmt.Errorf("评论ID无效")
	}
	if params.ModeratorID <= 0 {
		return nil, fmt.Errorf("审核员ID无效")
	}
	if params.NewStatus == "" {
		return nil, fmt.Errorf("新状态无效")
	}

	// 获取评论
	comment, err := s.dao.GetComment(ctx, params.CommentID)
	if err != nil {
		return nil, fmt.Errorf("评论不存在: %v", err)
	}

	oldStatus := comment.Status

	// 更新状态
	if err := s.dao.UpdateCommentStatus(ctx, params.CommentID, params.NewStatus); err != nil {
		return nil, fmt.Errorf("更新评论状态失败: %v", err)
	}

	// 记录审核日志
	log := &model.CommentModerationLog{
		CommentID:   params.CommentID,
		ModeratorID: params.ModeratorID,
		OldStatus:   oldStatus,
		NewStatus:   params.NewStatus,
		Reason:      params.Reason,
		CreatedAt:   time.Now(),
	}
	if err := s.dao.CreateModerationLog(ctx, log); err != nil {
		s.logger.Error(ctx, "Failed to create moderation log", logger.F("error", err.Error()))
	}

	// 更新评论对象
	comment.Status = params.NewStatus

	// 清除缓存
	s.clearCommentCache(ctx, comment.ObjectID, comment.ObjectType)

	// 发送事件
	eventType := model.EventCommentApproved
	if params.NewStatus == model.CommentStatusRejected {
		eventType = model.EventCommentRejected
	}
	s.publishEvent(ctx, eventType, comment)

	s.logger.Info(ctx, "Comment moderated successfully",
		logger.F("commentID", comment.ID),
		logger.F("moderatorID", params.ModeratorID),
		logger.F("oldStatus", oldStatus),
		logger.F("newStatus", params.NewStatus))

	return comment, nil
}

// PinComment 置顶评论
func (s *Service) PinComment(ctx context.Context, params *model.PinCommentParams) error {
	// 参数验证
	if params.CommentID <= 0 {
		return fmt.Errorf("评论ID无效")
	}
	if params.OperatorID <= 0 {
		return fmt.Errorf("操作员ID无效")
	}

	// 获取评论
	comment, err := s.dao.GetComment(ctx, params.CommentID)
	if err != nil {
		return fmt.Errorf("评论不存在: %v", err)
	}

	// 更新置顶状态
	if err := s.dao.UpdatePinStatus(ctx, params.CommentID, params.IsPinned); err != nil {
		return fmt.Errorf("更新置顶状态失败: %v", err)
	}

	// 清除缓存
	s.clearCommentCache(ctx, comment.ObjectID, comment.ObjectType)

	// 发送事件
	eventType := model.EventCommentPinned
	if !params.IsPinned {
		eventType = model.EventCommentUnpinned
	}
	s.publishEvent(ctx, eventType, comment)

	s.logger.Info(ctx, "Comment pin status updated",
		logger.F("commentID", comment.ID),
		logger.F("operatorID", params.OperatorID),
		logger.F("isPinned", params.IsPinned))

	return nil
}

// 辅助方法

// validateCreateCommentParams 验证创建评论参数
func (s *Service) validateCreateCommentParams(params *model.CreateCommentParams) error {
	if params.ObjectID <= 0 {
		return fmt.Errorf("对象ID无效")
	}
	if params.ObjectType == "" {
		return fmt.Errorf("对象类型无效")
	}
	if params.UserID <= 0 {
		return fmt.Errorf("用户ID无效")
	}
	if params.UserName == "" {
		return fmt.Errorf("用户名无效")
	}

	content := strings.TrimSpace(params.Content)
	if len(content) < model.MinCommentLength {
		return fmt.Errorf("评论内容不能为空")
	}
	if len(content) > model.MaxCommentLength {
		return fmt.Errorf("评论内容过长，最多%d个字符", model.MaxCommentLength)
	}

	return nil
}

// clearCommentCache 清除评论相关缓存
func (s *Service) clearCommentCache(ctx context.Context, objectID int64, objectType string) {
	// 清除评论列表缓存
	pattern := fmt.Sprintf("%s%d:%s:*", model.CommentListCachePrefix, objectID, objectType)
	keys, err := s.redis.Keys(ctx, pattern)
	if err == nil && len(keys) > 0 {
		s.redis.Del(ctx, keys...)
	}

	// 清除统计缓存
	statsKey := fmt.Sprintf("%s%d:%s", model.CommentStatsCachePrefix, objectID, objectType)
	s.redis.Del(ctx, statsKey)
}

// publishEvent 发布事件
func (s *Service) publishEvent(ctx context.Context, eventType string, comment *model.Comment) {
	if s.producer == nil {
		return
	}

	// 构建protobuf事件消息并发送到Kafka
	go func() {
		event := &rest.CommentEvent{
			Type:       eventType,
			CommentId:  comment.ID,
			ObjectId:   comment.ObjectID,
			ObjectType: convertObjectTypeToProto(comment.ObjectType),
			UserId:     comment.UserID,
			Timestamp:  time.Now().Unix(),
		}

		if err := s.producer.PublishMessage("comment-events", event); err != nil {
			s.logger.Error(context.Background(), "Failed to publish event",
				logger.F("eventType", eventType),
				logger.F("commentID", comment.ID),
				logger.F("error", err.Error()))
		}
	}()
}

// GetCommentStats 获取评论统计
func (s *Service) GetCommentStats(ctx context.Context, objectID int64, objectType string) (*model.CommentStats, error) {
	if objectID <= 0 {
		return nil, fmt.Errorf("对象ID无效")
	}
	if objectType == "" {
		return nil, fmt.Errorf("对象类型无效")
	}

	return s.dao.GetCommentStats(ctx, objectID, objectType)
}

// GetBatchCommentStats 批量获取评论统计
func (s *Service) GetBatchCommentStats(ctx context.Context, objectIDs []int64, objectType string) ([]*model.CommentStats, error) {
	if len(objectIDs) == 0 {
		return nil, fmt.Errorf("对象ID列表不能为空")
	}
	if objectType == "" {
		return nil, fmt.Errorf("对象类型无效")
	}

	return s.dao.GetBatchCommentStats(ctx, objectIDs, objectType)
}

// LikeComment 点赞评论
func (s *Service) LikeComment(ctx context.Context, commentID, userID int64) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "comment.service.LikeComment")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("comment.id", commentID),
		attribute.Int64("user.id", userID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	if commentID <= 0 {
		span.SetStatus(codes.Error, "invalid comment ID")
		return fmt.Errorf("评论ID无效")
	}
	if userID <= 0 {
		span.SetStatus(codes.Error, "invalid user ID")
		return fmt.Errorf("用户ID无效")
	}

	// 检查评论是否存在
	comment, err := s.dao.GetComment(ctx, commentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "comment not found")
		return fmt.Errorf("评论不存在: %v", err)
	}

	if comment.Status != model.CommentStatusApproved {
		span.SetStatus(codes.Error, "comment not approved")
		return fmt.Errorf("只能对已通过的评论点赞")
	}

	if err := s.dao.AddCommentLike(ctx, commentID, userID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add like")
		return err
	}

	s.logger.Info(ctx, "Comment liked successfully",
		logger.F("commentID", commentID),
		logger.F("userID", userID))

	span.SetStatus(codes.Ok, "comment liked successfully")
	return nil
}

// UnlikeComment 取消点赞评论
func (s *Service) UnlikeComment(ctx context.Context, commentID, userID int64) error {
	if commentID <= 0 {
		return fmt.Errorf("评论ID无效")
	}
	if userID <= 0 {
		return fmt.Errorf("用户ID无效")
	}

	return s.dao.RemoveCommentLike(ctx, commentID, userID)
}

// IsCommentLiked 检查是否已点赞
func (s *Service) IsCommentLiked(ctx context.Context, commentID, userID int64) (bool, error) {
	if commentID <= 0 || userID <= 0 {
		return false, fmt.Errorf("参数无效")
	}

	return s.dao.IsCommentLiked(ctx, commentID, userID)
}

// GetPendingComments 获取待审核评论
func (s *Service) GetPendingComments(ctx context.Context, page, pageSize int32) ([]*model.Comment, int64, error) {
	if page <= 0 {
		page = model.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	return s.dao.GetPendingComments(ctx, page, pageSize)
}

// GetCommentsByStatus 根据状态获取评论
func (s *Service) GetCommentsByStatus(ctx context.Context, status string, page, pageSize int32) ([]*model.Comment, int64, error) {
	if status == "" {
		return nil, 0, fmt.Errorf("状态无效")
	}
	if page <= 0 {
		page = model.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	return s.dao.GetCommentsByStatus(ctx, status, page, pageSize)
}

// convertObjectTypeToProto 将对象类型转换为protobuf枚举
func convertObjectTypeToProto(objectType string) rest.CommentObjectType {
	switch objectType {
	case model.ObjectTypePost:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_POST
	case model.ObjectTypeArticle:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_ARTICLE
	case model.ObjectTypeVideo:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_VIDEO
	default:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_UNSPECIFIED
	}
}
