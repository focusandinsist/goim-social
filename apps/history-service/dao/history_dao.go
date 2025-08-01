package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"goim-social/apps/history-service/model"
	"goim-social/pkg/database"
)

// historyDAO 历史记录数据访问实现
type historyDAO struct {
	db *database.PostgreSQL
}

// NewHistoryDAO 创建历史记录DAO实例
func NewHistoryDAO(db *database.PostgreSQL) HistoryDAO {
	return &historyDAO{
		db: db,
	}
}

// CreateHistory 创建历史记录
func (d *historyDAO) CreateHistory(ctx context.Context, record *model.HistoryRecord) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建历史记录
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		// 异步更新统计数据
		go func() {
			d.updateStatsAsync(record)
		}()

		return nil
	})
}

// BatchCreateHistory 批量创建历史记录
func (d *historyDAO) BatchCreateHistory(ctx context.Context, records []*model.HistoryRecord) error {
	if len(records) == 0 {
		return nil
	}

	if len(records) > model.MaxBatchCreateSize {
		return fmt.Errorf("批量创建数量超过限制: %d", model.MaxBatchCreateSize)
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 批量创建历史记录
		if err := tx.CreateInBatches(records, 100).Error; err != nil {
			return err
		}

		// 异步更新统计数据
		go func() {
			for _, record := range records {
				d.updateStatsAsync(record)
			}
		}()

		return nil
	})
}

// GetUserHistory 获取用户历史记录
func (d *historyDAO) GetUserHistory(ctx context.Context, params *model.GetUserHistoryParams) ([]*model.HistoryRecord, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.HistoryRecord{}).Where("user_id = ?", params.UserID)

	// 添加过滤条件
	if params.ActionType != "" {
		query = query.Where("action_type = ?", params.ActionType)
	}
	if params.ObjectType != "" {
		query = query.Where("object_type = ?", params.ObjectType)
	}
	if !params.StartTime.IsZero() {
		query = query.Where("created_at >= ?", params.StartTime)
	}
	if !params.EndTime.IsZero() {
		query = query.Where("created_at <= ?", params.EndTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if params.Page <= 0 {
		params.Page = model.DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}

	offset := (params.Page - 1) * params.PageSize
	query = query.Order("created_at DESC").Offset(int(offset)).Limit(int(params.PageSize))

	var records []*model.HistoryRecord
	err := query.Find(&records).Error
	return records, total, err
}

// GetObjectHistory 获取对象历史记录
func (d *historyDAO) GetObjectHistory(ctx context.Context, params *model.GetObjectHistoryParams) ([]*model.HistoryRecord, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.HistoryRecord{}).
		Where("object_type = ? AND object_id = ?", params.ObjectType, params.ObjectID)

	// 添加过滤条件
	if params.ActionType != "" {
		query = query.Where("action_type = ?", params.ActionType)
	}
	if !params.StartTime.IsZero() {
		query = query.Where("created_at >= ?", params.StartTime)
	}
	if !params.EndTime.IsZero() {
		query = query.Where("created_at <= ?", params.EndTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if params.Page <= 0 {
		params.Page = model.DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}

	offset := (params.Page - 1) * params.PageSize
	query = query.Order("created_at DESC").Offset(int(offset)).Limit(int(params.PageSize))

	var records []*model.HistoryRecord
	err := query.Find(&records).Error
	return records, total, err
}

// DeleteHistory 删除历史记录
func (d *historyDAO) DeleteHistory(ctx context.Context, userID int64, recordIDs []int64) (int32, error) {
	if len(recordIDs) == 0 {
		return 0, nil
	}

	if len(recordIDs) > model.MaxBatchDeleteSize {
		return 0, fmt.Errorf("批量删除数量超过限制: %d", model.MaxBatchDeleteSize)
	}

	result := d.db.WithContext(ctx).Where("user_id = ? AND id IN ?", userID, recordIDs).Delete(&model.HistoryRecord{})
	return int32(result.RowsAffected), result.Error
}

// ClearUserHistory 清空用户历史记录
func (d *historyDAO) ClearUserHistory(ctx context.Context, params *model.ClearUserHistoryParams) (int32, error) {
	query := d.db.WithContext(ctx).Where("user_id = ?", params.UserID)

	if params.ActionType != "" {
		query = query.Where("action_type = ?", params.ActionType)
	}
	if params.ObjectType != "" {
		query = query.Where("object_type = ?", params.ObjectType)
	}
	if !params.BeforeTime.IsZero() {
		query = query.Where("created_at < ?", params.BeforeTime)
	}

	result := query.Delete(&model.HistoryRecord{})
	return int32(result.RowsAffected), result.Error
}

// GetUserActionStats 获取用户行为统计
func (d *historyDAO) GetUserActionStats(ctx context.Context, userID int64, actionType string) (*model.UserActionStats, error) {
	var stats model.UserActionStats
	err := d.db.WithContext(ctx).Where("user_id = ? AND action_type = ?", userID, actionType).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果统计记录不存在，创建一个新的
			stats = model.UserActionStats{
				UserID:     userID,
				ActionType: actionType,
			}
			if createErr := d.db.WithContext(ctx).Create(&stats).Error; createErr != nil {
				return nil, createErr
			}
		} else {
			return nil, err
		}
	}
	return &stats, nil
}

// GetAllUserActionStats 获取用户所有行为统计
func (d *historyDAO) GetAllUserActionStats(ctx context.Context, userID int64) ([]*model.UserActionStats, error) {
	var statsList []*model.UserActionStats
	err := d.db.WithContext(ctx).Where("user_id = ?", userID).Find(&statsList).Error
	return statsList, err
}

// UpdateUserActionStats 更新用户行为统计
func (d *historyDAO) UpdateUserActionStats(ctx context.Context, stats *model.UserActionStats) error {
	return d.db.WithContext(ctx).Save(stats).Error
}

// IncrementUserActionStats 增加用户行为统计
func (d *historyDAO) IncrementUserActionStats(ctx context.Context, userID int64, actionType string) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先尝试更新
		result := tx.Model(&model.UserActionStats{}).
			Where("user_id = ? AND action_type = ?", userID, actionType).
			Updates(map[string]interface{}{
				"total_count":      gorm.Expr("total_count + 1"),
				"today_count":      gorm.Expr("CASE WHEN DATE(last_action_time) = DATE(?) THEN today_count + 1 ELSE 1 END", now),
				"week_count":       gorm.Expr("CASE WHEN last_action_time >= ? THEN week_count + 1 ELSE 1 END", weekStart),
				"month_count":      gorm.Expr("CASE WHEN last_action_time >= ? THEN month_count + 1 ELSE 1 END", monthStart),
				"last_action_time": now,
			})

		if result.Error != nil {
			return result.Error
		}

		// 如果没有更新到记录，说明统计记录不存在，需要创建
		if result.RowsAffected == 0 {
			stats := &model.UserActionStats{
				UserID:         userID,
				ActionType:     actionType,
				TotalCount:     1,
				TodayCount:     1,
				WeekCount:      1,
				MonthCount:     1,
				LastActionTime: now,
			}
			return tx.Create(stats).Error
		}

		return nil
	})
}

// updateStatsAsync 异步更新统计数据
func (d *historyDAO) updateStatsAsync(record *model.HistoryRecord) {
	ctx := context.Background()

	// 更新用户行为统计
	if err := d.IncrementUserActionStats(ctx, record.UserID, record.ActionType); err != nil {
		// 记录错误日志，但不影响主流程
		fmt.Printf("Failed to update user action stats: %v\n", err)
	}

	// 更新对象热度统计
	if err := d.IncrementObjectHotStats(ctx, record.ObjectType, record.ObjectID, record.ActionType); err != nil {
		// 记录错误日志，但不影响主流程
		fmt.Printf("Failed to update object hot stats: %v\n", err)
	}

	// 更新用户活跃度统计
	date := time.Date(record.CreatedAt.Year(), record.CreatedAt.Month(), record.CreatedAt.Day(), 0, 0, 0, 0, record.CreatedAt.Location())
	if err := d.IncrementUserActivityStats(ctx, record.UserID, date, 1, record.ObjectID); err != nil {
		// 记录错误日志，但不影响主流程
		fmt.Printf("Failed to update user activity stats: %v\n", err)
	}
}

// GetObjectHotStats 获取对象热度统计
func (d *historyDAO) GetObjectHotStats(ctx context.Context, objectType string, objectID int64) (*model.ObjectHotStats, error) {
	var stats model.ObjectHotStats
	err := d.db.WithContext(ctx).Where("object_type = ? AND object_id = ?", objectType, objectID).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果统计记录不存在，创建一个新的
			stats = model.ObjectHotStats{
				ObjectType: objectType,
				ObjectID:   objectID,
			}
			if createErr := d.db.WithContext(ctx).Create(&stats).Error; createErr != nil {
				return nil, createErr
			}
		} else {
			return nil, err
		}
	}
	return &stats, nil
}

// GetHotObjects 获取热门对象
func (d *historyDAO) GetHotObjects(ctx context.Context, params *model.GetHotObjectsParams) ([]*model.ObjectHotStats, error) {
	query := d.db.WithContext(ctx).Model(&model.ObjectHotStats{})

	if params.ObjectType != "" {
		query = query.Where("object_type = ?", params.ObjectType)
	}

	// 根据时间范围过滤
	if params.TimeRange != model.TimeRangeAll {
		var timeFilter time.Time
		now := time.Now()
		switch params.TimeRange {
		case model.TimeRangeToday:
			timeFilter = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case model.TimeRangeWeek:
			timeFilter = now.AddDate(0, 0, -7)
		case model.TimeRangeMonth:
			timeFilter = now.AddDate(0, -1, 0)
		}
		query = query.Where("last_active_time >= ?", timeFilter)
	}

	// 设置限制
	if params.Limit <= 0 {
		params.Limit = model.DefaultHotObjectLimit
	}
	if params.Limit > model.MaxHotObjectLimit {
		params.Limit = model.MaxHotObjectLimit
	}

	var objects []*model.ObjectHotStats
	err := query.Order("hot_score DESC").Limit(int(params.Limit)).Find(&objects).Error
	return objects, err
}

// UpdateObjectHotStats 更新对象热度统计
func (d *historyDAO) UpdateObjectHotStats(ctx context.Context, stats *model.ObjectHotStats) error {
	// 重新计算热度分数
	stats.HotScore = stats.CalculateHotScore()
	return d.db.WithContext(ctx).Save(stats).Error
}

// IncrementObjectHotStats 增加对象热度统计
func (d *historyDAO) IncrementObjectHotStats(ctx context.Context, objectType string, objectID int64, actionType string) error {
	now := time.Now()

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 构建更新字段
		updates := map[string]interface{}{
			"last_active_time": now,
		}

		switch actionType {
		case model.ActionTypeView:
			updates["view_count"] = gorm.Expr("view_count + 1")
		case model.ActionTypeLike:
			updates["like_count"] = gorm.Expr("like_count + 1")
		case model.ActionTypeFavorite:
			updates["favorite_count"] = gorm.Expr("favorite_count + 1")
		case model.ActionTypeShare:
			updates["share_count"] = gorm.Expr("share_count + 1")
		case model.ActionTypeComment:
			updates["comment_count"] = gorm.Expr("comment_count + 1")
		}

		// 先尝试更新
		result := tx.Model(&model.ObjectHotStats{}).
			Where("object_type = ? AND object_id = ?", objectType, objectID).
			Updates(updates)

		if result.Error != nil {
			return result.Error
		}

		// 如果没有更新到记录，说明统计记录不存在，需要创建
		if result.RowsAffected == 0 {
			stats := &model.ObjectHotStats{
				ObjectType:     objectType,
				ObjectID:       objectID,
				LastActiveTime: now,
			}

			// 设置初始计数
			switch actionType {
			case model.ActionTypeView:
				stats.ViewCount = 1
			case model.ActionTypeLike:
				stats.LikeCount = 1
			case model.ActionTypeFavorite:
				stats.FavoriteCount = 1
			case model.ActionTypeShare:
				stats.ShareCount = 1
			case model.ActionTypeComment:
				stats.CommentCount = 1
			}

			stats.HotScore = stats.CalculateHotScore()
			return tx.Create(stats).Error
		}

		// 更新热度分数
		var stats model.ObjectHotStats
		if err := tx.Where("object_type = ? AND object_id = ?", objectType, objectID).First(&stats).Error; err != nil {
			return err
		}
		stats.HotScore = stats.CalculateHotScore()
		return tx.Model(&stats).Update("hot_score", stats.HotScore).Error
	})
}

// GetUserActivityStats 获取用户活跃度统计
func (d *historyDAO) GetUserActivityStats(ctx context.Context, params *model.GetUserActivityStatsParams) ([]*model.UserActivityStats, error) {
	query := d.db.WithContext(ctx).Model(&model.UserActivityStats{}).Where("user_id = ?", params.UserID)

	if !params.StartDate.IsZero() {
		query = query.Where("date >= ?", params.StartDate)
	}
	if !params.EndDate.IsZero() {
		query = query.Where("date <= ?", params.EndDate)
	}

	var stats []*model.UserActivityStats
	err := query.Order("date DESC").Find(&stats).Error
	return stats, err
}

// UpdateUserActivityStats 更新用户活跃度统计
func (d *historyDAO) UpdateUserActivityStats(ctx context.Context, stats *model.UserActivityStats) error {
	// 重新计算活跃度分数
	stats.ActivityScore = stats.CalculateActivityScore()
	return d.db.WithContext(ctx).Save(stats).Error
}

// IncrementUserActivityStats 增加用户活跃度统计
func (d *historyDAO) IncrementUserActivityStats(ctx context.Context, userID int64, date time.Time, actionCount int64, objectID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先尝试更新
		result := tx.Model(&model.UserActivityStats{}).
			Where("user_id = ? AND date = ?", userID, date).
			Updates(map[string]interface{}{
				"total_actions": gorm.Expr("total_actions + ?", actionCount),
				// 这里简化处理，实际应该统计唯一对象数
				"unique_objects": gorm.Expr("unique_objects + 1"),
			})

		if result.Error != nil {
			return result.Error
		}

		// 如果没有更新到记录，说明统计记录不存在，需要创建
		if result.RowsAffected == 0 {
			stats := &model.UserActivityStats{
				UserID:        userID,
				Date:          date,
				TotalActions:  actionCount,
				UniqueObjects: 1,
			}
			stats.ActivityScore = stats.CalculateActivityScore()
			return tx.Create(stats).Error
		}

		// 更新活跃度分数
		var stats model.UserActivityStats
		if err := tx.Where("user_id = ? AND date = ?", userID, date).First(&stats).Error; err != nil {
			return err
		}
		stats.ActivityScore = stats.CalculateActivityScore()
		return tx.Model(&stats).Update("activity_score", stats.ActivityScore).Error
	})
}

// CleanOldRecords 清理旧的历史记录
func (d *historyDAO) CleanOldRecords(ctx context.Context, beforeTime time.Time) (int64, error) {
	result := d.db.WithContext(ctx).Where("created_at < ?", beforeTime).Delete(&model.HistoryRecord{})
	return result.RowsAffected, result.Error
}

// CleanOldStats 清理旧的统计数据
func (d *historyDAO) CleanOldStats(ctx context.Context, beforeTime time.Time) (int64, error) {
	result := d.db.WithContext(ctx).Where("date < ?", beforeTime).Delete(&model.UserActivityStats{})
	return result.RowsAffected, result.Error
}

// BatchGetUserHistory 批量获取用户历史记录
func (d *historyDAO) BatchGetUserHistory(ctx context.Context, userIDs []int64, actionType string, limit int32) (map[int64][]*model.HistoryRecord, error) {
	if len(userIDs) == 0 {
		return make(map[int64][]*model.HistoryRecord), nil
	}

	query := d.db.WithContext(ctx).Model(&model.HistoryRecord{}).Where("user_id IN ?", userIDs)
	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}

	if limit <= 0 {
		limit = model.DefaultPageSize
	}

	var records []*model.HistoryRecord
	err := query.Order("user_id, created_at DESC").Limit(int(limit * int32(len(userIDs)))).Find(&records).Error
	if err != nil {
		return nil, err
	}

	// 按用户ID分组
	result := make(map[int64][]*model.HistoryRecord)
	for _, record := range records {
		result[record.UserID] = append(result[record.UserID], record)
	}

	return result, nil
}

// BatchGetObjectStats 批量获取对象统计
func (d *historyDAO) BatchGetObjectStats(ctx context.Context, objectType string, objectIDs []int64) (map[int64]*model.ObjectHotStats, error) {
	if len(objectIDs) == 0 {
		return make(map[int64]*model.ObjectHotStats), nil
	}

	var statsList []*model.ObjectHotStats
	err := d.db.WithContext(ctx).Where("object_type = ? AND object_id IN ?", objectType, objectIDs).Find(&statsList).Error
	if err != nil {
		return nil, err
	}

	// 转换为map
	result := make(map[int64]*model.ObjectHotStats)
	for _, stats := range statsList {
		result[stats.ObjectID] = stats
	}

	return result, nil
}

// GetUserActionCount 获取用户行为计数
func (d *historyDAO) GetUserActionCount(ctx context.Context, userID int64, actionType string, startTime, endTime time.Time) (int64, error) {
	query := d.db.WithContext(ctx).Model(&model.HistoryRecord{}).Where("user_id = ?", userID)

	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// GetObjectActionCount 获取对象行为计数
func (d *historyDAO) GetObjectActionCount(ctx context.Context, objectType string, objectID int64, actionType string, startTime, endTime time.Time) (int64, error) {
	query := d.db.WithContext(ctx).Model(&model.HistoryRecord{}).
		Where("object_type = ? AND object_id = ?", objectType, objectID)

	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// GetTopActiveUsers 获取最活跃用户
func (d *historyDAO) GetTopActiveUsers(ctx context.Context, startTime, endTime time.Time, limit int32) ([]*model.UserActivityStats, error) {
	query := d.db.WithContext(ctx).Model(&model.UserActivityStats{})

	if !startTime.IsZero() {
		query = query.Where("date >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("date <= ?", endTime)
	}

	if limit <= 0 {
		limit = model.DefaultPageSize
	}

	var stats []*model.UserActivityStats
	err := query.Order("activity_score DESC").Limit(int(limit)).Find(&stats).Error
	return stats, err
}

// GetUserActionTrend 获取用户行为趋势
func (d *historyDAO) GetUserActionTrend(ctx context.Context, userID int64, actionType string, days int32) (map[string]int64, error) {
	if days <= 0 {
		days = 7 // 默认7天
	}

	startTime := time.Now().AddDate(0, 0, -int(days))
	query := d.db.WithContext(ctx).Model(&model.HistoryRecord{}).
		Where("user_id = ? AND created_at >= ?", userID, startTime)

	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}

	// 按日期分组统计
	var results []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	err := query.Select("DATE(created_at) as date, COUNT(*) as count").
		Group("DATE(created_at)").
		Order("date").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 转换为map
	trend := make(map[string]int64)
	for _, result := range results {
		trend[result.Date] = result.Count
	}

	return trend, nil
}

// GetRealtimeStats 获取实时统计
func (d *historyDAO) GetRealtimeStats(ctx context.Context, objectType string, objectID int64) (map[string]int64, error) {
	stats, err := d.GetObjectHotStats(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}

	result := map[string]int64{
		"view_count":     stats.ViewCount,
		"like_count":     stats.LikeCount,
		"favorite_count": stats.FavoriteCount,
		"share_count":    stats.ShareCount,
		"comment_count":  stats.CommentCount,
	}

	return result, nil
}

// UpdateRealtimeStats 更新实时统计
func (d *historyDAO) UpdateRealtimeStats(ctx context.Context, objectType string, objectID int64, actionType string, delta int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var updateField string
		switch actionType {
		case model.ActionTypeView:
			updateField = "view_count"
		case model.ActionTypeLike:
			updateField = "like_count"
		case model.ActionTypeFavorite:
			updateField = "favorite_count"
		case model.ActionTypeShare:
			updateField = "share_count"
		case model.ActionTypeComment:
			updateField = "comment_count"
		default:
			return fmt.Errorf("不支持的行为类型: %s", actionType)
		}

		// 更新统计
		result := tx.Model(&model.ObjectHotStats{}).
			Where("object_type = ? AND object_id = ?", objectType, objectID).
			UpdateColumn(updateField, gorm.Expr(updateField+" + ?", delta))

		if result.Error != nil {
			return result.Error
		}

		// 如果没有更新到记录，创建新记录
		if result.RowsAffected == 0 {
			stats := &model.ObjectHotStats{
				ObjectType:     objectType,
				ObjectID:       objectID,
				LastActiveTime: time.Now(),
			}

			// 设置初始值
			switch actionType {
			case model.ActionTypeView:
				stats.ViewCount = delta
			case model.ActionTypeLike:
				stats.LikeCount = delta
			case model.ActionTypeFavorite:
				stats.FavoriteCount = delta
			case model.ActionTypeShare:
				stats.ShareCount = delta
			case model.ActionTypeComment:
				stats.CommentCount = delta
			}

			stats.HotScore = stats.CalculateHotScore()
			return tx.Create(stats).Error
		}

		return nil
	})
}
