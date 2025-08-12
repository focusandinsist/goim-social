package model

import (
	"time"
)

// ============ 搜索请求和响应模型 ============

// SearchRequest 通用搜索请求
type SearchRequest struct {
	Query      string            `json:"query" binding:"required"`
	Type       string            `json:"type" binding:"required"` // content/user/message/group
	Filters    map[string]string `json:"filters,omitempty"`
	SortBy     string            `json:"sort_by,omitempty"`
	SortOrder  string            `json:"sort_order,omitempty"` // asc/desc
	Page       int               `json:"page,omitempty"`
	PageSize   int               `json:"page_size,omitempty"`
	Highlight  bool              `json:"highlight,omitempty"`
	UserID     int64             `json:"user_id,omitempty"`
}

// SearchResponse 通用搜索响应
type SearchResponse struct {
	Query       string        `json:"query"`
	Type        string        `json:"type"`
	Total       int64         `json:"total"`
	Page        int           `json:"page"`
	PageSize    int           `json:"page_size"`
	Results     []interface{} `json:"results"`
	Aggregations interface{}   `json:"aggregations,omitempty"`
	Suggestions []string      `json:"suggestions,omitempty"`
	Duration    int64         `json:"duration_ms"`
	FromCache   bool          `json:"from_cache"`
}

// SearchResult 搜索结果项
type SearchResult struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Score       float64                `json:"score"`
	Source      map[string]interface{} `json:"source"`
	Highlight   map[string][]string    `json:"highlight,omitempty"`
}

// ============ 内容搜索模型 ============

// ContentSearchDocument 内容搜索文档
type ContentSearchDocument struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Summary     string    `json:"summary,omitempty"`
	AuthorID    int64     `json:"author_id"`
	AuthorName  string    `json:"author_name"`
	CategoryID  int64     `json:"category_id,omitempty"`
	Category    string    `json:"category,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Status      string    `json:"status"`
	ViewCount   int64     `json:"view_count"`
	LikeCount   int64     `json:"like_count"`
	CommentCount int64    `json:"comment_count"`
	ShareCount  int64     `json:"share_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ContentSearchResult 内容搜索结果
type ContentSearchResult struct {
	ID           int64               `json:"id"`
	Title        string              `json:"title"`
	Content      string              `json:"content"`
	Summary      string              `json:"summary,omitempty"`
	AuthorID     int64               `json:"author_id"`
	AuthorName   string              `json:"author_name"`
	Category     string              `json:"category,omitempty"`
	Tags         []string            `json:"tags,omitempty"`
	ViewCount    int64               `json:"view_count"`
	LikeCount    int64               `json:"like_count"`
	CommentCount int64               `json:"comment_count"`
	CreatedAt    time.Time           `json:"created_at"`
	Score        float64             `json:"score"`
	Highlight    map[string][]string `json:"highlight,omitempty"`
}

// ============ 用户搜索模型 ============

// UserSearchDocument 用户搜索文档
type UserSearchDocument struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Nickname    string    `json:"nickname"`
	Email       string    `json:"email,omitempty"`
	Avatar      string    `json:"avatar,omitempty"`
	Bio         string    `json:"bio,omitempty"`
	Location    string    `json:"location,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Status      string    `json:"status"`
	IsVerified  bool      `json:"is_verified"`
	FriendCount int64     `json:"friend_count"`
	FollowerCount int64   `json:"follower_count"`
	PostCount   int64     `json:"post_count"`
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserSearchResult 用户搜索结果
type UserSearchResult struct {
	ID            int64               `json:"id"`
	Username      string              `json:"username"`
	Nickname      string              `json:"nickname"`
	Avatar        string              `json:"avatar,omitempty"`
	Bio           string              `json:"bio,omitempty"`
	Location      string              `json:"location,omitempty"`
	Tags          []string            `json:"tags,omitempty"`
	IsVerified    bool                `json:"is_verified"`
	FriendCount   int64               `json:"friend_count"`
	FollowerCount int64               `json:"follower_count"`
	IsFriend      bool                `json:"is_friend,omitempty"`
	Score         float64             `json:"score"`
	Highlight     map[string][]string `json:"highlight,omitempty"`
}

// ============ 消息搜索模型 ============

// MessageSearchDocument 消息搜索文档
type MessageSearchDocument struct {
	ID          int64     `json:"id"`
	FromUserID  int64     `json:"from_user_id"`
	FromUsername string   `json:"from_username"`
	ToUserID    int64     `json:"to_user_id,omitempty"`
	ToUsername  string    `json:"to_username,omitempty"`
	GroupID     int64     `json:"group_id,omitempty"`
	GroupName   string    `json:"group_name,omitempty"`
	Content     string    `json:"content"`
	MessageType string    `json:"message_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// MessageSearchResult 消息搜索结果
type MessageSearchResult struct {
	ID           int64               `json:"id"`
	FromUserID   int64               `json:"from_user_id"`
	FromUsername string              `json:"from_username"`
	ToUserID     int64               `json:"to_user_id,omitempty"`
	ToUsername   string              `json:"to_username,omitempty"`
	GroupID      int64               `json:"group_id,omitempty"`
	GroupName    string              `json:"group_name,omitempty"`
	Content      string              `json:"content"`
	MessageType  string              `json:"message_type"`
	CreatedAt    time.Time           `json:"created_at"`
	Score        float64             `json:"score"`
	Highlight    map[string][]string `json:"highlight,omitempty"`
}

// ============ 群组搜索模型 ============

// GroupSearchDocument 群组搜索文档
type GroupSearchDocument struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Avatar      string    `json:"avatar,omitempty"`
	OwnerID     int64     `json:"owner_id"`
	OwnerName   string    `json:"owner_name"`
	MemberCount int64     `json:"member_count"`
	MaxMembers  int64     `json:"max_members"`
	IsPublic    bool      `json:"is_public"`
	Tags        []string  `json:"tags,omitempty"`
	Category    string    `json:"category,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GroupSearchResult 群组搜索结果
type GroupSearchResult struct {
	ID          int64               `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Avatar      string              `json:"avatar,omitempty"`
	OwnerID     int64               `json:"owner_id"`
	OwnerName   string              `json:"owner_name"`
	MemberCount int64               `json:"member_count"`
	IsPublic    bool                `json:"is_public"`
	Tags        []string            `json:"tags,omitempty"`
	Category    string              `json:"category,omitempty"`
	IsMember    bool                `json:"is_member,omitempty"`
	Score       float64             `json:"score"`
	Highlight   map[string][]string `json:"highlight,omitempty"`
}

// ============ 搜索建议模型 ============

// SearchSuggestion 搜索建议
type SearchSuggestion struct {
	Text   string  `json:"text"`
	Score  float64 `json:"score"`
	Type   string  `json:"type"`   // query/completion
	Source string  `json:"source"` // history/popular/auto
}

// AutoCompleteRequest 自动完成请求
type AutoCompleteRequest struct {
	Query    string `json:"query" binding:"required"`
	Type     string `json:"type,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	UserID   int64  `json:"user_id,omitempty"`
}

// AutoCompleteResponse 自动完成响应
type AutoCompleteResponse struct {
	Query       string             `json:"query"`
	Suggestions []SearchSuggestion `json:"suggestions"`
	Duration    int64              `json:"duration_ms"`
}

// ============ 搜索统计模型 ============

// SearchStats 搜索统计
type SearchStats struct {
	TotalSearches    int64             `json:"total_searches"`
	UniqueUsers      int64             `json:"unique_users"`
	AvgResponseTime  float64           `json:"avg_response_time_ms"`
	TopQueries       []QueryStat       `json:"top_queries"`
	SearchesByType   map[string]int64  `json:"searches_by_type"`
	SearchesByHour   map[string]int64  `json:"searches_by_hour"`
	CacheHitRate     float64           `json:"cache_hit_rate"`
}

// QueryStat 查询统计
type QueryStat struct {
	Query       string  `json:"query"`
	Count       int64   `json:"count"`
	AvgResults  float64 `json:"avg_results"`
	ClickRate   float64 `json:"click_rate"`
}

// ============ 搜索配置模型 ============

// SearchConfig 搜索配置
type SearchConfig struct {
	DefaultPageSize   int                    `json:"default_page_size"`
	MaxPageSize       int                    `json:"max_page_size"`
	HighlightPreTag   string                 `json:"highlight_pre_tag"`
	HighlightPostTag  string                 `json:"highlight_post_tag"`
	Weights           map[string]float64     `json:"weights"`
	CacheTTL          map[string]int         `json:"cache_ttl"`
	IndexSettings     map[string]interface{} `json:"index_settings"`
}
