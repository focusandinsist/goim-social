package service

import (
	"context"
	"fmt"
	"time"

	"goim-social/apps/search-service/dao"
	"goim-social/apps/search-service/model"
	"goim-social/pkg/database"
	"goim-social/pkg/logger"
)

// searchService 搜索服务实现
type searchService struct {
	searchDAO    dao.SearchDAO
	historyDAO   dao.HistoryDAO
	cacheService CacheService
	eventService EventService
	config       *ServiceConfig
	logger       logger.Logger
}

// NewService 创建搜索服务实例（简化版本）
func NewService(elasticSearch *database.ElasticSearch, postgreSQL *database.PostgreSQL, log logger.Logger) SearchService {
	// 初始化DAO层
	searchDAO := dao.NewElasticsearchDAO(elasticSearch.GetClient(), log)
	historyDAO := dao.NewHistoryDAO(postgreSQL, log)

	// 初始化服务层依赖
	cacheService := NewMockCacheService()
	eventService := NewMockEventService()

	// 创建默认配置
	config := &ServiceConfig{
		DefaultPageSize:  20,
		MaxPageSize:      100,
		SearchTimeout:    5000,
		HighlightPreTag:  "<em>",
		HighlightPostTag: "</em>",
		CacheEnabled:     true,
		CacheTTL: map[string]int{
			"search_results": 300,
			"suggestions":    600,
			"hot_searches":   1800,
		},
		IndexSettings: map[string]interface{}{
			"refresh_interval":   "1s",
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
		FieldWeights: map[string]float64{
			"title":   2.0,
			"content": 1.0,
			"tags":    1.5,
		},
		EventEnabled: true,
		EventTopics: map[string]string{
			"search": "search_events",
			"index":  "index_events",
		},
	}

	return &searchService{
		searchDAO:    searchDAO,
		historyDAO:   historyDAO,
		cacheService: cacheService,
		eventService: eventService,
		config:       config,
		logger:       log,
	}
}

// NewSearchService 创建搜索服务实例（详细版本，保持向后兼容）
func NewSearchService(
	searchDAO dao.SearchDAO,
	historyDAO dao.HistoryDAO,
	cacheService CacheService,
	eventService EventService,
	config *ServiceConfig,
	log logger.Logger,
) SearchService {
	return &searchService{
		searchDAO:    searchDAO,
		historyDAO:   historyDAO,
		cacheService: cacheService,
		eventService: eventService,
		config:       config,
		logger:       log,
	}
}

// ============ 搜索功能 ============

// Search 通用搜索
func (s *searchService) Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	startTime := time.Now()

	// 验证请求
	if err := s.validateSearchRequest(req); err != nil {
		return nil, err
	}

	// 设置默认值
	s.setDefaultValues(req)

	// 生成缓存键
	cacheKey := s.generateCacheKey(req)

	// 尝试从缓存获取结果
	if s.config.CacheEnabled {
		if cachedResult, err := s.cacheService.GetSearchResult(ctx, cacheKey); err == nil {
			if response, ok := cachedResult.(*model.SearchResponse); ok {
				response.FromCache = true
				s.logger.Debug(ctx, "Search result from cache",
					logger.F("query", req.Query),
					logger.F("type", req.Type),
					logger.F("cache_key", cacheKey))

				// 记录搜索事件（异步）
				go s.recordSearchEvent(ctx, req, response, time.Since(startTime), true)
				return response, nil
			}
		}
	}

	var response *model.SearchResponse
	var err error

	// 根据搜索类型执行搜索
	switch req.Type {
	case model.SearchTypeContent:
		results, total, searchErr := s.searchDAO.SearchContent(ctx, req)
		if searchErr != nil {
			err = searchErr
		} else {
			response = &model.SearchResponse{
				Query:    req.Query,
				Type:     req.Type,
				Total:    total,
				Page:     req.Page,
				PageSize: req.PageSize,
				Results:  s.convertContentResults(results),
				Duration: time.Since(startTime).Milliseconds(),
			}
		}

	case model.SearchTypeUser:
		results, total, searchErr := s.searchDAO.SearchUsers(ctx, req)
		if searchErr != nil {
			err = searchErr
		} else {
			response = &model.SearchResponse{
				Query:    req.Query,
				Type:     req.Type,
				Total:    total,
				Page:     req.Page,
				PageSize: req.PageSize,
				Results:  s.convertUserResults(results),
				Duration: time.Since(startTime).Milliseconds(),
			}
		}

	case model.SearchTypeMessage:
		results, total, searchErr := s.searchDAO.SearchMessages(ctx, req)
		if searchErr != nil {
			err = searchErr
		} else {
			response = &model.SearchResponse{
				Query:    req.Query,
				Type:     req.Type,
				Total:    total,
				Page:     req.Page,
				PageSize: req.PageSize,
				Results:  s.convertMessageResults(results),
				Duration: time.Since(startTime).Milliseconds(),
			}
		}

	case model.SearchTypeGroup:
		results, total, searchErr := s.searchDAO.SearchGroups(ctx, req)
		if searchErr != nil {
			err = searchErr
		} else {
			response = &model.SearchResponse{
				Query:    req.Query,
				Type:     req.Type,
				Total:    total,
				Page:     req.Page,
				PageSize: req.PageSize,
				Results:  s.convertGroupResults(results),
				Duration: time.Since(startTime).Milliseconds(),
			}
		}

	case model.SearchTypeAll:
		response, err = s.searchDAO.MultiSearch(ctx, req)
		if response != nil {
			response.Duration = time.Since(startTime).Milliseconds()
		}

	default:
		err = fmt.Errorf("unsupported search type: %s", req.Type)
	}

	if err != nil {
		s.logger.Error(ctx, "Search failed",
			logger.F("query", req.Query),
			logger.F("type", req.Type),
			logger.F("error", err.Error()))

		// 记录搜索失败事件
		go s.recordSearchEvent(ctx, req, nil, time.Since(startTime), false)
		return nil, fmt.Errorf("search failed: %v", err)
	}

	// 缓存结果
	if s.config.CacheEnabled && response != nil {
		ttl := s.config.CacheTTL["search_results"]
		if ttl == 0 {
			ttl = model.DefaultSearchResultTTL
		}

		go func() {
			if cacheErr := s.cacheService.SetSearchResult(context.Background(), cacheKey, response, ttl); cacheErr != nil {
				s.logger.Warn(context.Background(), "Failed to cache search result",
					logger.F("cache_key", cacheKey),
					logger.F("error", cacheErr.Error()))
			}
		}()
	}

	// 记录搜索历史（异步）
	if req.UserID > 0 {
		go s.recordSearchHistory(ctx, req, response)
	}

	// 更新热门搜索（异步）
	go s.updateHotSearch(ctx, req.Query, req.Type)

	// 记录搜索事件（异步）
	go s.recordSearchEvent(ctx, req, response, time.Since(startTime), false)

	s.logger.Info(ctx, "Search completed",
		logger.F("query", req.Query),
		logger.F("type", req.Type),
		logger.F("total", response.Total),
		logger.F("duration_ms", response.Duration))

	return response, nil
}

// SearchContent 内容搜索
func (s *searchService) SearchContent(ctx context.Context, req *model.SearchRequest) ([]*model.ContentSearchResult, int64, error) {
	req.Type = model.SearchTypeContent
	if err := s.validateSearchRequest(req); err != nil {
		return nil, 0, err
	}

	s.setDefaultValues(req)

	var results []*model.ContentSearchResult
	var total int64
	var err error

	// 使用ElasticSearch搜索
	results, total, err = s.searchDAO.SearchContent(ctx, req)
	if err != nil {
		s.logger.Error(ctx, "Content search failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		return nil, 0, fmt.Errorf("content search failed: %v", err)
	}

	// 记录搜索历史和事件（异步）
	if req.UserID > 0 {
		go s.recordSearchHistory(ctx, req, &model.SearchResponse{Total: total})
		go s.updateHotSearch(ctx, req.Query, req.Type)
	}

	return results, total, nil
}

// SearchUsers 用户搜索
func (s *searchService) SearchUsers(ctx context.Context, req *model.SearchRequest) ([]*model.UserSearchResult, int64, error) {
	req.Type = model.SearchTypeUser
	if err := s.validateSearchRequest(req); err != nil {
		return nil, 0, err
	}

	s.setDefaultValues(req)

	results, total, err := s.searchDAO.SearchUsers(ctx, req)
	if err != nil {
		s.logger.Error(ctx, "User search failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		return nil, 0, fmt.Errorf("user search failed: %v", err)
	}

	// 记录搜索历史和事件（异步）
	if req.UserID > 0 {
		go s.recordSearchHistory(ctx, req, &model.SearchResponse{Total: total})
		go s.updateHotSearch(ctx, req.Query, req.Type)
	}

	return results, total, nil
}

// SearchMessages 消息搜索
func (s *searchService) SearchMessages(ctx context.Context, req *model.SearchRequest) ([]*model.MessageSearchResult, int64, error) {
	req.Type = model.SearchTypeMessage
	if err := s.validateSearchRequest(req); err != nil {
		return nil, 0, err
	}

	s.setDefaultValues(req)

	// 消息搜索需要用户ID
	if req.UserID <= 0 {
		return nil, 0, fmt.Errorf("user ID is required for message search")
	}

	results, total, err := s.searchDAO.SearchMessages(ctx, req)
	if err != nil {
		s.logger.Error(ctx, "Message search failed",
			logger.F("query", req.Query),
			logger.F("user_id", req.UserID),
			logger.F("error", err.Error()))
		return nil, 0, fmt.Errorf("message search failed: %v", err)
	}

	// 记录搜索历史和事件（异步）
	go s.recordSearchHistory(ctx, req, &model.SearchResponse{Total: total})
	go s.updateHotSearch(ctx, req.Query, req.Type)

	return results, total, nil
}

// SearchGroups 群组搜索
func (s *searchService) SearchGroups(ctx context.Context, req *model.SearchRequest) ([]*model.GroupSearchResult, int64, error) {
	req.Type = model.SearchTypeGroup
	if err := s.validateSearchRequest(req); err != nil {
		return nil, 0, err
	}

	s.setDefaultValues(req)

	results, total, err := s.searchDAO.SearchGroups(ctx, req)
	if err != nil {
		s.logger.Error(ctx, "Group search failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		return nil, 0, fmt.Errorf("group search failed: %v", err)
	}

	// 记录搜索历史和事件（异步）
	if req.UserID > 0 {
		go s.recordSearchHistory(ctx, req, &model.SearchResponse{Total: total})
		go s.updateHotSearch(ctx, req.Query, req.Type)
	}

	return results, total, nil
}

// MultiSearch 多类型搜索
func (s *searchService) MultiSearch(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	req.Type = model.SearchTypeAll
	if err := s.validateSearchRequest(req); err != nil {
		return nil, err
	}

	s.setDefaultValues(req)

	response, err := s.searchDAO.MultiSearch(ctx, req)
	if err != nil {
		s.logger.Error(ctx, "Multi search failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("multi search failed: %v", err)
	}

	// 记录搜索历史和事件（异步）
	if req.UserID > 0 {
		go s.recordSearchHistory(ctx, req, response)
		go s.updateHotSearch(ctx, req.Query, req.Type)
	}

	return response, nil
}

// ============ 搜索建议 ============

// GetSuggestions 获取搜索建议
func (s *searchService) GetSuggestions(ctx context.Context, query string, searchType string, limit int, userID int64) ([]model.SearchSuggestion, error) {
	if query == "" {
		return []model.SearchSuggestion{}, nil
	}

	if limit <= 0 {
		limit = model.DefaultSuggestionLimit
	}
	if limit > model.MaxSuggestionLimit {
		limit = model.MaxSuggestionLimit
	}

	// 尝试从缓存获取
	if s.config.CacheEnabled {
		if suggestions, err := s.cacheService.GetSuggestions(ctx, query, searchType); err == nil {
			return suggestions, nil
		}
	}

	// 从ElasticSearch获取建议
	suggestions, err := s.searchDAO.GetSuggestions(ctx, query, searchType, limit)
	if err != nil {
		s.logger.Error(ctx, "Failed to get suggestions",
			logger.F("query", query),
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get suggestions: %v", err)
	}

	// 如果有用户ID，添加个人历史建议
	if userID > 0 {
		historySuggestions := s.getHistorySuggestions(ctx, query, searchType, userID, limit/2)
		suggestions = s.mergeSuggestions(suggestions, historySuggestions, limit)
	}

	// 缓存结果
	if s.config.CacheEnabled {
		ttl := s.config.CacheTTL["suggestions"]
		if ttl == 0 {
			ttl = 300 // 5分钟
		}
		go s.cacheService.SetSuggestions(ctx, query, searchType, suggestions, ttl)
	}

	return suggestions, nil
}

// GetAutoComplete 获取自动完成建议
func (s *searchService) GetAutoComplete(ctx context.Context, req *model.AutoCompleteRequest) (*model.AutoCompleteResponse, error) {
	if req.Query == "" {
		return &model.AutoCompleteResponse{
			Query:       req.Query,
			Suggestions: []model.SearchSuggestion{},
			Duration:    0,
		}, nil
	}

	startTime := time.Now()

	suggestions, err := s.GetSuggestions(ctx, req.Query, req.Type, req.Limit, req.UserID)
	if err != nil {
		return nil, err
	}

	response := &model.AutoCompleteResponse{
		Query:       req.Query,
		Suggestions: suggestions,
		Duration:    time.Since(startTime).Milliseconds(),
	}

	return response, nil
}

// GetHotSearches 获取热门搜索
func (s *searchService) GetHotSearches(ctx context.Context, searchType string, limit int) ([]*model.HotSearch, error) {
	if limit <= 0 {
		limit = 10
	}

	// 尝试从缓存获取
	if s.config.CacheEnabled {
		if hotSearches, err := s.cacheService.GetHotSearches(ctx, searchType); err == nil {
			if len(hotSearches) <= limit {
				return hotSearches, nil
			}
			return hotSearches[:limit], nil
		}
	}

	// 从数据库获取
	hotSearches, err := s.historyDAO.GetHotSearches(ctx, searchType, limit)
	if err != nil {
		s.logger.Error(ctx, "Failed to get hot searches",
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get hot searches: %v", err)
	}

	// 缓存结果
	if s.config.CacheEnabled {
		ttl := s.config.CacheTTL["hot_queries"]
		if ttl == 0 {
			ttl = model.DefaultHotQueriesTTL
		}
		go s.cacheService.SetHotSearches(ctx, searchType, hotSearches, ttl)
	}

	return hotSearches, nil
}

// ============ 搜索历史 ============

// GetUserSearchHistory 获取用户搜索历史
func (s *searchService) GetUserSearchHistory(ctx context.Context, userID int64, searchType string, limit int) ([]*model.SearchHistory, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	if limit <= 0 {
		limit = 50
	}

	// 尝试从缓存获取
	if s.config.CacheEnabled {
		if history, err := s.cacheService.GetUserSearchHistory(ctx, userID); err == nil {
			// 过滤搜索类型
			if searchType != "" {
				filtered := make([]*model.SearchHistory, 0)
				for _, h := range history {
					if h.SearchType == searchType {
						filtered = append(filtered, h)
					}
				}
				history = filtered
			}

			if len(history) <= limit {
				return history, nil
			}
			return history[:limit], nil
		}
	}

	// 从数据库获取
	history, err := s.historyDAO.GetUserSearchHistory(ctx, userID, searchType, limit)
	if err != nil {
		s.logger.Error(ctx, "Failed to get user search history",
			logger.F("user_id", userID),
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get user search history: %v", err)
	}

	// 缓存结果
	if s.config.CacheEnabled {
		ttl := s.config.CacheTTL["user_history"]
		if ttl == 0 {
			ttl = model.DefaultUserHistoryTTL
		}
		go s.cacheService.SetUserSearchHistory(ctx, userID, history, ttl)
	}

	return history, nil
}

// ClearUserSearchHistory 清空用户搜索历史
func (s *searchService) ClearUserSearchHistory(ctx context.Context, userID int64, searchType string) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	err := s.historyDAO.ClearUserSearchHistory(ctx, userID, searchType)
	if err != nil {
		s.logger.Error(ctx, "Failed to clear user search history",
			logger.F("user_id", userID),
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to clear user search history: %v", err)
	}

	// 清除缓存
	if s.config.CacheEnabled {
		cacheKey := fmt.Sprintf("%s%d", model.CacheKeyUserHistory, userID)
		go s.cacheService.DeleteSearchResult(ctx, cacheKey)
	}

	s.logger.Info(ctx, "User search history cleared",
		logger.F("user_id", userID),
		logger.F("search_type", searchType))

	return nil
}

// DeleteSearchHistory 删除特定搜索历史
func (s *searchService) DeleteSearchHistory(ctx context.Context, userID int64, historyID int64) error {
	if userID <= 0 || historyID <= 0 {
		return fmt.Errorf("invalid user ID or history ID")
	}

	err := s.historyDAO.DeleteUserSearchHistory(ctx, userID, historyID)
	if err != nil {
		s.logger.Error(ctx, "Failed to delete search history",
			logger.F("user_id", userID),
			logger.F("history_id", historyID),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to delete search history: %v", err)
	}

	// 清除缓存
	if s.config.CacheEnabled {
		cacheKey := fmt.Sprintf("%s%d", model.CacheKeyUserHistory, userID)
		go s.cacheService.DeleteSearchResult(ctx, cacheKey)
	}

	return nil
}

// ============ 用户偏好 ============

// GetUserPreference 获取用户搜索偏好
func (s *searchService) GetUserPreference(ctx context.Context, userID int64) (*model.UserSearchPreference, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	// 尝试从缓存获取
	if s.config.CacheEnabled {
		if preference, err := s.cacheService.GetUserPreference(ctx, userID); err == nil {
			return preference, nil
		}
	}

	// 从数据库获取
	preference, err := s.historyDAO.GetUserPreference(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get user preference",
			logger.F("user_id", userID),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get user preference: %v", err)
	}

	// 缓存结果
	if s.config.CacheEnabled {
		ttl := s.config.CacheTTL["user_preference"]
		if ttl == 0 {
			ttl = 86400 // 24小时
		}
		go s.cacheService.SetUserPreference(ctx, userID, preference, ttl)
	}

	return preference, nil
}

// UpdateUserPreference 更新用户搜索偏好
func (s *searchService) UpdateUserPreference(ctx context.Context, preference *model.UserSearchPreference) error {
	if preference == nil || preference.UserID <= 0 {
		return fmt.Errorf("invalid user preference")
	}

	err := s.historyDAO.CreateOrUpdateUserPreference(ctx, preference)
	if err != nil {
		s.logger.Error(ctx, "Failed to update user preference",
			logger.F("user_id", preference.UserID),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to update user preference: %v", err)
	}

	// 更新缓存
	if s.config.CacheEnabled {
		ttl := s.config.CacheTTL["user_preference"]
		if ttl == 0 {
			ttl = 86400 // 24小时
		}
		go s.cacheService.SetUserPreference(ctx, preference.UserID, preference, ttl)
	}

	s.logger.Info(ctx, "User preference updated",
		logger.F("user_id", preference.UserID))

	return nil
}

// ============ 搜索统计 ============

// GetSearchStats 获取搜索统计
func (s *searchService) GetSearchStats(ctx context.Context, timeRange string) (*model.SearchStats, error) {
	// 从ElasticSearch获取基础统计
	esStats, err := s.searchDAO.GetSearchStats(ctx, timeRange)
	if err != nil {
		s.logger.Error(ctx, "Failed to get search stats from ES",
			logger.F("time_range", timeRange),
			logger.F("error", err.Error()))
		// 继续执行，使用数据库统计
	}

	// 从数据库获取性能统计
	perfStats, err := s.historyDAO.GetSearchPerformanceStats(ctx, timeRange)
	if err != nil {
		s.logger.Error(ctx, "Failed to get search performance stats",
			logger.F("time_range", timeRange),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get search performance stats: %v", err)
	}

	// 合并统计数据
	stats := &model.SearchStats{
		TotalSearches:   0,
		UniqueUsers:     0,
		AvgResponseTime: 0,
		TopQueries:      make([]model.QueryStat, 0),
		SearchesByType:  make(map[string]int64),
		SearchesByHour:  make(map[string]int64),
		CacheHitRate:    0,
	}

	if esStats != nil {
		stats.TotalSearches = esStats.TotalSearches
		stats.UniqueUsers = esStats.UniqueUsers
		stats.TopQueries = esStats.TopQueries
		stats.SearchesByType = esStats.SearchesByType
		stats.SearchesByHour = esStats.SearchesByHour
		stats.CacheHitRate = esStats.CacheHitRate
	}

	// 使用数据库的性能统计
	if totalSearches, ok := perfStats["total_searches"].(int64); ok {
		stats.TotalSearches = totalSearches
	}

	if avgResponseTime, ok := perfStats["avg_response_time"].(float64); ok {
		stats.AvgResponseTime = avgResponseTime
	}

	return stats, nil
}

// GetSearchAnalytics 获取搜索分析数据
func (s *searchService) GetSearchAnalytics(ctx context.Context, startTime, endTime string, searchType string) ([]*model.SearchAnalytics, error) {
	analytics, err := s.historyDAO.GetSearchAnalytics(ctx, startTime, endTime, searchType)
	if err != nil {
		s.logger.Error(ctx, "Failed to get search analytics",
			logger.F("start_time", startTime),
			logger.F("end_time", endTime),
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get search analytics: %v", err)
	}

	return analytics, nil
}
