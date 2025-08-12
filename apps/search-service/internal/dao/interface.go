package dao

import (
	"context"

	"goim-social/apps/search-service/internal/model"
)

// SearchDAO 搜索数据访问接口
type SearchDAO interface {
	// ============ 索引管理 ============
	
	// CreateIndex 创建索引
	CreateIndex(ctx context.Context, indexName string, mapping map[string]interface{}, settings map[string]interface{}) error
	
	// DeleteIndex 删除索引
	DeleteIndex(ctx context.Context, indexName string) error
	
	// IndexExists 检查索引是否存在
	IndexExists(ctx context.Context, indexName string) (bool, error)
	
	// GetIndexMapping 获取索引映射
	GetIndexMapping(ctx context.Context, indexName string) (map[string]interface{}, error)
	
	// UpdateIndexSettings 更新索引设置
	UpdateIndexSettings(ctx context.Context, indexName string, settings map[string]interface{}) error

	// ============ 文档操作 ============
	
	// IndexDocument 索引文档
	IndexDocument(ctx context.Context, indexName, docID string, document interface{}) error
	
	// BulkIndexDocuments 批量索引文档
	BulkIndexDocuments(ctx context.Context, indexName string, documents []BulkDocument) error
	
	// UpdateDocument 更新文档
	UpdateDocument(ctx context.Context, indexName, docID string, document interface{}) error
	
	// DeleteDocument 删除文档
	DeleteDocument(ctx context.Context, indexName, docID string) error
	
	// GetDocument 获取文档
	GetDocument(ctx context.Context, indexName, docID string) (map[string]interface{}, error)

	// ============ 搜索操作 ============
	
	// Search 通用搜索
	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
	
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
	GetSuggestions(ctx context.Context, query string, searchType string, limit int) ([]model.SearchSuggestion, error)
	
	// GetAutoComplete 获取自动完成建议
	GetAutoComplete(ctx context.Context, req *model.AutoCompleteRequest) (*model.AutoCompleteResponse, error)

	// ============ 聚合查询 ============
	
	// GetAggregations 获取聚合数据
	GetAggregations(ctx context.Context, indexName string, aggs map[string]interface{}) (map[string]interface{}, error)
	
	// GetSearchStats 获取搜索统计
	GetSearchStats(ctx context.Context, timeRange string) (*model.SearchStats, error)

	// ============ 健康检查 ============
	
	// Ping 检查ElasticSearch连接
	Ping(ctx context.Context) error
	
	// GetClusterHealth 获取集群健康状态
	GetClusterHealth(ctx context.Context) (map[string]interface{}, error)
	
	// GetClusterStats 获取集群统计信息
	GetClusterStats(ctx context.Context) (map[string]interface{}, error)
}

// HistoryDAO 搜索历史数据访问接口
type HistoryDAO interface {
	// ============ 搜索历史管理 ============
	
	// CreateSearchHistory 创建搜索历史
	CreateSearchHistory(ctx context.Context, history *model.SearchHistory) error
	
	// GetUserSearchHistory 获取用户搜索历史
	GetUserSearchHistory(ctx context.Context, userID int64, searchType string, limit int) ([]*model.SearchHistory, error)
	
	// DeleteUserSearchHistory 删除用户搜索历史
	DeleteUserSearchHistory(ctx context.Context, userID int64, historyID int64) error
	
	// ClearUserSearchHistory 清空用户搜索历史
	ClearUserSearchHistory(ctx context.Context, userID int64, searchType string) error

	// ============ 热门搜索管理 ============
	
	// UpdateHotSearch 更新热门搜索
	UpdateHotSearch(ctx context.Context, query string, searchType string) error
	
	// GetHotSearches 获取热门搜索
	GetHotSearches(ctx context.Context, searchType string, limit int) ([]*model.HotSearch, error)
	
	// CleanupOldHotSearches 清理过期热门搜索
	CleanupOldHotSearches(ctx context.Context, days int) error

	// ============ 搜索分析管理 ============
	
	// CreateSearchAnalytics 创建搜索分析记录
	CreateSearchAnalytics(ctx context.Context, analytics *model.SearchAnalytics) error
	
	// GetSearchAnalytics 获取搜索分析数据
	GetSearchAnalytics(ctx context.Context, startTime, endTime string, searchType string) ([]*model.SearchAnalytics, error)
	
	// GetSearchPerformanceStats 获取搜索性能统计
	GetSearchPerformanceStats(ctx context.Context, timeRange string) (map[string]interface{}, error)

	// ============ 用户偏好管理 ============
	
	// CreateOrUpdateUserPreference 创建或更新用户搜索偏好
	CreateOrUpdateUserPreference(ctx context.Context, preference *model.UserSearchPreference) error
	
	// GetUserPreference 获取用户搜索偏好
	GetUserPreference(ctx context.Context, userID int64) (*model.UserSearchPreference, error)
	
	// DeleteUserPreference 删除用户搜索偏好
	DeleteUserPreference(ctx context.Context, userID int64) error

	// ============ 索引配置管理 ============
	
	// CreateSearchIndex 创建搜索索引配置
	CreateSearchIndex(ctx context.Context, index *model.SearchIndex) error
	
	// GetSearchIndex 获取搜索索引配置
	GetSearchIndex(ctx context.Context, indexName string) (*model.SearchIndex, error)
	
	// UpdateSearchIndex 更新搜索索引配置
	UpdateSearchIndex(ctx context.Context, index *model.SearchIndex) error
	
	// ListSearchIndices 列出所有搜索索引配置
	ListSearchIndices(ctx context.Context, isActive bool) ([]*model.SearchIndex, error)

	// ============ 同步状态管理 ============
	
	// CreateSyncStatus 创建同步状态
	CreateSyncStatus(ctx context.Context, status *model.SyncStatus) error
	
	// UpdateSyncStatus 更新同步状态
	UpdateSyncStatus(ctx context.Context, status *model.SyncStatus) error
	
	// GetSyncStatus 获取同步状态
	GetSyncStatus(ctx context.Context, sourceTable, targetIndex string) (*model.SyncStatus, error)
	
	// ListSyncStatuses 列出所有同步状态
	ListSyncStatuses(ctx context.Context, sourceService string) ([]*model.SyncStatus, error)
}

// ============ 辅助结构体 ============

// BulkDocument 批量操作文档
type BulkDocument struct {
	ID       string      `json:"id"`
	Document interface{} `json:"document"`
	Action   string      `json:"action"` // index/update/delete
}

// SearchRequest ElasticSearch搜索请求
type SearchRequest struct {
	Index       string                 `json:"index"`
	Query       map[string]interface{} `json:"query"`
	Sort        []map[string]interface{} `json:"sort,omitempty"`
	From        int                    `json:"from,omitempty"`
	Size        int                    `json:"size,omitempty"`
	Highlight   map[string]interface{} `json:"highlight,omitempty"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	Source      interface{}            `json:"_source,omitempty"`
}

// SearchResponse ElasticSearch搜索响应
type SearchResponse struct {
	Took         int64                    `json:"took"`
	TimedOut     bool                     `json:"timed_out"`
	Hits         SearchHits               `json:"hits"`
	Aggregations map[string]interface{}   `json:"aggregations,omitempty"`
	Suggest      map[string]interface{}   `json:"suggest,omitempty"`
}

// SearchHits 搜索结果
type SearchHits struct {
	Total    SearchTotal `json:"total"`
	MaxScore float64     `json:"max_score"`
	Hits     []SearchHit `json:"hits"`
}

// SearchTotal 搜索总数
type SearchTotal struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

// SearchHit 搜索命中项
type SearchHit struct {
	Index     string                 `json:"_index"`
	Type      string                 `json:"_type"`
	ID        string                 `json:"_id"`
	Score     float64                `json:"_score"`
	Source    map[string]interface{} `json:"_source"`
	Highlight map[string][]string    `json:"highlight,omitempty"`
}

// ============ 错误定义 ============

// 自定义错误类型
type SearchError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *SearchError) Error() string {
	return e.Message
}

// 常见错误
var (
	ErrIndexNotFound    = &SearchError{Code: "INDEX_NOT_FOUND", Message: "index not found"}
	ErrDocumentNotFound = &SearchError{Code: "DOCUMENT_NOT_FOUND", Message: "document not found"}
	ErrInvalidQuery     = &SearchError{Code: "INVALID_QUERY", Message: "invalid query"}
	ErrTimeout          = &SearchError{Code: "TIMEOUT", Message: "operation timeout"}
	ErrConnectionFailed = &SearchError{Code: "CONNECTION_FAILED", Message: "connection failed"}
)
