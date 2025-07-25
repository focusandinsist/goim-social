package model

import (
	"time"
)

// Comment 评论模型
type Comment struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ObjectID         int64     `json:"object_id" gorm:"not null;index:idx_object"`                    // 被评论的对象ID
	ObjectType       string    `json:"object_type" gorm:"type:varchar(20);not null;index:idx_object"` // 被评论的对象类型
	UserID           int64     `json:"user_id" gorm:"not null;index"`                                 // 评论用户ID
	UserName         string    `json:"user_name" gorm:"type:varchar(100);not null"`                   // 评论用户名（冗余字段）
	UserAvatar       string    `json:"user_avatar" gorm:"type:varchar(500)"`                          // 评论用户头像（冗余字段）
	Content          string    `json:"content" gorm:"type:text;not null"`                             // 评论内容
	ParentID         int64     `json:"parent_id" gorm:"default:0;index"`                              // 父评论ID（0表示顶级评论）
	RootID           int64     `json:"root_id" gorm:"default:0;index"`                                // 根评论ID（用于快速定位评论树）
	ReplyToUserID    int64     `json:"reply_to_user_id" gorm:"default:0"`                             // 回复的用户ID
	ReplyToUserName  string    `json:"reply_to_user_name" gorm:"type:varchar(100)"`                   // 回复的用户名
	Status           string    `json:"status" gorm:"type:varchar(20);not null;index;default:'pending'"` // 评论状态
	LikeCount        int32     `json:"like_count" gorm:"default:0"`                                   // 点赞数
	ReplyCount       int32     `json:"reply_count" gorm:"default:0"`                                  // 回复数
	IsPinned         bool      `json:"is_pinned" gorm:"default:false;index"`                          // 是否置顶
	IsHot            bool      `json:"is_hot" gorm:"default:false;index"`                             // 是否热门
	IPAddress        string    `json:"ip_address" gorm:"type:varchar(45)"`                            // IP地址
	UserAgent        string    `json:"user_agent" gorm:"type:text"`                                   // 用户代理
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	
	// 关联字段（不存储到数据库）
	Replies []*Comment `json:"replies,omitempty" gorm:"-"` // 回复列表
}

// TableName 指定表名
func (Comment) TableName() string {
	return "comments"
}

// CommentStats 评论统计模型
type CommentStats struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ObjectID      int64     `json:"object_id" gorm:"not null;uniqueIndex:idx_object_stats"`
	ObjectType    string    `json:"object_type" gorm:"type:varchar(20);not null;uniqueIndex:idx_object_stats"`
	TotalCount    int64     `json:"total_count" gorm:"default:0"`    // 总评论数
	ApprovedCount int64     `json:"approved_count" gorm:"default:0"` // 已通过评论数
	PendingCount  int64     `json:"pending_count" gorm:"default:0"`  // 待审核评论数
	TodayCount    int64     `json:"today_count" gorm:"default:0"`    // 今日评论数
	HotCount      int64     `json:"hot_count" gorm:"default:0"`      // 热门评论数
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (CommentStats) TableName() string {
	return "comment_stats"
}

// CommentModerationLog 评论审核日志
type CommentModerationLog struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CommentID   int64     `json:"comment_id" gorm:"not null;index"`
	ModeratorID int64     `json:"moderator_id" gorm:"not null"`
	OldStatus   string    `json:"old_status" gorm:"type:varchar(20);not null"`
	NewStatus   string    `json:"new_status" gorm:"type:varchar(20);not null"`
	Reason      string    `json:"reason" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (CommentModerationLog) TableName() string {
	return "comment_moderation_logs"
}

// CommentLike 评论点赞记录
type CommentLike struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CommentID int64     `json:"comment_id" gorm:"not null;uniqueIndex:idx_comment_user"`
	UserID    int64     `json:"user_id" gorm:"not null;uniqueIndex:idx_comment_user"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (CommentLike) TableName() string {
	return "comment_likes"
}

// 查询参数结构体

// GetCommentsParams 获取评论列表参数
type GetCommentsParams struct {
	ObjectID        int64  `json:"object_id"`
	ObjectType      string `json:"object_type"`
	ParentID        int64  `json:"parent_id"`
	Status          string `json:"status"`
	SortBy          string `json:"sort_by"`
	SortOrder       string `json:"sort_order"`
	Page            int32  `json:"page"`
	PageSize        int32  `json:"page_size"`
	IncludeReplies  bool   `json:"include_replies"`
	MaxReplyCount   int32  `json:"max_reply_count"`
}

// GetUserCommentsParams 获取用户评论参数
type GetUserCommentsParams struct {
	UserID   int64  `json:"user_id"`
	Status   string `json:"status"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
}

// CreateCommentParams 创建评论参数
type CreateCommentParams struct {
	ObjectID         int64  `json:"object_id"`
	ObjectType       string `json:"object_type"`
	UserID           int64  `json:"user_id"`
	UserName         string `json:"user_name"`
	UserAvatar       string `json:"user_avatar"`
	Content          string `json:"content"`
	ParentID         int64  `json:"parent_id"`
	ReplyToUserID    int64  `json:"reply_to_user_id"`
	ReplyToUserName  string `json:"reply_to_user_name"`
	IPAddress        string `json:"ip_address"`
	UserAgent        string `json:"user_agent"`
}

// UpdateCommentParams 更新评论参数
type UpdateCommentParams struct {
	CommentID int64  `json:"comment_id"`
	UserID    int64  `json:"user_id"`
	Content   string `json:"content"`
}

// DeleteCommentParams 删除评论参数
type DeleteCommentParams struct {
	CommentID int64 `json:"comment_id"`
	UserID    int64 `json:"user_id"`
	IsAdmin   bool  `json:"is_admin"`
}

// ModerateCommentParams 审核评论参数
type ModerateCommentParams struct {
	CommentID   int64  `json:"comment_id"`
	ModeratorID int64  `json:"moderator_id"`
	NewStatus   string `json:"new_status"`
	Reason      string `json:"reason"`
}

// PinCommentParams 置顶评论参数
type PinCommentParams struct {
	CommentID  int64 `json:"comment_id"`
	OperatorID int64 `json:"operator_id"`
	IsPinned   bool  `json:"is_pinned"`
}

// 辅助方法

// IsTopLevel 判断是否为顶级评论
func (c *Comment) IsTopLevel() bool {
	return c.ParentID == 0
}

// IsReply 判断是否为回复
func (c *Comment) IsReply() bool {
	return c.ParentID > 0
}

// GetDepth 获取评论层级深度
func (c *Comment) GetDepth() int {
	if c.IsTopLevel() {
		return 0
	}
	if c.ParentID == c.RootID {
		return 1
	}
	return 2 // 最大支持3层，这里简化处理
}

// CanEdit 判断用户是否可以编辑评论
func (c *Comment) CanEdit(userID int64, isAdmin bool) bool {
	if isAdmin {
		return true
	}
	return c.UserID == userID && c.Status != CommentStatusDeleted
}

// CanDelete 判断用户是否可以删除评论
func (c *Comment) CanDelete(userID int64, isAdmin bool) bool {
	if isAdmin {
		return true
	}
	return c.UserID == userID && c.Status != CommentStatusDeleted
}
