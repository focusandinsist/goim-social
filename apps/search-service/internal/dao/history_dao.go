package dao

import (
	"context"
	"fmt"
	"time"

	"goim-social/apps/search-service/internal/model"
	"goim-social/pkg/database"
	"goim-social/pkg/logger"
)

// historyDAO 搜索历史数据访问对象
type historyDAO struct {
	db     *database.PostgreSQL
	logger logger.Logger
}

// NewHistoryDAO 创建搜索历史DAO实例
func NewHistoryDAO(db *database.PostgreSQL, log logger.Logger) HistoryDAO {
	return &historyDAO{
		db:     db,
		logger: log,
	}
}

// ============ 搜索历史管理 ============

// CreateSearchHistory 创建搜索历史
func (d *historyDAO) CreateSearchHistory(ctx context.Context, history *model.SearchHistory) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(history).Error; err != nil {
		d.logger.Error(ctx, "Failed to create search history",
			logger.F("user_id", history.UserID),
			logger.F("query", history.Query),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to create search history: %v", err)
	}

	d.logger.Debug(ctx, "Search history created",
		logger.F("user_id", history.UserID),
		logger.F("query", history.Query),
		logger.F("search_type", history.SearchType))
	return nil
}

// GetUserSearchHistory 获取用户搜索历史
func (d *historyDAO) GetUserSearchHistory(ctx context.Context, userID int64, searchType string, limit int) ([]*model.SearchHistory, error) {
	var histories []*model.SearchHistory
	db := d.db.GetDB()

	query := db.WithContext(ctx).Where("user_id = ?", userID)

	if searchType != "" {
		query = query.Where("search_type = ?", searchType)
	}

	if limit <= 0 {
		limit = 50 // 默认限制
	}

	if err := query.Order("search_time DESC").Limit(limit).Find(&histories).Error; err != nil {
		d.logger.Error(ctx, "Failed to get user search history",
			logger.F("user_id", userID),
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get user search history: %v", err)
	}

	return histories, nil
}

// DeleteUserSearchHistory 删除用户搜索历史
func (d *historyDAO) DeleteUserSearchHistory(ctx context.Context, userID int64, historyID int64) error {
	db := d.db.GetDB()

	result := db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, historyID).Delete(&model.SearchHistory{})
	if result.Error != nil {
		d.logger.Error(ctx, "Failed to delete search history",
			logger.F("user_id", userID),
			logger.F("history_id", historyID),
			logger.F("error", result.Error.Error()))
		return fmt.Errorf("failed to delete search history: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("search history not found")
	}

	d.logger.Debug(ctx, "Search history deleted",
		logger.F("user_id", userID),
		logger.F("history_id", historyID))
	return nil
}

// ClearUserSearchHistory 清空用户搜索历史
func (d *historyDAO) ClearUserSearchHistory(ctx context.Context, userID int64, searchType string) error {
	db := d.db.GetDB()

	query := db.WithContext(ctx).Where("user_id = ?", userID)
	if searchType != "" {
		query = query.Where("search_type = ?", searchType)
	}

	if err := query.Delete(&model.SearchHistory{}).Error; err != nil {
		d.logger.Error(ctx, "Failed to clear user search history",
			logger.F("user_id", userID),
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to clear user search history: %v", err)
	}

	d.logger.Info(ctx, "User search history cleared",
		logger.F("user_id", userID),
		logger.F("search_type", searchType))
	return nil
}

// ============ 热门搜索管理 ============

// UpdateHotSearch 更新热门搜索
func (d *historyDAO) UpdateHotSearch(ctx context.Context, query string, searchType string) error {
	db := d.db.GetDB()

	// 尝试更新现有记录
	var hotSearch model.HotSearch
	err := db.WithContext(ctx).Where("query = ? AND search_type = ?", query, searchType).First(&hotSearch).Error

	if err != nil {
		// 记录不存在，创建新记录
		hotSearch = model.HotSearch{
			Query:          query,
			SearchType:     searchType,
			SearchCount:    1,
			LastSearchTime: time.Now(),
		}

		if err := db.WithContext(ctx).Create(&hotSearch).Error; err != nil {
			d.logger.Error(ctx, "Failed to create hot search",
				logger.F("query", query),
				logger.F("search_type", searchType),
				logger.F("error", err.Error()))
			return fmt.Errorf("failed to create hot search: %v", err)
		}
	} else {
		// 记录存在，更新计数和时间
		hotSearch.SearchCount++
		hotSearch.LastSearchTime = time.Now()

		if err := db.WithContext(ctx).Save(&hotSearch).Error; err != nil {
			d.logger.Error(ctx, "Failed to update hot search",
				logger.F("query", query),
				logger.F("search_type", searchType),
				logger.F("error", err.Error()))
			return fmt.Errorf("failed to update hot search: %v", err)
		}
	}

	return nil
}

// GetHotSearches 获取热门搜索
func (d *historyDAO) GetHotSearches(ctx context.Context, searchType string, limit int) ([]*model.HotSearch, error) {
	var hotSearches []*model.HotSearch
	db := d.db.GetDB()

	query := db.WithContext(ctx)
	if searchType != "" {
		query = query.Where("search_type = ?", searchType)
	}

	if limit <= 0 {
		limit = 10 // 默认限制
	}

	if err := query.Order("search_count DESC, last_search_time DESC").Limit(limit).Find(&hotSearches).Error; err != nil {
		d.logger.Error(ctx, "Failed to get hot searches",
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get hot searches: %v", err)
	}

	return hotSearches, nil
}

// CleanupOldHotSearches 清理过期热门搜索
func (d *historyDAO) CleanupOldHotSearches(ctx context.Context, days int) error {
	db := d.db.GetDB()

	cutoffTime := time.Now().AddDate(0, 0, -days)

	result := db.WithContext(ctx).Where("last_search_time < ?", cutoffTime).Delete(&model.HotSearch{})
	if result.Error != nil {
		d.logger.Error(ctx, "Failed to cleanup old hot searches",
			logger.F("days", days),
			logger.F("error", result.Error.Error()))
		return fmt.Errorf("failed to cleanup old hot searches: %v", result.Error)
	}

	d.logger.Info(ctx, "Old hot searches cleaned up",
		logger.F("days", days),
		logger.F("deleted_count", result.RowsAffected))
	return nil
}

// ============ 搜索分析管理 ============

// CreateSearchAnalytics 创建搜索分析记录
func (d *historyDAO) CreateSearchAnalytics(ctx context.Context, analytics *model.SearchAnalytics) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(analytics).Error; err != nil {
		d.logger.Error(ctx, "Failed to create search analytics",
			logger.F("query_hash", analytics.QueryHash),
			logger.F("search_type", analytics.SearchType),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to create search analytics: %v", err)
	}

	return nil
}

// GetSearchAnalytics 获取搜索分析数据
func (d *historyDAO) GetSearchAnalytics(ctx context.Context, startTime, endTime string, searchType string) ([]*model.SearchAnalytics, error) {
	var analytics []*model.SearchAnalytics
	db := d.db.GetDB()

	query := db.WithContext(ctx)

	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}

	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	if searchType != "" {
		query = query.Where("search_type = ?", searchType)
	}

	if err := query.Order("created_at DESC").Find(&analytics).Error; err != nil {
		d.logger.Error(ctx, "Failed to get search analytics",
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get search analytics: %v", err)
	}

	return analytics, nil
}

// GetSearchPerformanceStats 获取搜索性能统计
func (d *historyDAO) GetSearchPerformanceStats(ctx context.Context, timeRange string) (map[string]interface{}, error) {
	db := d.db.GetDB()

	// 计算时间范围
	var startTime time.Time
	switch timeRange {
	case "1h":
		startTime = time.Now().Add(-1 * time.Hour)
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
	case "7d":
		startTime = time.Now().AddDate(0, 0, -7)
	case "30d":
		startTime = time.Now().AddDate(0, 0, -30)
	default:
		startTime = time.Now().Add(-24 * time.Hour)
	}

	// 查询统计数据
	var stats struct {
		TotalSearches   int64   `json:"total_searches"`
		AvgResponseTime float64 `json:"avg_response_time"`
		MaxResponseTime int     `json:"max_response_time"`
		MinResponseTime int     `json:"min_response_time"`
	}

	err := db.WithContext(ctx).Model(&model.SearchAnalytics{}).
		Where("created_at >= ?", startTime).
		Select("COUNT(*) as total_searches, AVG(execution_time_ms) as avg_response_time, MAX(execution_time_ms) as max_response_time, MIN(execution_time_ms) as min_response_time").
		Scan(&stats).Error

	if err != nil {
		d.logger.Error(ctx, "Failed to get search performance stats",
			logger.F("time_range", timeRange),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get search performance stats: %v", err)
	}

	result := map[string]interface{}{
		"total_searches":    stats.TotalSearches,
		"avg_response_time": stats.AvgResponseTime,
		"max_response_time": stats.MaxResponseTime,
		"min_response_time": stats.MinResponseTime,
		"time_range":        timeRange,
		"start_time":        startTime,
	}

	return result, nil
}

// ============ 用户偏好管理 ============

// CreateOrUpdateUserPreference 创建或更新用户搜索偏好
func (d *historyDAO) CreateOrUpdateUserPreference(ctx context.Context, preference *model.UserSearchPreference) error {
	db := d.db.GetDB()

	// 使用UPSERT操作
	if err := db.WithContext(ctx).Save(preference).Error; err != nil {
		d.logger.Error(ctx, "Failed to create or update user preference",
			logger.F("user_id", preference.UserID),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to create or update user preference: %v", err)
	}

	d.logger.Debug(ctx, "User preference saved",
		logger.F("user_id", preference.UserID))
	return nil
}

// GetUserPreference 获取用户搜索偏好
func (d *historyDAO) GetUserPreference(ctx context.Context, userID int64) (*model.UserSearchPreference, error) {
	var preference model.UserSearchPreference
	db := d.db.GetDB()

	err := db.WithContext(ctx).Where("user_id = ?", userID).First(&preference).Error
	if err != nil {
		if err.Error() == "record not found" {
			// 返回默认偏好
			return &model.UserSearchPreference{
				UserID:             userID,
				PreferredTypes:     []string{model.SearchTypeContent, model.SearchTypeUser},
				SearchFilters:      make(map[string]interface{}),
				SortPreferences:    make(map[string]string),
				LanguagePreference: "zh",
				ResultsPerPage:     model.DefaultPageSize,
				EnableSuggestions:  true,
				EnableHistory:      true,
			}, nil
		}

		d.logger.Error(ctx, "Failed to get user preference",
			logger.F("user_id", userID),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get user preference: %v", err)
	}

	return &preference, nil
}

// DeleteUserPreference 删除用户搜索偏好
func (d *historyDAO) DeleteUserPreference(ctx context.Context, userID int64) error {
	db := d.db.GetDB()

	result := db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.UserSearchPreference{})
	if result.Error != nil {
		d.logger.Error(ctx, "Failed to delete user preference",
			logger.F("user_id", userID),
			logger.F("error", result.Error.Error()))
		return fmt.Errorf("failed to delete user preference: %v", result.Error)
	}

	d.logger.Debug(ctx, "User preference deleted",
		logger.F("user_id", userID))
	return nil
}

// ============ 索引配置管理 ============

// CreateSearchIndex 创建搜索索引配置
func (d *historyDAO) CreateSearchIndex(ctx context.Context, index *model.SearchIndex) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(index).Error; err != nil {
		d.logger.Error(ctx, "Failed to create search index config",
			logger.F("index_name", index.IndexName),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to create search index config: %v", err)
	}

	d.logger.Info(ctx, "Search index config created",
		logger.F("index_name", index.IndexName),
		logger.F("index_type", index.IndexType))
	return nil
}

// GetSearchIndex 获取搜索索引配置
func (d *historyDAO) GetSearchIndex(ctx context.Context, indexName string) (*model.SearchIndex, error) {
	var index model.SearchIndex
	db := d.db.GetDB()

	err := db.WithContext(ctx).Where("index_name = ?", indexName).First(&index).Error
	if err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("search index config not found")
		}

		d.logger.Error(ctx, "Failed to get search index config",
			logger.F("index_name", indexName),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get search index config: %v", err)
	}

	return &index, nil
}

// UpdateSearchIndex 更新搜索索引配置
func (d *historyDAO) UpdateSearchIndex(ctx context.Context, index *model.SearchIndex) error {
	db := d.db.GetDB()

	if err := db.WithContext(ctx).Save(index).Error; err != nil {
		d.logger.Error(ctx, "Failed to update search index config",
			logger.F("index_name", index.IndexName),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to update search index config: %v", err)
	}

	d.logger.Info(ctx, "Search index config updated",
		logger.F("index_name", index.IndexName))
	return nil
}

// ListSearchIndices 列出所有搜索索引配置
func (d *historyDAO) ListSearchIndices(ctx context.Context, isActive bool) ([]*model.SearchIndex, error) {
	var indices []*model.SearchIndex
	db := d.db.GetDB()

	query := db.WithContext(ctx)
	if isActive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Order("created_at DESC").Find(&indices).Error; err != nil {
		d.logger.Error(ctx, "Failed to list search indices",
			logger.F("is_active", isActive),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to list search indices: %v", err)
	}

	return indices, nil
}

// ============ 同步状态管理 ============

// CreateSyncStatus 创建同步状态
func (d *historyDAO) CreateSyncStatus(ctx context.Context, status *model.SyncStatus) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(status).Error; err != nil {
		d.logger.Error(ctx, "Failed to create sync status",
			logger.F("source_table", status.SourceTable),
			logger.F("target_index", status.TargetIndex),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to create sync status: %v", err)
	}

	d.logger.Debug(ctx, "Sync status created",
		logger.F("source_table", status.SourceTable),
		logger.F("target_index", status.TargetIndex))
	return nil
}

// UpdateSyncStatus 更新同步状态
func (d *historyDAO) UpdateSyncStatus(ctx context.Context, status *model.SyncStatus) error {
	db := d.db.GetDB()

	if err := db.WithContext(ctx).Save(status).Error; err != nil {
		d.logger.Error(ctx, "Failed to update sync status",
			logger.F("source_table", status.SourceTable),
			logger.F("target_index", status.TargetIndex),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to update sync status: %v", err)
	}

	d.logger.Debug(ctx, "Sync status updated",
		logger.F("source_table", status.SourceTable),
		logger.F("target_index", status.TargetIndex),
		logger.F("sync_status", status.SyncStatus))
	return nil
}

// GetSyncStatus 获取同步状态
func (d *historyDAO) GetSyncStatus(ctx context.Context, sourceTable, targetIndex string) (*model.SyncStatus, error) {
	var status model.SyncStatus
	db := d.db.GetDB()

	err := db.WithContext(ctx).Where("source_table = ? AND target_index = ?", sourceTable, targetIndex).First(&status).Error
	if err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("sync status not found")
		}

		d.logger.Error(ctx, "Failed to get sync status",
			logger.F("source_table", sourceTable),
			logger.F("target_index", targetIndex),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get sync status: %v", err)
	}

	return &status, nil
}

// ListSyncStatuses 列出所有同步状态
func (d *historyDAO) ListSyncStatuses(ctx context.Context, sourceService string) ([]*model.SyncStatus, error) {
	var statuses []*model.SyncStatus
	db := d.db.GetDB()

	query := db.WithContext(ctx)
	if sourceService != "" {
		query = query.Where("source_service = ?", sourceService)
	}

	if err := query.Order("updated_at DESC").Find(&statuses).Error; err != nil {
		d.logger.Error(ctx, "Failed to list sync statuses",
			logger.F("source_service", sourceService),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to list sync statuses: %v", err)
	}

	return statuses, nil
}
