package service

import (
	"context"

	"goim-social/apps/search-service/model"
)

// SearchService 搜索服务接口
type SearchService interface {
	// ============ 搜索功能 ============
	
	// Search 通用搜索
	Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error)
	
	// SearchContent 内容搜索
	SearchContent(ctx context.Context, req *model.SearchRequest) ([]*model.ContentSearchResult, int64, error)
	
	// SearchUsers 用户搜索
	SearchUsers(ctx context.Context, req *model.SearchRequest) ([]*model.UserSearchResult, int64, error)
	
	// SearchMessages 消息搜索
	SearchMessages(ctx context.Context, req *model.SearchRequest) ([]*model.MessageSearchResult, int64, error)
	
	// SearchGroups 群组搜索
	SearchGroups(ctx context.Context, req *model.SearchRequest) ([]*model.GroupSearchResult, int64, error)
	
	// MultiSearch 多类型搜索
	MultiSearch(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error)

	// ============ 搜索建议 ============
	
	// GetSuggestions 获取搜索建议
	GetSuggestions(ctx context.Context, query string, searchType string, limit int, userID int64) ([]model.SearchSuggestion, error)
	
	// GetAutoComplete 获取自动完成建议
	GetAutoComplete(ctx context.Context, req *model.AutoCompleteRequest) (*model.AutoCompleteResponse, error)
	
	// GetHotSearches 获取热门搜索
	GetHotSearches(ctx context.Context, searchType string, limit int) ([]*model.HotSearch, error)

	// ============ 搜索历史 ============
	
	// GetUserSearchHistory 获取用户搜索历史
	GetUserSearchHistory(ctx context.Context, userID int64, searchType string, limit int) ([]*model.SearchHistory, error)
	
	// ClearUserSearchHistory 清空用户搜索历史
	ClearUserSearchHistory(ctx context.Context, userID int64, searchType string) error
	
	// DeleteSearchHistory 删除特定搜索历史
	DeleteSearchHistory(ctx context.Context, userID int64, historyID int64) error

	// ============ 用户偏好 ============
	
	// GetUserPreference 获取用户搜索偏好
	GetUserPreference(ctx context.Context, userID int64) (*model.UserSearchPreference, error)
	
	// UpdateUserPreference 更新用户搜索偏好
	UpdateUserPreference(ctx context.Context, preference *model.UserSearchPreference) error

	// ============ 搜索统计 ============
	
	// GetSearchStats 获取搜索统计
	GetSearchStats(ctx context.Context, timeRange string) (*model.SearchStats, error)
	
	// GetSearchAnalytics 获取搜索分析数据
	GetSearchAnalytics(ctx context.Context, startTime, endTime string, searchType string) ([]*model.SearchAnalytics, error)
}

// IndexService 索引管理服务接口
type IndexService interface {
	// ============ 索引管理 ============
	
	// CreateIndex 创建索引
	CreateIndex(ctx context.Context, indexName string, indexType string) error
	
	// DeleteIndex 删除索引
	DeleteIndex(ctx context.Context, indexName string) error
	
	// ReindexAll 重建所有索引
	ReindexAll(ctx context.Context) error
	
	// ReindexByType 按类型重建索引
	ReindexByType(ctx context.Context, indexType string) error

	// ============ 文档管理 ============
	
	// IndexDocument 索引单个文档
	IndexDocument(ctx context.Context, indexType string, docID string, document interface{}) error
	
	// BulkIndexDocuments 批量索引文档
	BulkIndexDocuments(ctx context.Context, indexType string, documents []IndexDocument) error
	
	// UpdateDocument 更新文档
	UpdateDocument(ctx context.Context, indexType string, docID string, document interface{}) error
	
	// DeleteDocument 删除文档
	DeleteDocument(ctx context.Context, indexType string, docID string) error

	// ============ 数据同步 ============
	
	// SyncFromDatabase 从数据库同步数据
	SyncFromDatabase(ctx context.Context, sourceService string, sourceTable string, targetIndex string) error
	
	// GetSyncStatus 获取同步状态
	GetSyncStatus(ctx context.Context, sourceTable string, targetIndex string) (*model.SyncStatus, error)
	
	// ListSyncStatuses 列出同步状态
	ListSyncStatuses(ctx context.Context, sourceService string) ([]*model.SyncStatus, error)

	// ============ 健康检查 ============
	
	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
	
	// GetClusterInfo 获取集群信息
	GetClusterInfo(ctx context.Context) (map[string]interface{}, error)
}

// CacheService 缓存服务接口
type CacheService interface {
	// ============ 搜索结果缓存 ============
	
	// GetSearchResult 获取缓存的搜索结果
	GetSearchResult(ctx context.Context, cacheKey string) (interface{}, error)
	
	// SetSearchResult 设置搜索结果缓存
	SetSearchResult(ctx context.Context, cacheKey string, result interface{}, ttl int) error
	
	// DeleteSearchResult 删除搜索结果缓存
	DeleteSearchResult(ctx context.Context, cacheKey string) error

	// ============ 用户数据缓存 ============
	
	// GetUserSearchHistory 获取用户搜索历史缓存
	GetUserSearchHistory(ctx context.Context, userID int64) ([]*model.SearchHistory, error)
	
	// SetUserSearchHistory 设置用户搜索历史缓存
	SetUserSearchHistory(ctx context.Context, userID int64, history []*model.SearchHistory, ttl int) error
	
	// GetUserPreference 获取用户偏好缓存
	GetUserPreference(ctx context.Context, userID int64) (*model.UserSearchPreference, error)
	
	// SetUserPreference 设置用户偏好缓存
	SetUserPreference(ctx context.Context, userID int64, preference *model.UserSearchPreference, ttl int) error

	// ============ 热门数据缓存 ============
	
	// GetHotSearches 获取热门搜索缓存
	GetHotSearches(ctx context.Context, searchType string) ([]*model.HotSearch, error)
	
	// SetHotSearches 设置热门搜索缓存
	SetHotSearches(ctx context.Context, searchType string, hotSearches []*model.HotSearch, ttl int) error
	
	// GetSuggestions 获取搜索建议缓存
	GetSuggestions(ctx context.Context, query string, searchType string) ([]model.SearchSuggestion, error)
	
	// SetSuggestions 设置搜索建议缓存
	SetSuggestions(ctx context.Context, query string, searchType string, suggestions []model.SearchSuggestion, ttl int) error

	// ============ 缓存管理 ============
	
	// ClearCache 清空缓存
	ClearCache(ctx context.Context, pattern string) error
	
	// GetCacheStats 获取缓存统计
	GetCacheStats(ctx context.Context) (map[string]interface{}, error)
}

// EventService 事件服务接口
type EventService interface {
	// ============ 搜索事件 ============
	
	// PublishSearchEvent 发布搜索事件
	PublishSearchEvent(ctx context.Context, event *SearchEvent) error
	
	// PublishIndexEvent 发布索引事件
	PublishIndexEvent(ctx context.Context, event *IndexEvent) error

	// ============ 事件处理 ============
	
	// HandleContentEvent 处理内容事件
	HandleContentEvent(ctx context.Context, event *ContentEvent) error
	
	// HandleUserEvent 处理用户事件
	HandleUserEvent(ctx context.Context, event *UserEvent) error
	
	// HandleMessageEvent 处理消息事件
	HandleMessageEvent(ctx context.Context, event *MessageEvent) error
	
	// HandleGroupEvent 处理群组事件
	HandleGroupEvent(ctx context.Context, event *GroupEvent) error
}

// ============ 事件结构体 ============

// SearchEvent 搜索事件
type SearchEvent struct {
	UserID       int64                  `json:"user_id"`
	Query        string                 `json:"query"`
	SearchType   string                 `json:"search_type"`
	ResultCount  int                    `json:"result_count"`
	Duration     int64                  `json:"duration_ms"`
	Filters      map[string]string      `json:"filters,omitempty"`
	Timestamp    int64                  `json:"timestamp"`
	RequestID    string                 `json:"request_id,omitempty"`
	ClientIP     string                 `json:"client_ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
}

// IndexEvent 索引事件
type IndexEvent struct {
	Action      string                 `json:"action"` // create/update/delete
	IndexName   string                 `json:"index_name"`
	DocumentID  string                 `json:"document_id"`
	DocumentType string                `json:"document_type"`
	Document    map[string]interface{} `json:"document,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
	Source      string                 `json:"source"` // 事件来源服务
}

// ContentEvent 内容事件
type ContentEvent struct {
	Action    string                 `json:"action"` // create/update/delete
	ContentID int64                  `json:"content_id"`
	Content   map[string]interface{} `json:"content,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// UserEvent 用户事件
type UserEvent struct {
	Action    string                 `json:"action"` // create/update/delete
	UserID    int64                  `json:"user_id"`
	User      map[string]interface{} `json:"user,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// MessageEvent 消息事件
type MessageEvent struct {
	Action    string                 `json:"action"` // create/update/delete
	MessageID int64                  `json:"message_id"`
	Message   map[string]interface{} `json:"message,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// GroupEvent 群组事件
type GroupEvent struct {
	Action    string                 `json:"action"` // create/update/delete
	GroupID   int64                  `json:"group_id"`
	Group     map[string]interface{} `json:"group,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// IndexDocument 索引文档
type IndexDocument struct {
	ID       string      `json:"id"`
	Document interface{} `json:"document"`
}

// ============ 服务配置 ============

// ServiceConfig 服务配置
type ServiceConfig struct {
	// 搜索配置
	DefaultPageSize   int                    `json:"default_page_size"`
	MaxPageSize       int                    `json:"max_page_size"`
	SearchTimeout     int                    `json:"search_timeout_ms"`
	HighlightPreTag   string                 `json:"highlight_pre_tag"`
	HighlightPostTag  string                 `json:"highlight_post_tag"`
	
	// 缓存配置
	CacheEnabled      bool                   `json:"cache_enabled"`
	CacheTTL          map[string]int         `json:"cache_ttl"`
	
	// 索引配置
	IndexSettings     map[string]interface{} `json:"index_settings"`
	
	// 权重配置
	FieldWeights      map[string]float64     `json:"field_weights"`
	
	// 事件配置
	EventEnabled      bool                   `json:"event_enabled"`
	EventTopics       map[string]string      `json:"event_topics"`
}

// ============ 错误定义 ============

// ServiceError 服务错误
type ServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ServiceError) Error() string {
	return e.Message
}

// 常见服务错误
var (
	ErrInvalidRequest     = &ServiceError{Code: "INVALID_REQUEST", Message: "invalid request"}
	ErrSearchFailed       = &ServiceError{Code: "SEARCH_FAILED", Message: "search operation failed"}
	ErrIndexFailed        = &ServiceError{Code: "INDEX_FAILED", Message: "index operation failed"}
	ErrCacheFailed        = &ServiceError{Code: "CACHE_FAILED", Message: "cache operation failed"}
	ErrPermissionDenied   = &ServiceError{Code: "PERMISSION_DENIED", Message: "permission denied"}
	ErrResourceNotFound   = &ServiceError{Code: "RESOURCE_NOT_FOUND", Message: "resource not found"}
	ErrServiceUnavailable = &ServiceError{Code: "SERVICE_UNAVAILABLE", Message: "service unavailable"}
)
