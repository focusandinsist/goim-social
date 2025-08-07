package model

import (
	"time"
)

// Content 内容模型
type Content struct {
	ID            int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	AuthorID      int64      `json:"author_id" gorm:"not null;index"`
	Title         string     `json:"title" gorm:"type:varchar(200);not null"`
	Content       string     `json:"content" gorm:"type:text"`
	Type          string     `json:"type" gorm:"type:varchar(20);not null;index"`
	Status        string     `json:"status" gorm:"type:varchar(20);not null;index;default:'draft'"`
	TemplateData  string     `json:"template_data" gorm:"type:text"` // JSON格式的模板数据
	ViewCount     int64      `json:"view_count" gorm:"default:0"`
	LikeCount     int64      `json:"like_count" gorm:"default:0"`     // 点赞数
	CommentCount  int64      `json:"comment_count" gorm:"default:0"`  // 评论数
	ShareCount    int64      `json:"share_count" gorm:"default:0"`    // 分享数
	FavoriteCount int64      `json:"favorite_count" gorm:"default:0"` // 收藏数
	CreatedAt     time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	PublishedAt   *time.Time `json:"published_at" gorm:"index"`

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

// Comment 评论模型 - 支持多态关联
type Comment struct {
	ID              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TargetID        int64     `json:"target_id" gorm:"not null;index:idx_target"`                      // 被评论的对象ID
	TargetType      string    `json:"target_type" gorm:"type:varchar(20);not null;index:idx_target"`   // 被评论的对象类型
	UserID          int64     `json:"user_id" gorm:"not null;index"`                                   // 评论用户ID
	UserName        string    `json:"user_name" gorm:"type:varchar(100);not null"`                     // 评论用户名（冗余字段）
	UserAvatar      string    `json:"user_avatar" gorm:"type:varchar(500)"`                            // 评论用户头像（冗余字段）
	Content         string    `json:"content" gorm:"type:text;not null"`                               // 评论内容
	ParentID        int64     `json:"parent_id" gorm:"default:0;index"`                                // 父评论ID（0表示顶级评论）
	RootID          int64     `json:"root_id" gorm:"default:0;index"`                                  // 根评论ID（用于快速定位评论树）
	ReplyToUserID   int64     `json:"reply_to_user_id" gorm:"default:0"`                               // 回复的用户ID
	ReplyToUserName string    `json:"reply_to_user_name" gorm:"type:varchar(100)"`                     // 回复的用户名
	Status          string    `json:"status" gorm:"type:varchar(20);not null;index;default:'pending'"` // 评论状态
	LikeCount       int32     `json:"like_count" gorm:"default:0"`                                     // 点赞数
	ReplyCount      int32     `json:"reply_count" gorm:"default:0"`                                    // 回复数
	IsPinned        bool      `json:"is_pinned" gorm:"default:false;index"`                            // 是否置顶
	IsHot           bool      `json:"is_hot" gorm:"default:false;index"`                               // 是否热门
	IPAddress       string    `json:"ip_address" gorm:"type:varchar(45)"`                              // IP地址
	UserAgent       string    `json:"user_agent" gorm:"type:text"`                                     // 用户代理
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName .
func (Comment) TableName() string {
	return "comments"
}

// Interaction 互动模型 - 支持多态关联
type Interaction struct {
	ID              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID          int64     `json:"user_id" gorm:"not null;index:idx_user_target"`                      // 用户ID
	TargetID        int64     `json:"target_id" gorm:"not null;index:idx_user_target,idx_target_type"`    // 目标对象ID
	TargetType      string    `json:"target_type" gorm:"type:varchar(20);not null;index:idx_target_type"` // 目标对象类型
	InteractionType string    `json:"interaction_type" gorm:"type:varchar(20);not null;index"`            // 互动类型
	Metadata        string    `json:"metadata" gorm:"type:text"`                                          // JSON格式的元数据
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName .
func (Interaction) TableName() string {
	return "interactions"
}

// InteractionStats 互动统计表（用于缓存热门数据）
type InteractionStats struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TargetID      int64     `json:"target_id" gorm:"not null;uniqueIndex:idx_target_type"`
	TargetType    string    `json:"target_type" gorm:"type:varchar(20);not null;uniqueIndex:idx_target_type"`
	LikeCount     int64     `json:"like_count" gorm:"default:0"`
	FavoriteCount int64     `json:"favorite_count" gorm:"default:0"`
	ShareCount    int64     `json:"share_count" gorm:"default:0"`
	RepostCount   int64     `json:"repost_count" gorm:"default:0"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName .
func (InteractionStats) TableName() string {
	return "interaction_stats"
}

// StatsUpdate 统计更新结构
type StatsUpdate struct {
	TargetID        int64  `json:"target_id"`
	TargetType      string `json:"target_type"`
	InteractionType string `json:"interaction_type"`
	Delta           int64  `json:"delta"`
}

// CommentQuery 评论查询参数
type CommentQuery struct {
	TargetID   int64
	TargetType string
	ParentID   int64
	UserID     int64
	Status     string
	SortBy     string
	SortOrder  string
	Page       int32
	PageSize   int32
}

// InteractionQuery 互动查询参数
type InteractionQuery struct {
	UserID          int64
	TargetID        int64
	TargetType      string
	InteractionType string
	Page            int32
	PageSize        int32
}

// ContentDetailResult 内容详情聚合结果
type ContentDetailResult struct {
	Content          *Content
	TopComments      []*Comment
	InteractionStats *InteractionStats
	UserInteractions map[string]bool
}

// ContentFeedItem 内容流项目
type ContentFeedItem struct {
	Content          *Content
	InteractionStats *InteractionStats
	UserInteractions map[string]bool
	CommentPreview   int32
}
