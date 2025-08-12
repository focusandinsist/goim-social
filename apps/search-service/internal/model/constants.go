package model

// ============ 搜索类型常量 ============

const (
	// 搜索类型
	SearchTypeContent = "content" // 内容搜索
	SearchTypeUser    = "user"    // 用户搜索
	SearchTypeMessage = "message" // 消息搜索
	SearchTypeGroup   = "group"   // 群组搜索
	SearchTypeAll     = "all"     // 全局搜索
)

// ============ 索引名称常量 ============

const (
	// ElasticSearch索引名称
	IndexContent = "goim-content"
	IndexUser    = "goim-user"
	IndexMessage = "goim-message"
	IndexGroup   = "goim-group"
)

// ============ 排序字段常量 ============

const (
	// 排序字段
	SortByRelevance  = "relevance"   // 相关性
	SortByTime       = "time"        // 时间
	SortByPopularity = "popularity"  // 热度
	SortByViews      = "views"       // 浏览量
	SortByLikes      = "likes"       // 点赞数
	SortByComments   = "comments"    // 评论数
	SortByShares     = "shares"      // 分享数
	SortByMembers    = "members"     // 成员数
	SortByCreated    = "created"     // 创建时间
	SortByUpdated    = "updated"     // 更新时间
)

// ============ 排序方向常量 ============

const (
	// 排序方向
	SortOrderAsc  = "asc"  // 升序
	SortOrderDesc = "desc" // 降序
)

// ============ 消息类型常量 ============

const (
	// 消息类型
	MessageTypeText  = "text"  // 文本消息
	MessageTypeImage = "image" // 图片消息
	MessageTypeAudio = "audio" // 语音消息
	MessageTypeVideo = "video" // 视频消息
	MessageTypeFile  = "file"  // 文件消息
	MessageTypeLink  = "link"  // 链接消息
)

// ============ 用户状态常量 ============

const (
	// 用户状态
	UserStatusActive   = "active"   // 活跃
	UserStatusInactive = "inactive" // 不活跃
	UserStatusBanned   = "banned"   // 被禁用
	UserStatusDeleted  = "deleted"  // 已删除
)

// ============ 内容状态常量 ============

const (
	// 内容状态
	ContentStatusDraft     = "draft"     // 草稿
	ContentStatusPublished = "published" // 已发布
	ContentStatusArchived  = "archived"  // 已归档
	ContentStatusDeleted   = "deleted"   // 已删除
	ContentStatusReviewing = "reviewing" // 审核中
	ContentStatusRejected  = "rejected"  // 已拒绝
)

// ============ 群组状态常量 ============

const (
	// 群组状态
	GroupStatusActive   = "active"   // 活跃
	GroupStatusInactive = "inactive" // 不活跃
	GroupStatusArchived = "archived" // 已归档
	GroupStatusDeleted  = "deleted"  // 已删除
)

// ============ 搜索建议类型常量 ============

const (
	// 搜索建议类型
	SuggestionTypeQuery      = "query"      // 查询建议
	SuggestionTypeCompletion = "completion" // 自动完成
)

// ============ 搜索建议来源常量 ============

const (
	// 搜索建议来源
	SuggestionSourceHistory = "history" // 历史记录
	SuggestionSourcePopular = "popular" // 热门搜索
	SuggestionSourceAuto    = "auto"    // 自动生成
)

// ============ 缓存键前缀常量 ============

const (
	// Redis缓存键前缀
	CacheKeySearchResult     = "search:result:"     // 搜索结果缓存
	CacheKeyHotQueries       = "search:hot:"        // 热门查询缓存
	CacheKeyUserHistory      = "search:history:"    // 用户搜索历史缓存
	CacheKeySuggestions      = "search:suggest:"    // 搜索建议缓存
	CacheKeyUserPreference   = "search:pref:"       // 用户偏好缓存
	CacheKeySearchStats      = "search:stats:"      // 搜索统计缓存
)

// ============ 默认配置常量 ============

const (
	// 默认分页配置
	DefaultPageSize = 20
	MaxPageSize     = 100
	MinPageSize     = 1

	// 默认高亮标签
	DefaultHighlightPreTag  = "<mark>"
	DefaultHighlightPostTag = "</mark>"

	// 默认搜索权重
	DefaultTitleWeight   = 3.0
	DefaultContentWeight = 1.0
	DefaultTagWeight     = 2.0
	DefaultNameWeight    = 2.5

	// 默认缓存TTL (秒)
	DefaultSearchResultTTL = 300   // 5分钟
	DefaultHotQueriesTTL   = 3600  // 1小时
	DefaultUserHistoryTTL  = 86400 // 24小时

	// 默认建议数量
	DefaultSuggestionLimit = 10
	MaxSuggestionLimit     = 50

	// 默认统计时间范围
	DefaultStatsTimeRange = 24 // 24小时
)

// ============ ElasticSearch配置常量 ============

const (
	// ElasticSearch默认配置
	DefaultESShards    = 1
	DefaultESReplicas  = 0
	DefaultESRefresh   = "1s"
	DefaultESTimeout   = "30s"
	DefaultESMaxRetries = 3

	// ElasticSearch字段类型
	ESFieldTypeText    = "text"
	ESFieldTypeKeyword = "keyword"
	ESFieldTypeDate    = "date"
	ESFieldTypeLong    = "long"
	ESFieldTypeInteger = "integer"
	ESFieldTypeFloat   = "float"
	ESFieldTypeBoolean = "boolean"
	ESFieldTypeObject  = "object"
	ESFieldTypeNested  = "nested"

	// ElasticSearch分析器
	ESAnalyzerStandard = "standard"
	ESAnalyzerKeyword  = "keyword"
	ESAnalyzerIKSmart  = "ik_smart"
	ESAnalyzerIKMaxWord = "ik_max_word"
)

// ============ Kafka主题常量 ============

const (
	// Kafka主题名称
	TopicContentIndex = "content-index-events"
	TopicUserIndex    = "user-index-events"
	TopicMessageIndex = "message-index-events"
	TopicGroupIndex   = "group-index-events"
	TopicSearchEvents = "search-events"
)

// ============ 错误消息常量 ============

const (
	// 错误消息
	ErrInvalidSearchType    = "invalid search type"
	ErrInvalidPageSize      = "invalid page size"
	ErrInvalidSortField     = "invalid sort field"
	ErrQueryTooShort        = "query too short"
	ErrQueryTooLong         = "query too long"
	ErrIndexNotFound        = "index not found"
	ErrDocumentNotFound     = "document not found"
	ErrElasticsearchTimeout = "elasticsearch timeout"
	ErrCacheTimeout         = "cache timeout"
	ErrInvalidUserID        = "invalid user id"
	ErrPermissionDenied     = "permission denied"
)

// ============ 日志消息常量 ============

const (
	// 日志消息
	LogSearchRequest     = "search request received"
	LogSearchResponse    = "search response sent"
	LogIndexDocument     = "document indexed"
	LogDeleteDocument    = "document deleted"
	LogUpdateDocument    = "document updated"
	LogCacheHit          = "cache hit"
	LogCacheMiss         = "cache miss"
	LogElasticsearchCall = "elasticsearch call"
	LogDatabaseCall      = "database call"
	LogKafkaMessage      = "kafka message processed"
)

// ============ 权重配置映射 ============

var (
	// 默认字段权重配置
	DefaultFieldWeights = map[string]float64{
		"title":       DefaultTitleWeight,
		"content":     DefaultContentWeight,
		"tags":        DefaultTagWeight,
		"username":    DefaultNameWeight,
		"nickname":    DefaultNameWeight,
		"group_name":  DefaultNameWeight,
		"description": DefaultContentWeight,
	}

	// 搜索类型到索引的映射
	SearchTypeToIndex = map[string]string{
		SearchTypeContent: IndexContent,
		SearchTypeUser:    IndexUser,
		SearchTypeMessage: IndexMessage,
		SearchTypeGroup:   IndexGroup,
	}

	// 有效的搜索类型
	ValidSearchTypes = []string{
		SearchTypeContent,
		SearchTypeUser,
		SearchTypeMessage,
		SearchTypeGroup,
		SearchTypeAll,
	}

	// 有效的排序字段
	ValidSortFields = []string{
		SortByRelevance,
		SortByTime,
		SortByPopularity,
		SortByViews,
		SortByLikes,
		SortByComments,
		SortByShares,
		SortByMembers,
		SortByCreated,
		SortByUpdated,
	}

	// 有效的排序方向
	ValidSortOrders = []string{
		SortOrderAsc,
		SortOrderDesc,
	}
)

// ============ 辅助函数 ============

// IsValidSearchType 检查搜索类型是否有效
func IsValidSearchType(searchType string) bool {
	for _, validType := range ValidSearchTypes {
		if searchType == validType {
			return true
		}
	}
	return false
}

// IsValidSortField 检查排序字段是否有效
func IsValidSortField(sortField string) bool {
	for _, validField := range ValidSortFields {
		if sortField == validField {
			return true
		}
	}
	return false
}

// IsValidSortOrder 检查排序方向是否有效
func IsValidSortOrder(sortOrder string) bool {
	for _, validOrder := range ValidSortOrders {
		if sortOrder == validOrder {
			return true
		}
	}
	return false
}

// GetIndexBySearchType 根据搜索类型获取索引名称
func GetIndexBySearchType(searchType string) string {
	if index, exists := SearchTypeToIndex[searchType]; exists {
		return index
	}
	return ""
}

// GetCacheKey 生成缓存键
func GetCacheKey(prefix, key string) string {
	return prefix + key
}
