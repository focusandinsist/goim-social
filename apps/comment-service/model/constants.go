package model

// 评论状态常量
const (
	CommentStatusPending  = "pending"  // 待审核
	CommentStatusApproved = "approved" // 已通过
	CommentStatusRejected = "rejected" // 已拒绝
	CommentStatusDeleted  = "deleted"  // 已删除
)

// 对象类型常量
const (
	ObjectTypePost    = "post"    // 帖子
	ObjectTypeArticle = "article" // 文章
	ObjectTypeVideo   = "video"   // 视频
	ObjectTypeProduct = "product" // 商品
)

// 排序字段常量
const (
	SortByTime = "time" // 按时间排序
	SortByHot  = "hot"  // 按热度排序
	SortByLike = "like" // 按点赞数排序
)

// 排序方向常量
const (
	SortOrderAsc  = "asc"  // 升序
	SortOrderDesc = "desc" // 降序
)

// 分页常量
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// 评论内容限制
const (
	MaxCommentLength = 2000 // 评论最大长度
	MinCommentLength = 1    // 评论最小长度
)

// 回复层级限制
const (
	MaxReplyDepth    = 3   // 最大回复层级
	DefaultReplyShow = 3   // 默认显示的回复数
	MaxReplyShow     = 10  // 最大显示的回复数
)

// 缓存相关常量
const (
	CommentCachePrefix      = "comment:"
	CommentStatsCachePrefix = "comment_stats:"
	CommentListCachePrefix  = "comment_list:"
	CacheExpireTime         = 3600 // 缓存过期时间（秒）
)

// 事件类型常量
const (
	EventCommentCreated   = "comment.created"
	EventCommentUpdated   = "comment.updated"
	EventCommentDeleted   = "comment.deleted"
	EventCommentApproved  = "comment.approved"
	EventCommentRejected  = "comment.rejected"
	EventCommentPinned    = "comment.pinned"
	EventCommentUnpinned  = "comment.unpinned"
)
