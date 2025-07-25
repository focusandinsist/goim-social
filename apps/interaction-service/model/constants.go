package model

// 默认配置
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// 互动类型
const (
	InteractionTypeLike     = "like"     // 点赞
	InteractionTypeFavorite = "favorite" // 收藏
	InteractionTypeShare    = "share"    // 分享
	InteractionTypeRepost   = "repost"   // 转发
)

// 对象类型
const (
	ObjectTypePost    = "post"    // 帖子
	ObjectTypeComment = "comment" // 评论
	ObjectTypeUser    = "user"    // 用户
)

// 排序字段
const (
	SortByCreatedAt = "created_at"
	SortByUpdatedAt = "updated_at"
)

// 排序方向
const (
	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// Redis缓存键前缀
const (
	CacheKeyInteractionStats = "interaction:stats"     // 互动统计缓存
	CacheKeyUserInteraction  = "interaction:user"      // 用户互动缓存
	CacheKeyObjectHot        = "interaction:hot"       // 热门对象缓存
	CacheKeyInteractionList  = "interaction:list"      // 互动列表缓存
)

// 缓存过期时间（秒）
const (
	CacheExpireStats      = 300  // 统计缓存5分钟
	CacheExpireUserAction = 3600 // 用户行为缓存1小时
	CacheExpireHotList    = 600  // 热门列表缓存10分钟
)

// 批量操作限制
const (
	MaxBatchSize = 100 // 批量操作最大数量
)

// 有效的互动类型列表
var ValidInteractionTypes = []string{
	InteractionTypeLike,
	InteractionTypeFavorite,
	InteractionTypeShare,
	InteractionTypeRepost,
}

// 有效的对象类型列表
var ValidObjectTypes = []string{
	ObjectTypePost,
	ObjectTypeComment,
	ObjectTypeUser,
}
