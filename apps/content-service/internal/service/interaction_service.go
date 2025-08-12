package service

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"

	"goim-social/apps/content-service/internal/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
	"goim-social/pkg/telemetry"
)

// ==================== 互动相关业务逻辑 ====================

// DoInteraction 执行互动操作
func (s *Service) DoInteraction(ctx context.Context, userID, targetID int64, targetType, interactionType, metadata string) (*model.Interaction, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.DoInteraction")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("interaction.user_id", userID),
		attribute.Int64("interaction.target_id", targetID),
		attribute.String("interaction.target_type", targetType),
		attribute.String("interaction.type", interactionType),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 参数验证
	if err := s.validateInteractionParams(userID, targetID, targetType, interactionType); err != nil {
		span.SetStatus(codes.Error, "invalid parameters")
		return nil, err
	}

	// 检查是否已经存在相同的互动
	existingInteraction, err := s.dao.GetInteraction(ctx, userID, targetID, targetType, interactionType)
	if err != nil && err != gorm.ErrRecordNotFound {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check existing interaction")
		return nil, fmt.Errorf("检查互动状态失败: %v", err)
	}

	if existingInteraction != nil {
		span.SetStatus(codes.Error, "interaction already exists")
		return nil, fmt.Errorf("已经执行过此互动")
	}

	// 检查目标对象是否存在
	if err := s.validateInteractionTarget(ctx, targetID, targetType); err != nil {
		span.SetStatus(codes.Error, "target validation failed")
		return nil, err
	}

	// 创建互动记录
	interaction := &model.Interaction{
		UserID:          userID,
		TargetID:        targetID,
		TargetType:      targetType,
		InteractionType: interactionType,
		Metadata:        metadata,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.dao.CreateInteraction(ctx, interaction); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create interaction")
		return nil, fmt.Errorf("创建互动失败: %v", err)
	}

	// 设置交互ID到span
	span.SetAttributes(attribute.Int64("interaction.id", interaction.ID))

	// 更新统计数据
	go s.updateInteractionStats(context.Background(), targetID, targetType, interactionType, 1)

	// 清除相关缓存
	go s.clearInteractionCache(context.Background(), userID, targetID, targetType, interactionType)

	// 发送事件到消息队列
	go s.publishInteractionEvent(context.Background(), "create", interaction)

	s.logger.Info(ctx, "Interaction created successfully",
		logger.F("interactionID", interaction.ID),
		logger.F("userID", userID),
		logger.F("targetID", targetID),
		logger.F("interactionType", interactionType))

	span.SetStatus(codes.Ok, "interaction created successfully")
	return interaction, nil
}

// UndoInteraction 取消互动操作
func (s *Service) UndoInteraction(ctx context.Context, userID, targetID int64, targetType, interactionType string) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.UndoInteraction")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("interaction.user_id", userID),
		attribute.Int64("interaction.target_id", targetID),
		attribute.String("interaction.target_type", targetType),
		attribute.String("interaction.type", interactionType),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 参数验证
	if err := s.validateInteractionParams(userID, targetID, targetType, interactionType); err != nil {
		span.SetStatus(codes.Error, "invalid parameters")
		return err
	}

	// 检查互动是否存在
	existingInteraction, err := s.dao.GetInteraction(ctx, userID, targetID, targetType, interactionType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			span.SetStatus(codes.Error, "interaction not found")
			return fmt.Errorf("互动不存在")
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get interaction")
		return fmt.Errorf("获取互动失败: %v", err)
	}

	// 删除互动记录
	if err := s.dao.DeleteInteraction(ctx, userID, targetID, targetType, interactionType); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete interaction")
		return fmt.Errorf("删除互动失败: %v", err)
	}

	// 更新统计数据
	go s.updateInteractionStats(context.Background(), targetID, targetType, interactionType, -1)

	// 清除相关缓存
	go s.clearInteractionCache(context.Background(), userID, targetID, targetType, interactionType)

	// 发送事件到消息队列
	go s.publishInteractionEvent(context.Background(), "delete", existingInteraction)

	s.logger.Info(ctx, "Interaction deleted successfully",
		logger.F("userID", userID),
		logger.F("targetID", targetID),
		logger.F("interactionType", interactionType))

	span.SetStatus(codes.Ok, "interaction deleted successfully")
	return nil
}

// CheckInteraction 检查互动状态
func (s *Service) CheckInteraction(ctx context.Context, userID, targetID int64, targetType, interactionType string) (bool, *model.Interaction, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.CheckInteraction")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("interaction.user_id", userID),
		attribute.Int64("interaction.target_id", targetID),
		attribute.String("interaction.target_type", targetType),
		attribute.String("interaction.type", interactionType),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 参数验证
	if err := s.validateInteractionParams(userID, targetID, targetType, interactionType); err != nil {
		span.SetStatus(codes.Error, "invalid parameters")
		return false, nil, err
	}

	// 检查互动是否存在
	interaction, err := s.dao.GetInteraction(ctx, userID, targetID, targetType, interactionType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			span.SetStatus(codes.Ok, "interaction not found")
			return false, nil, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get interaction")
		return false, nil, fmt.Errorf("检查互动失败: %v", err)
	}

	span.SetAttributes(attribute.Bool("interaction.exists", true))
	span.SetStatus(codes.Ok, "interaction checked successfully")
	return true, interaction, nil
}

// GetInteractionStats 获取互动统计
func (s *Service) GetInteractionStats(ctx context.Context, targetID int64, targetType string) (*model.InteractionStats, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "content.service.GetInteractionStats")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("interaction.target_id", targetID),
		attribute.String("interaction.target_type", targetType),
	)

	// 参数验证
	if targetID <= 0 {
		span.SetStatus(codes.Error, "invalid target ID")
		return nil, fmt.Errorf("目标ID无效")
	}
	if targetType == "" {
		span.SetStatus(codes.Error, "invalid target type")
		return nil, fmt.Errorf("目标类型不能为空")
	}

	// 获取统计数据
	stats, err := s.dao.GetInteractionStats(ctx, targetID, targetType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get interaction stats")
		return nil, fmt.Errorf("获取互动统计失败: %v", err)
	}

	span.SetAttributes(
		attribute.Int64("interaction.like_count", stats.LikeCount),
		attribute.Int64("interaction.favorite_count", stats.FavoriteCount),
		attribute.Int64("interaction.share_count", stats.ShareCount),
		attribute.Int64("interaction.repost_count", stats.RepostCount),
	)

	s.logger.Info(ctx, "Interaction stats retrieved successfully",
		logger.F("targetID", targetID),
		logger.F("targetType", targetType),
		logger.F("likeCount", stats.LikeCount),
		logger.F("favoriteCount", stats.FavoriteCount))

	span.SetStatus(codes.Ok, "interaction stats retrieved successfully")
	return stats, nil
}

// validateInteractionParams 验证互动参数
func (s *Service) validateInteractionParams(userID, targetID int64, targetType, interactionType string) error {
	if userID <= 0 {
		return fmt.Errorf("用户ID无效")
	}
	if targetID <= 0 {
		return fmt.Errorf("目标ID无效")
	}
	if targetType == "" {
		return fmt.Errorf("目标类型不能为空")
	}
	if interactionType == "" {
		return fmt.Errorf("互动类型不能为空")
	}

	// 验证目标类型
	validTargetTypes := []string{model.TargetTypeContent, model.TargetTypeComment, model.TargetTypeUser}
	isValidTargetType := false
	for _, validType := range validTargetTypes {
		if targetType == validType {
			isValidTargetType = true
			break
		}
	}
	if !isValidTargetType {
		return fmt.Errorf("不支持的目标类型: %s", targetType)
	}

	// 验证互动类型
	validInteractionTypes := []string{
		model.InteractionTypeLike, model.InteractionTypeFavorite,
		model.InteractionTypeShare, model.InteractionTypeRepost,
	}
	isValidInteractionType := false
	for _, validType := range validInteractionTypes {
		if interactionType == validType {
			isValidInteractionType = true
			break
		}
	}
	if !isValidInteractionType {
		return fmt.Errorf("不支持的互动类型: %s", interactionType)
	}

	return nil
}

// validateInteractionTarget 验证互动目标
func (s *Service) validateInteractionTarget(ctx context.Context, targetID int64, targetType string) error {
	switch targetType {
	case model.TargetTypeContent:
		// 检查内容是否存在且已发布
		content, err := s.dao.GetContent(ctx, targetID)
		if err != nil {
			return fmt.Errorf("内容不存在")
		}
		if content.Status != model.ContentStatusPublished {
			return fmt.Errorf("内容未发布，无法互动")
		}
	case model.TargetTypeComment:
		// 检查评论是否存在
		_, err := s.dao.GetComment(ctx, targetID)
		if err != nil {
			return fmt.Errorf("评论不存在")
		}
	case model.TargetTypeUser:
		// 用户相关的互动（如关注）暂时跳过验证
		// TODO: 可以添加用户存在性验证
	default:
		return fmt.Errorf("不支持的目标类型: %s", targetType)
	}
	return nil
}
