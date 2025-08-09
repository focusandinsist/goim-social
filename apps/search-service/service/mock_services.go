package service

import (
	"context"

	"goim-social/apps/search-service/model"
)

// ============ Mock Cache Service ============

// mockCacheService Mock缓存服务实现
type mockCacheService struct{}

// NewMockCacheService 创建Mock缓存服务
func NewMockCacheService() CacheService {
	return &mockCacheService{}
}

func (m *mockCacheService) GetSearchResult(ctx context.Context, cacheKey string) (interface{}, error) {
	// Mock实现：总是返回缓存未命中
	return nil, ErrCacheMiss
}

func (m *mockCacheService) SetSearchResult(ctx context.Context, cacheKey string, result interface{}, ttl int) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockCacheService) DeleteSearchResult(ctx context.Context, cacheKey string) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockCacheService) GetUserSearchHistory(ctx context.Context, userID int64) ([]*model.SearchHistory, error) {
	// Mock实现：返回空历史
	return []*model.SearchHistory{}, nil
}

func (m *mockCacheService) SetUserSearchHistory(ctx context.Context, userID int64, history []*model.SearchHistory, ttl int) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockCacheService) GetUserPreference(ctx context.Context, userID int64) (*model.UserSearchPreference, error) {
	// Mock实现：返回默认偏好
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

func (m *mockCacheService) SetUserPreference(ctx context.Context, userID int64, preference *model.UserSearchPreference, ttl int) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockCacheService) GetHotSearches(ctx context.Context, searchType string) ([]*model.HotSearch, error) {
	// Mock实现：返回空热门搜索
	return []*model.HotSearch{}, nil
}

func (m *mockCacheService) SetHotSearches(ctx context.Context, searchType string, hotSearches []*model.HotSearch, ttl int) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockCacheService) GetSuggestions(ctx context.Context, query string, searchType string) ([]model.SearchSuggestion, error) {
	// Mock实现：返回空建议
	return []model.SearchSuggestion{}, nil
}

func (m *mockCacheService) SetSuggestions(ctx context.Context, query string, searchType string, suggestions []model.SearchSuggestion, ttl int) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockCacheService) ClearCache(ctx context.Context, pattern string) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockCacheService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	// Mock实现：返回空统计
	return map[string]interface{}{
		"hits":   0,
		"misses": 0,
		"size":   0,
	}, nil
}

// ============ Mock Event Service ============

// mockEventService Mock事件服务实现
type mockEventService struct{}

// NewMockEventService 创建Mock事件服务
func NewMockEventService() EventService {
	return &mockEventService{}
}

func (m *mockEventService) PublishSearchEvent(ctx context.Context, event *SearchEvent) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockEventService) PublishIndexEvent(ctx context.Context, event *IndexEvent) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockEventService) HandleContentEvent(ctx context.Context, event *ContentEvent) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockEventService) HandleUserEvent(ctx context.Context, event *UserEvent) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockEventService) HandleMessageEvent(ctx context.Context, event *MessageEvent) error {
	// Mock实现：什么都不做
	return nil
}

func (m *mockEventService) HandleGroupEvent(ctx context.Context, event *GroupEvent) error {
	// Mock实现：什么都不做
	return nil
}

// ============ 错误定义 ============

var (
	// ErrCacheMiss 缓存未命中错误
	ErrCacheMiss = &ServiceError{Code: "CACHE_MISS", Message: "cache miss"}
)
