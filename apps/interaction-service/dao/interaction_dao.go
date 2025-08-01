package dao

import (
	"context"
	"fmt"
	"time"

	"goim-social/apps/interaction-service/model"
	"goim-social/pkg/database"

	"gorm.io/gorm"
)

// interactionDAO 互动数据访问实现
type interactionDAO struct {
	db *database.PostgreSQL
}

// NewInteractionDAO 创建互动DAO实例
func NewInteractionDAO(db *database.PostgreSQL) InteractionDAO {
	return &interactionDAO{db: db}
}

// CreateInteraction 创建互动记录
func (d *interactionDAO) CreateInteraction(ctx context.Context, interaction *model.Interaction) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否已存在相同的互动
		var count int64
		err := tx.Model(&model.Interaction{}).
			Where("user_id = ? AND object_id = ? AND object_type = ? AND interaction_type = ?",
				interaction.UserID, interaction.ObjectID, interaction.ObjectType, interaction.InteractionType).
			Count(&count).Error
		if err != nil {
			return err
		}

		if count > 0 {
			return fmt.Errorf("互动已存在")
		}

		// 创建互动记录
		if err := tx.Create(interaction).Error; err != nil {
			return err
		}

		// 更新统计
		return d.updateStatsInTx(tx, interaction.ObjectID, interaction.ObjectType, interaction.InteractionType, 1)
	})
}

// DeleteInteraction 删除互动记录
func (d *interactionDAO) DeleteInteraction(ctx context.Context, userID, objectID int64, objectType, interactionType string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除互动记录
		result := tx.Where("user_id = ? AND object_id = ? AND object_type = ? AND interaction_type = ?",
			userID, objectID, objectType, interactionType).
			Delete(&model.Interaction{})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("互动记录不存在")
		}

		// 更新统计
		return d.updateStatsInTx(tx, objectID, objectType, interactionType, -1)
	})
}

// GetInteraction 获取互动记录
func (d *interactionDAO) GetInteraction(ctx context.Context, userID, objectID int64, objectType, interactionType string) (*model.Interaction, error) {
	var interaction model.Interaction
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND object_id = ? AND object_type = ? AND interaction_type = ?",
			userID, objectID, objectType, interactionType).
		First(&interaction).Error

	if err != nil {
		return nil, err
	}
	return &interaction, nil
}

// BatchCheckInteractions 批量检查互动状态
func (d *interactionDAO) BatchCheckInteractions(ctx context.Context, query *model.BatchInteractionQuery) (map[int64]bool, error) {
	if len(query.ObjectIDs) == 0 {
		return make(map[int64]bool), nil
	}

	var interactions []model.Interaction
	err := d.db.WithContext(ctx).
		Select("object_id").
		Where("user_id = ? AND object_id IN ? AND object_type = ? AND interaction_type = ?",
			query.UserID, query.ObjectIDs, query.ObjectType, query.InteractionType).
		Find(&interactions).Error

	if err != nil {
		return nil, err
	}

	result := make(map[int64]bool)
	for _, objectID := range query.ObjectIDs {
		result[objectID] = false
	}

	for _, interaction := range interactions {
		result[interaction.ObjectID] = true
	}

	return result, nil
}

// BatchGetInteractions 批量获取互动记录
func (d *interactionDAO) BatchGetInteractions(ctx context.Context, userID int64, objectIDs []int64, objectType, interactionType string) ([]*model.Interaction, error) {
	if len(objectIDs) == 0 {
		return []*model.Interaction{}, nil
	}

	var interactions []*model.Interaction
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND object_id IN ? AND object_type = ? AND interaction_type = ?",
			userID, objectIDs, objectType, interactionType).
		Find(&interactions).Error

	return interactions, err
}

// GetUserInteractions 获取用户互动列表
func (d *interactionDAO) GetUserInteractions(ctx context.Context, query *model.InteractionQuery) ([]*model.Interaction, int64, error) {
	dbQuery := d.db.WithContext(ctx).Model(&model.Interaction{}).
		Where("user_id = ?", query.UserID)

	// 添加过滤条件
	if query.ObjectType != "" {
		dbQuery = dbQuery.Where("object_type = ?", query.ObjectType)
	}
	if query.InteractionType != "" {
		dbQuery = dbQuery.Where("interaction_type = ?", query.InteractionType)
	}

	// 获取总数
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if query.Page > 0 && query.PageSize > 0 {
		offset := (query.Page - 1) * query.PageSize
		dbQuery = dbQuery.Offset(int(offset)).Limit(int(query.PageSize))
	}

	// 排序
	orderBy := "created_at DESC"
	if query.SortBy != "" {
		direction := "DESC"
		if query.SortOrder == model.SortOrderAsc {
			direction = "ASC"
		}
		orderBy = fmt.Sprintf("%s %s", query.SortBy, direction)
	}
	dbQuery = dbQuery.Order(orderBy)

	var interactions []*model.Interaction
	err := dbQuery.Find(&interactions).Error
	return interactions, total, err
}

// GetObjectInteractions 获取对象互动列表
func (d *interactionDAO) GetObjectInteractions(ctx context.Context, query *model.InteractionQuery) ([]*model.Interaction, int64, error) {
	dbQuery := d.db.WithContext(ctx).Model(&model.Interaction{}).
		Where("object_id = ? AND object_type = ?", query.ObjectID, query.ObjectType)

	// 添加过滤条件
	if query.InteractionType != "" {
		dbQuery = dbQuery.Where("interaction_type = ?", query.InteractionType)
	}

	// 获取总数
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if query.Page > 0 && query.PageSize > 0 {
		offset := (query.Page - 1) * query.PageSize
		dbQuery = dbQuery.Offset(int(offset)).Limit(int(query.PageSize))
	}

	// 排序
	orderBy := "created_at DESC"
	if query.SortBy != "" {
		direction := "DESC"
		if query.SortOrder == model.SortOrderAsc {
			direction = "ASC"
		}
		orderBy = fmt.Sprintf("%s %s", query.SortBy, direction)
	}
	dbQuery = dbQuery.Order(orderBy)

	var interactions []*model.Interaction
	err := dbQuery.Find(&interactions).Error
	return interactions, total, err
}

// GetInteractionStats 获取互动统计
func (d *interactionDAO) GetInteractionStats(ctx context.Context, objectID int64, objectType string) (*model.InteractionStats, error) {
	var stats model.InteractionStats
	err := d.db.WithContext(ctx).
		Where("object_id = ? AND object_type = ?", objectID, objectType).
		First(&stats).Error

	if err == gorm.ErrRecordNotFound {
		// 如果统计记录不存在，创建一个空的
		stats = model.InteractionStats{
			ObjectID:   objectID,
			ObjectType: objectType,
			UpdatedAt:  time.Now(),
		}
		if createErr := d.db.WithContext(ctx).Create(&stats).Error; createErr != nil {
			return nil, createErr
		}
	} else if err != nil {
		return nil, err
	}

	return &stats, nil
}

// BatchGetInteractionStats 批量获取互动统计
func (d *interactionDAO) BatchGetInteractionStats(ctx context.Context, query *model.InteractionStatsQuery) ([]*model.InteractionStats, error) {
	if len(query.ObjectIDs) == 0 {
		return []*model.InteractionStats{}, nil
	}

	var stats []*model.InteractionStats
	err := d.db.WithContext(ctx).
		Where("object_id IN ? AND object_type = ?", query.ObjectIDs, query.ObjectType).
		Find(&stats).Error

	return stats, err
}

// UpdateInteractionStats 更新互动统计
func (d *interactionDAO) UpdateInteractionStats(ctx context.Context, objectID int64, objectType, interactionType string, delta int64) error {
	return d.updateStatsInTx(d.db.WithContext(ctx), objectID, objectType, interactionType, delta)
}

// updateStatsInTx 在事务中更新统计（内部方法）
func (d *interactionDAO) updateStatsInTx(tx *gorm.DB, objectID int64, objectType, interactionType string, delta int64) error {
	// 构建更新字段
	var updateField string
	switch interactionType {
	case model.InteractionTypeLike:
		updateField = "like_count"
	case model.InteractionTypeFavorite:
		updateField = "favorite_count"
	case model.InteractionTypeShare:
		updateField = "share_count"
	case model.InteractionTypeRepost:
		updateField = "repost_count"
	default:
		return fmt.Errorf("不支持的互动类型: %s", interactionType)
	}

	// 使用 UPSERT 操作
	return tx.Exec(fmt.Sprintf(`
		INSERT INTO interaction_stats (object_id, object_type, %s, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (object_id, object_type)
		DO UPDATE SET %s = interaction_stats.%s + ?, updated_at = ?
	`, updateField, updateField, updateField),
		objectID, objectType, delta, time.Now(),
		delta, time.Now()).Error
}

// GetInteractionCount 获取互动计数
func (d *interactionDAO) GetInteractionCount(ctx context.Context, objectID int64, objectType, interactionType string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Interaction{}).
		Where("object_id = ? AND object_type = ? AND interaction_type = ?",
			objectID, objectType, interactionType).
		Count(&count).Error
	return count, err
}

// IncrementInteractionCount 增加互动计数
func (d *interactionDAO) IncrementInteractionCount(ctx context.Context, objectID int64, objectType, interactionType string) error {
	return d.UpdateInteractionStats(ctx, objectID, objectType, interactionType, 1)
}

// DecrementInteractionCount 减少互动计数
func (d *interactionDAO) DecrementInteractionCount(ctx context.Context, objectID int64, objectType, interactionType string) error {
	return d.UpdateInteractionStats(ctx, objectID, objectType, interactionType, -1)
}

// GetHotObjects 获取热门对象（简化实现，实际应该从Redis获取）
func (d *interactionDAO) GetHotObjects(ctx context.Context, objectType, interactionType string, limit int32) ([]*model.HotObject, error) {
	// 这里简化实现，实际应该从Redis的有序集合中获取
	var stats []*model.InteractionStats

	query := d.db.WithContext(ctx).
		Where("object_type = ?", objectType).
		Limit(int(limit))

	// 根据互动类型排序
	switch interactionType {
	case model.InteractionTypeLike:
		query = query.Order("like_count DESC")
	case model.InteractionTypeFavorite:
		query = query.Order("favorite_count DESC")
	case model.InteractionTypeShare:
		query = query.Order("share_count DESC")
	case model.InteractionTypeRepost:
		query = query.Order("repost_count DESC")
	default:
		query = query.Order("(like_count + favorite_count + share_count + repost_count) DESC")
	}

	if err := query.Find(&stats).Error; err != nil {
		return nil, err
	}

	var hotObjects []*model.HotObject
	for _, stat := range stats {
		var score float64
		var interactionCount int64

		switch interactionType {
		case model.InteractionTypeLike:
			score = float64(stat.LikeCount)
			interactionCount = stat.LikeCount
		case model.InteractionTypeFavorite:
			score = float64(stat.FavoriteCount)
			interactionCount = stat.FavoriteCount
		case model.InteractionTypeShare:
			score = float64(stat.ShareCount)
			interactionCount = stat.ShareCount
		case model.InteractionTypeRepost:
			score = float64(stat.RepostCount)
			interactionCount = stat.RepostCount
		default:
			score = float64(stat.LikeCount + stat.FavoriteCount + stat.ShareCount + stat.RepostCount)
			interactionCount = stat.LikeCount + stat.FavoriteCount + stat.ShareCount + stat.RepostCount
		}

		hotObjects = append(hotObjects, &model.HotObject{
			ObjectID:         stat.ObjectID,
			ObjectType:       stat.ObjectType,
			Score:            score,
			InteractionCount: interactionCount,
			LastActiveTime:   stat.UpdatedAt,
		})
	}

	return hotObjects, nil
}

// UpdateHotScore 更新热度分数（简化实现）
func (d *interactionDAO) UpdateHotScore(ctx context.Context, objectID int64, objectType string, score float64) error {
	// 这里简化实现，实际应该更新Redis中的有序集合
	// 暂时更新数据库中的统计表的更新时间
	return d.db.WithContext(ctx).Model(&model.InteractionStats{}).
		Where("object_id = ? AND object_type = ?", objectID, objectType).
		Update("updated_at", time.Now()).Error
}
