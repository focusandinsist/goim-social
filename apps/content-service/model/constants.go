package model

// 默认配置
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// 内容类型
const (
	ContentTypeText     = "text"     // 纯文本
	ContentTypeImage    = "image"    // 图片
	ContentTypeVideo    = "video"    // 视频
	ContentTypeAudio    = "audio"    // 音频
	ContentTypeMixed    = "mixed"    // 图文混合
	ContentTypeTemplate = "template" // 模板内容
)

// 内容状态
const (
	ContentStatusDraft     = "draft"     // 草稿
	ContentStatusPending   = "pending"   // 待审核
	ContentStatusPublished = "published" // 已发布
	ContentStatusRejected  = "rejected"  // 已拒绝
	ContentStatusDeleted   = "deleted"   // 已删除
)

// 排序字段
const (
	SortByCreatedAt   = "created_at"
	SortByUpdatedAt   = "updated_at"
	SortByViewCount   = "view_count"
	SortByLikeCount   = "like_count"
	SortByPublishedAt = "published_at"
)

// 排序方向
const (
	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// 目标类型 (Target Types) - 用于多态关联
const (
	TargetTypeContent = "content" // 内容/帖子
	TargetTypeComment = "comment" // 评论
	TargetTypeUser    = "user"    // 用户
)

// 评论状态
const (
	CommentStatusPending  = "pending"  // 待审核
	CommentStatusApproved = "approved" // 已通过
	CommentStatusRejected = "rejected" // 已拒绝
	CommentStatusDeleted  = "deleted"  // 已删除
)

// 互动类型
const (
	InteractionTypeLike     = "like"     // 点赞
	InteractionTypeFavorite = "favorite" // 收藏
	InteractionTypeShare    = "share"    // 分享
	InteractionTypeRepost   = "repost"   // 转发
)

// 评论内容限制
const (
	MaxCommentLength = 2000 // 评论最大长度
	MinCommentLength = 1    // 评论最小长度
)

// Redis缓存键前缀
const (
	CacheKeyContentDetail    = "content:detail"    // 内容详情缓存
	CacheKeyContentStats     = "content:stats"     // 内容统计缓存
	CacheKeyContentComments  = "content:comments"  // 内容评论列表缓存
	CacheKeyInteractionStats = "interaction:stats" // 互动统计缓存
	CacheKeyUserInteraction  = "interaction:user"  // 用户互动缓存
	CacheKeyHotContent       = "content:hot"       // 热门内容缓存
	CacheKeyUserContent      = "user:content"      // 用户内容列表缓存
)

// 缓存过期时间（秒）
const (
	CacheExpireContentDetail = 300  // 内容详情缓存5分钟
	CacheExpireStats         = 300  // 统计缓存5分钟
	CacheExpireUserAction    = 3600 // 用户行为缓存1小时
	CacheExpireHotList       = 600  // 热门列表缓存10分钟
	CacheExpireCommentList   = 180  // 评论列表缓存3分钟
)

// 批量操作限制
const (
	MaxBatchSize = 100 // 批量操作最大数量
)

// 媒体文件类型
const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeAudio = "audio"
	MediaTypeFile  = "file"
)
