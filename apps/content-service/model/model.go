package model

import (
	"time"
)

// Content 内容模型
type Content struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	AuthorID     int64      `json:"author_id" gorm:"not null;index"`
	Title        string     `json:"title" gorm:"type:varchar(200);not null"`
	Content      string     `json:"content" gorm:"type:text"`
	Type         string     `json:"type" gorm:"type:varchar(20);not null;index"`
	Status       string     `json:"status" gorm:"type:varchar(20);not null;index;default:'draft'"`
	TemplateData string     `json:"template_data" gorm:"type:text"` // JSON格式的模板数据
	ViewCount    int64      `json:"view_count" gorm:"default:0"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	PublishedAt  *time.Time `json:"published_at" gorm:"index"`

	// 关联数据
	MediaFiles []ContentMediaFile `json:"media_files" gorm:"foreignKey:ContentID"`
	Tags       []ContentTag       `json:"tags" gorm:"many2many:content_tag_relations;"`
	Topics     []ContentTopic     `json:"topics" gorm:"many2many:content_topic_relations;"`
}

// TableName .
func (Content) TableName() string {
	return "contents"
}

// ContentMediaFile 内容媒体文件
type ContentMediaFile struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ContentID int64     `json:"content_id" gorm:"not null;index"`
	URL       string    `json:"url" gorm:"type:varchar(500);not null"`
	Filename  string    `json:"filename" gorm:"type:varchar(255)"`
	Size      int64     `json:"size" gorm:"default:0"`
	MimeType  string    `json:"mime_type" gorm:"type:varchar(100)"`
	Width     int32     `json:"width" gorm:"default:0"`
	Height    int32     `json:"height" gorm:"default:0"`
	Duration  int32     `json:"duration" gorm:"default:0"` // 音视频时长(秒)
	SortOrder int32     `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName .
func (ContentMediaFile) TableName() string {
	return "content_media_files"
}

// ContentTag 内容标签
type ContentTag struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name       string    `json:"name" gorm:"type:varchar(50);not null;uniqueIndex"`
	UsageCount int64     `json:"usage_count" gorm:"default:0"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName .
func (ContentTag) TableName() string {
	return "content_tags"
}

// ContentTopic 内容话题
type ContentTopic struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string    `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	Description  string    `json:"description" gorm:"type:text"`
	CoverImage   string    `json:"cover_image" gorm:"type:varchar(500)"`
	ContentCount int64     `json:"content_count" gorm:"default:0"`
	IsHot        bool      `json:"is_hot" gorm:"default:false;index"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName .
func (ContentTopic) TableName() string {
	return "content_topics"
}

// ContentTagRelation 内容标签关联表
type ContentTagRelation struct {
	ContentID int64     `json:"content_id" gorm:"primaryKey"`
	TagID     int64     `json:"tag_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName .
func (ContentTagRelation) TableName() string {
	return "content_tag_relations"
}

// ContentTopicRelation 内容话题关联表
type ContentTopicRelation struct {
	ContentID int64     `json:"content_id" gorm:"primaryKey"`
	TopicID   int64     `json:"topic_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName .
func (ContentTopicRelation) TableName() string {
	return "content_topic_relations"
}

// ContentStatusLog 内容状态变更日志
type ContentStatusLog struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ContentID  int64     `json:"content_id" gorm:"not null;index"`
	FromStatus string    `json:"from_status" gorm:"type:varchar(20)"`
	ToStatus   string    `json:"to_status" gorm:"type:varchar(20);not null"`
	OperatorID int64     `json:"operator_id" gorm:"not null"`
	Reason     string    `json:"reason" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName .
func (ContentStatusLog) TableName() string {
	return "content_status_logs"
}

// ContentStats 内容统计
type ContentStats struct {
	TotalContents     int64 `json:"total_contents"`
	PublishedContents int64 `json:"published_contents"`
	DraftContents     int64 `json:"draft_contents"`
	PendingContents   int64 `json:"pending_contents"`
	TotalViews        int64 `json:"total_views"`
	TotalLikes        int64 `json:"total_likes"`
}

// SearchContentParams 搜索内容参数
type SearchContentParams struct {
	Keyword   string
	Type      string
	Status    string
	TagIDs    []int64
	TopicIDs  []int64
	AuthorID  int64
	Page      int32
	PageSize  int32
	SortBy    string
	SortOrder string
}

// ValidateContentType 验证内容类型
func ValidateContentType(contentType string) bool {
	validTypes := []string{
		ContentTypeText, ContentTypeImage, ContentTypeVideo,
		ContentTypeAudio, ContentTypeMixed, ContentTypeTemplate,
	}
	for _, t := range validTypes {
		if t == contentType {
			return true
		}
	}
	return false
}

// ValidateContentStatus 验证内容状态
func ValidateContentStatus(status string) bool {
	validStatuses := []string{
		ContentStatusDraft, ContentStatusPending, ContentStatusPublished,
		ContentStatusRejected, ContentStatusDeleted,
	}
	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// CanTransitionStatus 检查状态转换是否合法
func CanTransitionStatus(from, to string) bool {
	// 定义合法的状态转换
	transitions := map[string][]string{
		ContentStatusDraft: {
			ContentStatusPending,   // 草稿 -> 待审核
			ContentStatusPublished, // 草稿 -> 已发布 (自动发布)
			ContentStatusDeleted,   // 草稿 -> 已删除
		},
		ContentStatusPending: {
			ContentStatusPublished, // 待审核 -> 已发布
			ContentStatusRejected,  // 待审核 -> 已拒绝
			ContentStatusDraft,     // 待审核 -> 草稿 (退回修改)
		},
		ContentStatusPublished: {
			ContentStatusDeleted, // 已发布 -> 已删除
		},
		ContentStatusRejected: {
			ContentStatusDraft,   // 已拒绝 -> 草稿 (重新编辑)
			ContentStatusDeleted, // 已拒绝 -> 已删除
		},
	}

	allowedTransitions, exists := transitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}
	return false
}
