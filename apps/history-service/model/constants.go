package model

// 行为类型常量
const (
	ActionTypeView     = "view"     // 浏览
	ActionTypeLike     = "like"     // 点赞
	ActionTypeFavorite = "favorite" // 收藏
	ActionTypeShare    = "share"    // 分享
	ActionTypeComment  = "comment"  // 评论
	ActionTypeFollow   = "follow"   // 关注
	ActionTypeLogin    = "login"    // 登录
	ActionTypeSearch   = "search"   // 搜索
	ActionTypeDownload = "download" // 下载
	ActionTypePurchase = "purchase" // 购买
)

// 对象类型常量
const (
	ObjectTypePost    = "post"    // 帖子
	ObjectTypeArticle = "article" // 文章
	ObjectTypeVideo   = "video"   // 视频
	ObjectTypeUser    = "user"    // 用户
	ObjectTypeProduct = "product" // 商品
	ObjectTypeGroup   = "group"   // 群组
)

// 分页常量
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// 时间范围常量
const (
	TimeRangeToday = "today"
	TimeRangeWeek  = "week"
	TimeRangeMonth = "month"
	TimeRangeAll   = "all"
)

// 缓存相关常量
const (
	HistoryCachePrefix      = "history:"
	UserStatsCachePrefix    = "user_stats:"
	ObjectStatsCachePrefix  = "object_stats:"
	HotObjectsCachePrefix   = "hot_objects:"
	ActivityStatsCachePrefix = "activity_stats:"
	CacheExpireTime         = 3600 // 缓存过期时间（秒）
)

// 事件类型常量
const (
	EventHistoryCreated = "history.created"
	EventUserActive     = "user.active"
	EventObjectHot      = "object.hot"
)

// 统计相关常量
const (
	DefaultHotObjectLimit = 50  // 默认热门对象数量限制
	MaxHotObjectLimit     = 200 // 最大热门对象数量限制
	HotScoreDecayDays     = 7   // 热度分数衰减天数
)

// 数据保留策略常量
const (
	DefaultRetentionDays = 365 // 默认数据保留天数
	MaxRetentionDays     = 730 // 最大数据保留天数
)

// 批量操作限制
const (
	MaxBatchCreateSize = 1000 // 最大批量创建数量
	MaxBatchDeleteSize = 1000 // 最大批量删除数量
)

// 设备类型常量
const (
	DeviceTypeWeb     = "web"
	DeviceTypeMobile  = "mobile"
	DeviceTypeTablet  = "tablet"
	DeviceTypeDesktop = "desktop"
	DeviceTypeUnknown = "unknown"
)

// 地理位置相关常量
const (
	LocationUnknown = "unknown"
)

// 热度计算权重
const (
	ViewWeight     = 1.0  // 浏览权重
	LikeWeight     = 2.0  // 点赞权重
	FavoriteWeight = 3.0  // 收藏权重
	ShareWeight    = 4.0  // 分享权重
	CommentWeight  = 5.0  // 评论权重
)

// 活跃度计算相关
const (
	MinActivityScore = 0.0   // 最小活跃度分数
	MaxActivityScore = 100.0 // 最大活跃度分数
)
