package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"goim-social/apps/search-service/internal/model"
	"goim-social/pkg/logger"
)

// ============ 验证和默认值设置 ============

// validateSearchRequest 验证搜索请求
func (s *searchService) validateSearchRequest(req *model.SearchRequest) error {
	if req == nil {
		return fmt.Errorf("search request is nil")
	}

	if req.Query == "" && req.Type != model.SearchTypeAll {
		return fmt.Errorf("search query is required")
	}

	if len(req.Query) > 500 {
		return fmt.Errorf("search query too long (max 500 characters)")
	}

	if !model.IsValidSearchType(req.Type) {
		return fmt.Errorf("invalid search type: %s", req.Type)
	}

	if req.PageSize > s.config.MaxPageSize {
		return fmt.Errorf("page size too large (max %d)", s.config.MaxPageSize)
	}

	if req.SortBy != "" && !model.IsValidSortField(req.SortBy) {
		return fmt.Errorf("invalid sort field: %s", req.SortBy)
	}

	if req.SortOrder != "" && !model.IsValidSortOrder(req.SortOrder) {
		return fmt.Errorf("invalid sort order: %s", req.SortOrder)
	}

	return nil
}

// setDefaultValues 设置默认值
func (s *searchService) setDefaultValues(req *model.SearchRequest) {
	if req.Page <= 0 {
		req.Page = 1
	}

	if req.PageSize <= 0 {
		req.PageSize = s.config.DefaultPageSize
	}

	if req.PageSize > s.config.MaxPageSize {
		req.PageSize = s.config.MaxPageSize
	}

	if req.SortBy == "" {
		req.SortBy = model.SortByRelevance
	}

	if req.SortOrder == "" {
		req.SortOrder = model.SortOrderDesc
	}
}

// ============ 缓存键生成 ============

// generateCacheKey 生成缓存键
func (s *searchService) generateCacheKey(req *model.SearchRequest) string {
	data := fmt.Sprintf("%s:%s:%d:%d:%s:%s:%v:%d",
		req.Query, req.Type, req.Page, req.PageSize,
		req.SortBy, req.SortOrder, req.Filters, req.UserID)

	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%s%x", model.CacheKeySearchResult, hash)
}

// ============ 结果转换 ============

// convertContentResults 转换内容搜索结果
func (s *searchService) convertContentResults(results []*model.ContentSearchResult) []interface{} {
	converted := make([]interface{}, len(results))
	for i, result := range results {
		converted[i] = result
	}
	return converted
}

// convertUserResults 转换用户搜索结果
func (s *searchService) convertUserResults(results []*model.UserSearchResult) []interface{} {
	converted := make([]interface{}, len(results))
	for i, result := range results {
		converted[i] = result
	}
	return converted
}

// convertMessageResults 转换消息搜索结果
func (s *searchService) convertMessageResults(results []*model.MessageSearchResult) []interface{} {
	converted := make([]interface{}, len(results))
	for i, result := range results {
		converted[i] = result
	}
	return converted
}

// convertGroupResults 转换群组搜索结果
func (s *searchService) convertGroupResults(results []*model.GroupSearchResult) []interface{} {
	converted := make([]interface{}, len(results))
	for i, result := range results {
		converted[i] = result
	}
	return converted
}

// ============ 搜索建议辅助方法 ============

// getHistorySuggestions 获取历史搜索建议
func (s *searchService) getHistorySuggestions(ctx context.Context, query string, searchType string, userID int64, limit int) []model.SearchSuggestion {
	history, err := s.historyDAO.GetUserSearchHistory(ctx, userID, searchType, limit*2)
	if err != nil {
		s.logger.Debug(ctx, "Failed to get user search history for suggestions",
			logger.F("user_id", userID),
			logger.F("error", err.Error()))
		return []model.SearchSuggestion{}
	}

	suggestions := make([]model.SearchSuggestion, 0)
	queryLower := strings.ToLower(query)

	for _, h := range history {
		historyQueryLower := strings.ToLower(h.Query)

		// 检查是否匹配查询前缀
		if strings.HasPrefix(historyQueryLower, queryLower) && h.Query != query {
			suggestion := model.SearchSuggestion{
				Text:   h.Query,
				Score:  1.0, // 历史搜索的基础分数
				Type:   model.SuggestionTypeQuery,
				Source: model.SuggestionSourceHistory,
			}

			// 根据搜索频率调整分数
			if h.ResultCount > 0 {
				suggestion.Score += 0.5
			}

			suggestions = append(suggestions, suggestion)

			if len(suggestions) >= limit {
				break
			}
		}
	}

	return suggestions
}

// mergeSuggestions 合并搜索建议
func (s *searchService) mergeSuggestions(esSuggestions []model.SearchSuggestion, historySuggestions []model.SearchSuggestion, limit int) []model.SearchSuggestion {
	// 使用map去重
	suggestionMap := make(map[string]model.SearchSuggestion)

	// 添加ElasticSearch建议
	for _, suggestion := range esSuggestions {
		suggestionMap[suggestion.Text] = suggestion
	}

	// 添加历史建议，如果已存在则提高分数
	for _, suggestion := range historySuggestions {
		if existing, exists := suggestionMap[suggestion.Text]; exists {
			existing.Score += suggestion.Score * 0.5 // 历史建议加权
			suggestionMap[suggestion.Text] = existing
		} else {
			suggestionMap[suggestion.Text] = suggestion
		}
	}

	// 转换为切片并排序
	merged := make([]model.SearchSuggestion, 0, len(suggestionMap))
	for _, suggestion := range suggestionMap {
		merged = append(merged, suggestion)
	}

	// 按分数排序
	for i := 0; i < len(merged)-1; i++ {
		for j := i + 1; j < len(merged); j++ {
			if merged[i].Score < merged[j].Score {
				merged[i], merged[j] = merged[j], merged[i]
			}
		}
	}

	// 限制数量
	if len(merged) > limit {
		merged = merged[:limit]
	}

	return merged
}

// ============ 搜索记录方法 ============

// recordSearchHistory 记录搜索历史
func (s *searchService) recordSearchHistory(ctx context.Context, req *model.SearchRequest, response *model.SearchResponse) {
	if req.UserID <= 0 {
		return
	}

	history := &model.SearchHistory{
		UserID:      req.UserID,
		Query:       req.Query,
		SearchType:  req.Type,
		ResultCount: 0,
		SearchTime:  time.Now(),
	}

	if response != nil {
		history.ResultCount = int(response.Total)
	}

	if err := s.historyDAO.CreateSearchHistory(ctx, history); err != nil {
		s.logger.Warn(ctx, "Failed to record search history",
			logger.F("user_id", req.UserID),
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
	}
}

// updateHotSearch 更新热门搜索
func (s *searchService) updateHotSearch(ctx context.Context, query string, searchType string) {
	if query == "" {
		return
	}

	if err := s.historyDAO.UpdateHotSearch(ctx, query, searchType); err != nil {
		s.logger.Warn(ctx, "Failed to update hot search",
			logger.F("query", query),
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
	}
}

// recordSearchEvent 记录搜索事件
func (s *searchService) recordSearchEvent(ctx context.Context, req *model.SearchRequest, response *model.SearchResponse, duration time.Duration, fromCache bool) {
	if !s.config.EventEnabled {
		return
	}

	event := &SearchEvent{
		UserID:     req.UserID,
		Query:      req.Query,
		SearchType: req.Type,
		Duration:   duration.Milliseconds(),
		Filters:    req.Filters,
		Timestamp:  time.Now().Unix(),
	}

	if response != nil {
		event.ResultCount = int(response.Total)
	}

	// 从上下文获取请求信息
	if requestID := ctx.Value("request_id"); requestID != nil {
		event.RequestID = fmt.Sprintf("%v", requestID)
	}

	if clientIP := ctx.Value("client_ip"); clientIP != nil {
		event.ClientIP = fmt.Sprintf("%v", clientIP)
	}

	if userAgent := ctx.Value("user_agent"); userAgent != nil {
		event.UserAgent = fmt.Sprintf("%v", userAgent)
	}

	// 记录搜索分析
	analytics := &model.SearchAnalytics{
		QueryHash:           s.generateQueryHash(req.Query, req.Type, req.Filters),
		Query:               req.Query,
		SearchType:          req.Type,
		UserID:              req.UserID,
		ExecutionTimeMs:     int(duration.Milliseconds()),
		ResultCount:         event.ResultCount,
		HitCache:            fromCache,
		ElasticsearchTimeMs: 0, // 这里可以从响应中获取ES的执行时间
		TotalHits:           int64(event.ResultCount),
		SearchDate:          time.Now(),
		CreatedAt:           time.Now(),
	}

	// 异步记录分析数据
	go func() {
		if err := s.historyDAO.CreateSearchAnalytics(context.Background(), analytics); err != nil {
			s.logger.Warn(context.Background(), "Failed to record search analytics",
				logger.F("query_hash", analytics.QueryHash),
				logger.F("error", err.Error()))
		}
	}()

	// 发布搜索事件
	if err := s.eventService.PublishSearchEvent(ctx, event); err != nil {
		s.logger.Warn(ctx, "Failed to publish search event",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
	}
}

// generateQueryHash 生成查询哈希
func (s *searchService) generateQueryHash(query string, searchType string, filters map[string]string) string {
	data := fmt.Sprintf("%s:%s:%v", query, searchType, filters)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}
