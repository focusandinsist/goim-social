package dao

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"goim-social/apps/content-service/internal/model"
)

// ==================== 聚合查询方法实现 ====================

// GetContentWithDetails 获取内容详情（包含评论和互动）
func (d *contentDAO) GetContentWithDetails(ctx context.Context, contentID, userID int64, commentLimit int32) (*model.Content, []*model.Comment, *model.InteractionStats, map[string]bool, error) {
	var content model.Content
	var comments []*model.Comment
	var stats model.InteractionStats
	userInteractions := make(map[string]bool)

	// 获取内容
	if err := d.db.GetDB().WithContext(ctx).First(&content, contentID).Error; err != nil {
		return nil, nil, nil, nil, err
	}

	// 获取热门评论
	if commentLimit > 0 {
		err := d.db.GetDB().WithContext(ctx).
			Where("target_id = ? AND target_type = ? AND parent_id = 0", contentID, model.TargetTypeContent).
			Order("like_count DESC, created_at DESC").
			Limit(int(commentLimit)).
			Find(&comments).Error
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	// 获取互动统计
	err := d.db.GetDB().WithContext(ctx).
		Where("target_id = ? AND target_type = ?", contentID, model.TargetTypeContent).
		First(&stats).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, nil, nil, nil, err
	}

	// 获取用户互动状态
	if userID > 0 {
		var interactions []model.Interaction
		err := d.db.GetDB().WithContext(ctx).
			Where("user_id = ? AND target_id = ? AND target_type = ?", userID, contentID, model.TargetTypeContent).
			Find(&interactions).Error
		if err != nil {
			return nil, nil, nil, nil, err
		}

		for _, interaction := range interactions {
			userInteractions[interaction.InteractionType] = true
		}
	}

	return &content, comments, &stats, userInteractions, nil
}

// GetContentFeed 获取内容流
func (d *contentDAO) GetContentFeed(ctx context.Context, userID int64, contentType, sortBy string, page, pageSize int32) ([]*model.Content, []*model.InteractionStats, map[int64]map[string]bool, error) {
	var contents []*model.Content
	
	query := d.db.GetDB().WithContext(ctx).Model(&model.Content{}).
		Where("status = ?", model.ContentStatusPublished)

	if contentType != "" {
		query = query.Where("type = ?", contentType)
	}

	// 排序
	switch sortBy {
	case "hot":
		query = query.Order("like_count DESC, view_count DESC, created_at DESC")
	case "trending":
		query = query.Order("(like_count + comment_count + share_count) DESC, created_at DESC")
	default: // time
		query = query.Order("created_at DESC")
	}

	// 分页
	offset := (page - 1) * pageSize
	err := query.Offset(int(offset)).Limit(int(pageSize)).Find(&contents).Error
	if err != nil {
		return nil, nil, nil, err
	}

	if len(contents) == 0 {
		return contents, nil, nil, nil
	}

	// 获取内容ID列表
	contentIDs := make([]int64, len(contents))
	for i, content := range contents {
		contentIDs[i] = content.ID
	}

	// 批量获取互动统计
	stats, err := d.BatchGetInteractionStats(ctx, contentIDs, model.TargetTypeContent)
	if err != nil {
		return nil, nil, nil, err
	}

	// 获取用户互动状态
	userInteractionsMap := make(map[int64]map[string]bool)
	if userID > 0 {
		var interactions []model.Interaction
		err := d.db.GetDB().WithContext(ctx).
			Where("user_id = ? AND target_id IN ? AND target_type = ?", userID, contentIDs, model.TargetTypeContent).
			Find(&interactions).Error
		if err != nil {
			return nil, nil, nil, err
		}

		// 初始化用户互动map
		for _, contentID := range contentIDs {
			userInteractionsMap[contentID] = make(map[string]bool)
		}

		// 填充用户互动状态
		for _, interaction := range interactions {
			userInteractionsMap[interaction.TargetID][interaction.InteractionType] = true
		}
	}

	return contents, stats, userInteractionsMap, nil
}

// GetTrendingContent 获取热门内容
func (d *contentDAO) GetTrendingContent(ctx context.Context, timeRange, contentType string, limit int32) ([]*model.Content, []*model.InteractionStats, error) {
	var contents []*model.Content
	
	query := d.db.GetDB().WithContext(ctx).Model(&model.Content{}).
		Where("status = ?", model.ContentStatusPublished)

	if contentType != "" {
		query = query.Where("type = ?", contentType)
	}

	// 根据时间范围过滤
	switch timeRange {
	case "hour":
		query = query.Where("created_at >= NOW() - INTERVAL '1 hour'")
	case "day":
		query = query.Where("created_at >= NOW() - INTERVAL '1 day'")
	case "week":
		query = query.Where("created_at >= NOW() - INTERVAL '1 week'")
	case "month":
		query = query.Where("created_at >= NOW() - INTERVAL '1 month'")
	}

	// 按热度排序
	err := query.Order("(like_count * 3 + comment_count * 2 + share_count * 5 + view_count * 0.1) DESC").
		Limit(int(limit)).
		Find(&contents).Error
	if err != nil {
		return nil, nil, err
	}

	if len(contents) == 0 {
		return contents, nil, nil
	}

	// 获取内容ID列表
	contentIDs := make([]int64, len(contents))
	for i, content := range contents {
		contentIDs[i] = content.ID
	}

	// 批量获取互动统计
	stats, err := d.BatchGetInteractionStats(ctx, contentIDs, model.TargetTypeContent)
	if err != nil {
		return nil, nil, err
	}

	return contents, stats, nil
}

// ==================== 事务操作方法实现 ====================

// DeleteContentWithRelated 删除内容及其相关数据
func (d *contentDAO) DeleteContentWithRelated(ctx context.Context, contentID int64) error {
	return d.db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 删除内容相关的评论
		if err := tx.Where("target_id = ? AND target_type = ?", contentID, model.TargetTypeContent).
			Delete(&model.Comment{}).Error; err != nil {
			return fmt.Errorf("failed to delete comments: %v", err)
		}

		// 2. 删除内容相关的互动
		if err := tx.Where("target_id = ? AND target_type = ?", contentID, model.TargetTypeContent).
			Delete(&model.Interaction{}).Error; err != nil {
			return fmt.Errorf("failed to delete interactions: %v", err)
		}

		// 3. 删除互动统计
		if err := tx.Where("target_id = ? AND target_type = ?", contentID, model.TargetTypeContent).
			Delete(&model.InteractionStats{}).Error; err != nil {
			return fmt.Errorf("failed to delete interaction stats: %v", err)
		}

		// 4. 删除内容标签关联
		if err := tx.Where("content_id = ?", contentID).
			Delete(&model.ContentTagRelation{}).Error; err != nil {
			return fmt.Errorf("failed to delete content tag relations: %v", err)
		}

		// 5. 删除内容话题关联
		if err := tx.Where("content_id = ?", contentID).
			Delete(&model.ContentTopicRelation{}).Error; err != nil {
			return fmt.Errorf("failed to delete content topic relations: %v", err)
		}

		// 6. 删除媒体文件
		if err := tx.Where("content_id = ?", contentID).
			Delete(&model.ContentMediaFile{}).Error; err != nil {
			return fmt.Errorf("failed to delete media files: %v", err)
		}

		// 7. 删除状态日志
		if err := tx.Where("content_id = ?", contentID).
			Delete(&model.ContentStatusLog{}).Error; err != nil {
			return fmt.Errorf("failed to delete status logs: %v", err)
		}

		// 8. 最后删除内容本身
		if err := tx.Delete(&model.Content{}, contentID).Error; err != nil {
			return fmt.Errorf("failed to delete content: %v", err)
		}

		return nil
	})
}

// BatchUpdateStats 批量更新统计数据
func (d *contentDAO) BatchUpdateStats(ctx context.Context, updates []model.StatsUpdate) error {
	return d.db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			// 更新互动统计表
			if err := d.UpdateInteractionStats(ctx, update.TargetID, update.TargetType, update.InteractionType, update.Delta); err != nil {
				return fmt.Errorf("failed to update interaction stats for target %d: %v", update.TargetID, err)
			}

			// 更新目标对象的计数
			if err := d.updateTargetInteractionCount(ctx, update.TargetID, update.TargetType, update.InteractionType, update.Delta); err != nil {
				return fmt.Errorf("failed to update target interaction count for target %d: %v", update.TargetID, err)
			}
		}
		return nil
	})
}
