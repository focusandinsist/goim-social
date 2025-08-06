package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/apps/content-service/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
	"goim-social/pkg/telemetry"
)

// ==================== 评论相关业务逻辑 ====================

// CreateCommentParams 创建评论参数
type CreateCommentParams struct {
	TargetID        int64
	TargetType      string
	UserID          int64
	UserName        string
	UserAvatar      string
	Content         string
	ParentID        int64
	ReplyToUserID   int64
	ReplyToUserName string
	IPAddress       string
	UserAgent       string
}

// CreateComment 创建评论
func (s *Service) CreateComment(ctx context.Context, params CreateCommentParams) (*model.Comment, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.CreateComment")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("comment.target_id", params.TargetID),
		attribute.String("comment.target_type", params.TargetType),
		attribute.Int64("comment.user_id", params.UserID),
		attribute.Int64("comment.parent_id", params.ParentID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, params.UserID)

	// 参数验证
	if err := s.validateCreateCommentParams(params); err != nil {
		span.SetStatus(codes.Error, "invalid parameters")
		return nil, err
	}

	// 检查目标对象是否存在
	if err := s.validateCommentTarget(ctx, params.TargetID, params.TargetType); err != nil {
		span.SetStatus(codes.Error, "target validation failed")
		return nil, err
	}

	// 构建评论对象
	comment := &model.Comment{
		TargetID:        params.TargetID,
		TargetType:      params.TargetType,
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

	// 设置根评论ID
	if params.ParentID == 0 {
		comment.RootID = 0 // 顶级评论
	} else {
		// 获取父评论信息
		parentComment, err := s.dao.GetComment(ctx, params.ParentID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get parent comment")
			return nil, fmt.Errorf("获取父评论失败: %v", err)
		}

		if parentComment.RootID == 0 {
			comment.RootID = parentComment.ID // 父评论是顶级评论
		} else {
			comment.RootID = parentComment.RootID // 继承根评论ID
		}
	}

	// 创建评论
	if err := s.dao.CreateComment(ctx, comment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create comment")
		return nil, fmt.Errorf("创建评论失败: %v", err)
	}

	// 更新相关计数
	go s.updateCommentCounts(context.Background(), comment)

	// 发送事件到消息队列
	go s.publishCommentEvent(context.Background(), "create", comment)

	s.logger.Info(ctx, "Comment created successfully",
		logger.F("commentID", comment.ID),
		logger.F("targetID", params.TargetID),
		logger.F("targetType", params.TargetType),
		logger.F("userID", params.UserID))

	span.SetStatus(codes.Ok, "comment created successfully")
	return comment, nil
}

// DeleteComment 删除评论
func (s *Service) DeleteComment(ctx context.Context, commentID, userID int64) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.DeleteComment")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("comment.id", commentID),
		attribute.Int64("comment.user_id", userID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 获取评论信息
	comment, err := s.dao.GetComment(ctx, commentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "comment not found")
		return fmt.Errorf("评论不存在")
	}

	// 权限检查（只能删除自己的评论）
	if comment.UserID != userID {
		span.SetStatus(codes.Error, "permission denied")
		return fmt.Errorf("无权限删除此评论")
	}

	// 删除评论
	if err := s.dao.DeleteComment(ctx, commentID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete comment")
		return fmt.Errorf("删除评论失败: %v", err)
	}

	// 更新相关计数
	go s.updateCommentCountsOnDelete(context.Background(), comment)

	// 发送事件到消息队列
	go s.publishCommentEvent(context.Background(), "delete", comment)

	s.logger.Info(ctx, "Comment deleted successfully",
		logger.F("commentID", commentID),
		logger.F("userID", userID))

	span.SetStatus(codes.Ok, "comment deleted successfully")
	return nil
}

// GetComments 获取评论列表
func (s *Service) GetComments(ctx context.Context, targetID int64, targetType string, parentID int64, sortBy, sortOrder string, page, pageSize int32) ([]*model.Comment, int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.GetComments")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("comment.target_id", targetID),
		attribute.String("comment.target_type", targetType),
		attribute.Int64("comment.parent_id", parentID),
		attribute.String("comment.sort_by", sortBy),
		attribute.Int("comment.page", int(page)),
		attribute.Int("comment.page_size", int(pageSize)),
	)

	// 参数验证
	if targetID <= 0 {
		span.SetStatus(codes.Error, "invalid target ID")
		return nil, 0, fmt.Errorf("目标ID无效")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > model.MaxBatchSize {
		pageSize = model.DefaultPageSize
	}

	// 获取评论列表
	comments, total, err := s.dao.GetComments(ctx, targetID, targetType, parentID, sortBy, sortOrder, page, pageSize)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get comments")
		return nil, 0, fmt.Errorf("获取评论列表失败: %v", err)
	}

	span.SetAttributes(
		attribute.Int64("comment.total", total),
		attribute.Int("comment.count", len(comments)),
	)

	s.logger.Info(ctx, "Comments retrieved successfully",
		logger.F("targetID", targetID),
		logger.F("targetType", targetType),
		logger.F("total", total),
		logger.F("count", len(comments)))

	span.SetStatus(codes.Ok, "comments retrieved successfully")
	return comments, total, nil
}

// GetCommentReplies 获取评论回复
func (s *Service) GetCommentReplies(ctx context.Context, commentID int64, sortBy, sortOrder string, page, pageSize int32) ([]*model.Comment, int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.GetCommentReplies")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("comment.id", commentID),
		attribute.String("comment.sort_by", sortBy),
		attribute.Int("comment.page", int(page)),
		attribute.Int("comment.page_size", int(pageSize)),
	)

	// 参数验证
	if commentID <= 0 {
		span.SetStatus(codes.Error, "invalid comment ID")
		return nil, 0, fmt.Errorf("评论ID无效")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > model.MaxBatchSize {
		pageSize = model.DefaultPageSize
	}

	// 获取回复列表
	replies, total, err := s.dao.GetCommentReplies(ctx, commentID, sortBy, sortOrder, page, pageSize)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get comment replies")
		return nil, 0, fmt.Errorf("获取评论回复失败: %v", err)
	}

	span.SetAttributes(
		attribute.Int64("comment.total", total),
		attribute.Int("comment.count", len(replies)),
	)

	s.logger.Info(ctx, "Comment replies retrieved successfully",
		logger.F("commentID", commentID),
		logger.F("total", total),
		logger.F("count", len(replies)))

	span.SetStatus(codes.Ok, "comment replies retrieved successfully")
	return replies, total, nil
}

// validateCreateCommentParams 验证创建评论参数
func (s *Service) validateCreateCommentParams(params CreateCommentParams) error {
	if params.TargetID <= 0 {
		return fmt.Errorf("目标ID无效")
	}
	if params.TargetType == "" {
		return fmt.Errorf("目标类型不能为空")
	}
	if params.UserID <= 0 {
		return fmt.Errorf("用户ID无效")
	}
	if params.UserName == "" {
		return fmt.Errorf("用户名不能为空")
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

// validateCommentTarget 验证评论目标
func (s *Service) validateCommentTarget(ctx context.Context, targetID int64, targetType string) error {
	switch targetType {
	case model.TargetTypeContent:
		// 检查内容是否存在且已发布
		content, err := s.dao.GetContent(ctx, targetID)
		if err != nil {
			return fmt.Errorf("内容不存在")
		}
		if content.Status != model.ContentStatusPublished {
			return fmt.Errorf("内容未发布，无法评论")
		}
	case model.TargetTypeComment:
		// 检查评论是否存在
		_, err := s.dao.GetComment(ctx, targetID)
		if err != nil {
			return fmt.Errorf("评论不存在")
		}
	default:
		return fmt.Errorf("不支持的目标类型: %s", targetType)
	}
	return nil
}
