package model

import (
	"time"
)

// ============ 搜索历史模型 ============

// SearchHistory 用户搜索历史
type SearchHistory struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID           int64     `json:"user_id" gorm:"not null;index"`
	Query            string    `json:"query" gorm:"type:text;not null"`
	SearchType       string    `json:"search_type" gorm:"type:varchar(50);not null;index"`
	ResultCount      int       `json:"result_count" gorm:"default:0"`
	ClickedResultID  string    `json:"clicked_result_id" gorm:"type:varchar(100)"`
	ClickedResultType string   `json:"clicked_result_type" gorm:"type:varchar(50)"`
	SearchTime       time.Time `json:"search_time" gorm:"autoCreateTime;index"`
}

// TableName 表名
func (SearchHistory) TableName() string {
	return "search_history"
}

// ============ 热门搜索模型 ============

// HotSearch 热门搜索词
type HotSearch struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Query          string    `json:"query" gorm:"type:text;not null"`
	SearchType     string    `json:"search_type" gorm:"type:varchar(50);not null;index"`
	SearchCount    int       `json:"search_count" gorm:"default:1;index"`
	LastSearchTime time.Time `json:"last_search_time" gorm:"autoUpdateTime;index"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (HotSearch) TableName() string {
	return "hot_searches"
}

// ============ 搜索索引配置模型 ============

// SearchIndex 搜索索引配置
type SearchIndex struct {
	ID             int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	IndexName      string                 `json:"index_name" gorm:"type:varchar(100);not null;uniqueIndex"`
	IndexType      string                 `json:"index_type" gorm:"type:varchar(50);not null;index"`
	MappingConfig  map[string]interface{} `json:"mapping_config" gorm:"type:json"`
	SettingsConfig map[string]interface{} `json:"settings_config" gorm:"type:json"`
	IsActive       bool                   `json:"is_active" gorm:"default:true;index"`
	CreatedAt      time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (SearchIndex) TableName() string {
	return "search_indices"
}

// ============ 数据同步状态模型 ============

// SyncStatus 数据同步状态
type SyncStatus struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceTable  string    `json:"source_table" gorm:"type:varchar(100);not null"`
	SourceService string   `json:"source_service" gorm:"type:varchar(100);not null;index"`
	TargetIndex  string    `json:"target_index" gorm:"type:varchar(100);not null"`
	LastSyncID   int64     `json:"last_sync_id" gorm:"default:0"`
	LastSyncTime time.Time `json:"last_sync_time" gorm:"autoUpdateTime;index"`
	SyncStatus   string    `json:"sync_status" gorm:"type:varchar(20);default:'pending';index"`
	ErrorMessage string    `json:"error_message" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (SyncStatus) TableName() string {
	return "sync_status"
}

// 同步状态常量
const (
	SyncStatusPending   = "pending"
	SyncStatusRunning   = "running"
	SyncStatusCompleted = "completed"
	SyncStatusFailed    = "failed"
)

// ============ 搜索分析模型 ============

// SearchAnalytics 搜索性能分析
type SearchAnalytics struct {
	ID                  int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	QueryHash           string    `json:"query_hash" gorm:"type:varchar(64);not null;index"`
	Query               string    `json:"query" gorm:"type:text;not null"`
	SearchType          string    `json:"search_type" gorm:"type:varchar(50);not null;index"`
	UserID              int64     `json:"user_id" gorm:"index"`
	ExecutionTimeMs     int       `json:"execution_time_ms" gorm:"not null;index"`
	ResultCount         int       `json:"result_count" gorm:"default:0"`
	HitCache            bool      `json:"hit_cache" gorm:"default:false"`
	ElasticsearchTimeMs int       `json:"elasticsearch_time_ms" gorm:"default:0"`
	TotalHits           int64     `json:"total_hits" gorm:"default:0"`
	SearchDate          time.Time `json:"search_date" gorm:"type:date;not null;index"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}

// TableName 表名
func (SearchAnalytics) TableName() string {
	return "search_analytics"
}

// ============ 搜索建议模型 ============

// SearchSuggestionHistory 搜索建议历史
type SearchSuggestionHistory struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int64     `json:"user_id" gorm:"not null;index"`
	Query      string    `json:"query" gorm:"type:text;not null"`
	Suggestion string    `json:"suggestion" gorm:"type:text;not null"`
	Accepted   bool      `json:"accepted" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 表名
func (SearchSuggestionHistory) TableName() string {
	return "search_suggestion_history"
}

// ============ 用户搜索偏好模型 ============

// UserSearchPreference 用户搜索偏好
type UserSearchPreference struct {
	ID                int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID            int64                  `json:"user_id" gorm:"not null;uniqueIndex"`
	PreferredTypes    []string               `json:"preferred_types" gorm:"type:json"`
	SearchFilters     map[string]interface{} `json:"search_filters" gorm:"type:json"`
	SortPreferences   map[string]string      `json:"sort_preferences" gorm:"type:json"`
	LanguagePreference string                `json:"language_preference" gorm:"type:varchar(10);default:'zh'"`
	ResultsPerPage    int                    `json:"results_per_page" gorm:"default:20"`
	EnableSuggestions bool                   `json:"enable_suggestions" gorm:"default:true"`
	EnableHistory     bool                   `json:"enable_history" gorm:"default:true"`
	CreatedAt         time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (UserSearchPreference) TableName() string {
	return "user_search_preferences"
}

// ============ 搜索标签模型 ============

// SearchTag 搜索标签
type SearchTag struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	Category    string    `json:"category" gorm:"type:varchar(50);index"`
	Description string    `json:"description" gorm:"type:text"`
	Color       string    `json:"color" gorm:"type:varchar(7)"`
	UsageCount  int64     `json:"usage_count" gorm:"default:0;index"`
	IsActive    bool      `json:"is_active" gorm:"default:true;index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (SearchTag) TableName() string {
	return "search_tags"
}

// ============ 搜索过滤器模型 ============

// SearchFilter 搜索过滤器
type SearchFilter struct {
	ID          int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string                 `json:"name" gorm:"type:varchar(100);not null"`
	Type        string                 `json:"type" gorm:"type:varchar(50);not null"` // range/term/terms/exists
	Field       string                 `json:"field" gorm:"type:varchar(100);not null"`
	Options     map[string]interface{} `json:"options" gorm:"type:json"`
	DefaultValue interface{}           `json:"default_value" gorm:"type:json"`
	IsRequired  bool                   `json:"is_required" gorm:"default:false"`
	IsActive    bool                   `json:"is_active" gorm:"default:true;index"`
	SortOrder   int                    `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (SearchFilter) TableName() string {
	return "search_filters"
}

// ============ 搜索模板模型 ============

// SearchTemplate 搜索模板
type SearchTemplate struct {
	ID          int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string                 `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	Description string                 `json:"description" gorm:"type:text"`
	SearchType  string                 `json:"search_type" gorm:"type:varchar(50);not null;index"`
	Template    map[string]interface{} `json:"template" gorm:"type:json;not null"`
	Parameters  map[string]interface{} `json:"parameters" gorm:"type:json"`
	IsDefault   bool                   `json:"is_default" gorm:"default:false"`
	IsActive    bool                   `json:"is_active" gorm:"default:true;index"`
	UsageCount  int64                  `json:"usage_count" gorm:"default:0"`
	CreatedBy   int64                  `json:"created_by" gorm:"index"`
	CreatedAt   time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (SearchTemplate) TableName() string {
	return "search_templates"
}
