package dao

import (
	"context"
	"fmt"

	"goim-social/apps/content-service/internal/model"
	"gorm.io/gorm"
)

// ==================== 互动相关方法实现 ====================

// CreateInteraction 创建互动
func (d *contentDAO) CreateInteraction(ctx context.Context, interaction *model.Interaction) error {
	return d.db.GetDB().WithContext(ctx).Create(interaction).Error
}

// DeleteInteraction 删除互动
func (d *contentDAO) DeleteInteraction(ctx context.Context, userID, targetID int64, targetType, interactionType string) error {
	return d.db.GetDB().WithContext(ctx).
		Where("user_id = ? AND target_id = ? AND target_type = ? AND interaction_type = ?",
			userID, targetID, targetType, interactionType).
		Delete(&model.Interaction{}).Error
}

// GetInteraction 获取互动
func (d *contentDAO) GetInteraction(ctx context.Context, userID, targetID int64, targetType, interactionType string) (*model.Interaction, error) {
	var interaction model.Interaction
	err := d.db.GetDB().WithContext(ctx).
		Where("user_id = ? AND target_id = ? AND target_type = ? AND interaction_type = ?",
			userID, targetID, targetType, interactionType).
		First(&interaction).Error

	if err != nil {
		return nil, err
	}
	return &interaction, nil
}

// BatchCheckInteractions 批量检查互动
func (d *contentDAO) BatchCheckInteractions(ctx context.Context, userID int64, targetIDs []int64, targetType, interactionType string) (map[int64]bool, error) {
	var interactions []model.Interaction
	result := make(map[int64]bool)

	// 初始化结果map
	for _, targetID := range targetIDs {
		result[targetID] = false
	}

	err := d.db.GetDB().WithContext(ctx).
		Where("user_id = ? AND target_id IN ? AND target_type = ? AND interaction_type = ?",
			userID, targetIDs, targetType, interactionType).
		Find(&interactions).Error

	if err != nil {
		return nil, err
	}

	// 更新存在的互动
	for _, interaction := range interactions {
		result[interaction.TargetID] = true
	}

	return result, nil
}

// GetUserInteractions 获取用户互动列表
func (d *contentDAO) GetUserInteractions(ctx context.Context, userID int64, targetType, interactionType string, page, pageSize int32) ([]*model.Interaction, int64, error) {
	var interactions []*model.Interaction
	var total int64

	query := d.db.GetDB().WithContext(ctx).Model(&model.Interaction{}).
		Where("user_id = ?", userID)

	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if interactionType != "" {
		query = query.Where("interaction_type = ?", interactionType)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Find(&interactions).Error

	return interactions, total, err
}

// GetInteractionStats 获取互动统计
func (d *contentDAO) GetInteractionStats(ctx context.Context, targetID int64, targetType string) (*model.InteractionStats, error) {
	var stats model.InteractionStats
	err := d.db.GetDB().WithContext(ctx).
		Where("target_id = ? AND target_type = ?", targetID, targetType).
		First(&stats).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果没有统计记录，创建一个空的
			stats = model.InteractionStats{
				TargetID:   targetID,
				TargetType: targetType,
			}
			return &stats, nil
		}
		return nil, err
	}
	return &stats, nil
}

// BatchGetInteractionStats 批量获取互动统计
func (d *contentDAO) BatchGetInteractionStats(ctx context.Context, targetIDs []int64, targetType string) ([]*model.InteractionStats, error) {
	var stats []*model.InteractionStats

	err := d.db.GetDB().WithContext(ctx).
		Where("target_id IN ? AND target_type = ?", targetIDs, targetType).
		Find(&stats).Error

	if err != nil {
		return nil, err
	}

	// 为没有统计记录的目标创建空记录
	existingMap := make(map[int64]bool)
	for _, stat := range stats {
		existingMap[stat.TargetID] = true
	}

	for _, targetID := range targetIDs {
		if !existingMap[targetID] {
			stats = append(stats, &model.InteractionStats{
				TargetID:   targetID,
				TargetType: targetType,
			})
		}
	}

	return stats, nil
}

// UpdateInteractionStats 更新互动统计
func (d *contentDAO) UpdateInteractionStats(ctx context.Context, targetID int64, targetType, interactionType string, delta int64) error {
	// 根据互动类型更新对应字段
	var updateColumn string
	switch interactionType {
	case model.InteractionTypeLike:
		updateColumn = "like_count"
	case model.InteractionTypeFavorite:
		updateColumn = "favorite_count"
	case model.InteractionTypeShare:
		updateColumn = "share_count"
	case model.InteractionTypeRepost:
		updateColumn = "repost_count"
	default:
		return fmt.Errorf("unsupported interaction type: %s", interactionType)
	}

	// 使用 ON CONFLICT 进行 UPSERT
	return d.db.GetDB().WithContext(ctx).
		Exec(fmt.Sprintf(`
			INSERT INTO interaction_stats (target_id, target_type, %s, updated_at) 
			VALUES (?, ?, ?, NOW()) 
			ON CONFLICT (target_id, target_type) 
			DO UPDATE SET %s = interaction_stats.%s + ?, updated_at = NOW()
		`, updateColumn, updateColumn, updateColumn),
			targetID, targetType, delta, delta).Error
}

// IncrementInteractionCount 增加互动计数
func (d *contentDAO) IncrementInteractionCount(ctx context.Context, targetID int64, targetType, interactionType string) error {
	return d.updateTargetInteractionCount(ctx, targetID, targetType, interactionType, 1)
}

// DecrementInteractionCount 减少互动计数
func (d *contentDAO) DecrementInteractionCount(ctx context.Context, targetID int64, targetType, interactionType string) error {
	return d.updateTargetInteractionCount(ctx, targetID, targetType, interactionType, -1)
}

// updateTargetInteractionCount 更新目标对象的互动计数
func (d *contentDAO) updateTargetInteractionCount(ctx context.Context, targetID int64, targetType, interactionType string, delta int64) error {
	switch targetType {
	case model.TargetTypeContent:
		return d.updateContentInteractionCount(ctx, targetID, interactionType, delta)
	case model.TargetTypeComment:
		return d.updateCommentInteractionCount(ctx, targetID, interactionType, delta)
	default:
		return fmt.Errorf("unsupported target type: %s", targetType)
	}
}

// updateContentInteractionCount 更新内容的互动计数
func (d *contentDAO) updateContentInteractionCount(ctx context.Context, contentID int64, interactionType string, delta int64) error {
	var updateColumn string
	switch interactionType {
	case model.InteractionTypeLike:
		updateColumn = "like_count"
	case model.InteractionTypeFavorite:
		updateColumn = "favorite_count"
	case model.InteractionTypeShare:
		updateColumn = "share_count"
	default:
		return nil // 其他类型不更新内容表
	}

	return d.db.GetDB().WithContext(ctx).Model(&model.Content{}).
		Where("id = ?", contentID).
		UpdateColumn(updateColumn, gorm.Expr(fmt.Sprintf("%s + ?", updateColumn), delta)).Error
}

// updateCommentInteractionCount 更新评论的互动计数
func (d *contentDAO) updateCommentInteractionCount(ctx context.Context, commentID int64, interactionType string, delta int64) error {
	if interactionType == model.InteractionTypeLike {
		return d.db.GetDB().WithContext(ctx).Model(&model.Comment{}).
			Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
	}
	return nil // 其他类型不更新评论表
}
