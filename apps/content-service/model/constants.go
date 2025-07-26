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

// 媒体文件类型
const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeAudio = "audio"
	MediaTypeFile  = "file"
)
