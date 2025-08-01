package dao

import (
	"context"
	"time"

	"goim-social/apps/comment-service/model"
)

// CommentDAO 评论数据访问接口
type CommentDAO interface {
	// 基础评论操作
	CreateComment(ctx context.Context, comment *model.Comment) error
	GetComment(ctx context.Context, commentID int64) (*model.Comment, error)
	UpdateComment(ctx context.Context, comment *model.Comment) error
	DeleteComment(ctx context.Context, commentID int64) error

	// 评论查询
	GetComments(ctx context.Context, params *model.GetCommentsParams) ([]*model.Comment, int64, error)
	GetUserComments(ctx context.Context, params *model.GetUserCommentsParams) ([]*model.Comment, int64, error)
	GetCommentsByParent(ctx context.Context, parentID int64, limit int32) ([]*model.Comment, error)
	GetCommentTree(ctx context.Context, rootID int64, maxDepth int) ([]*model.Comment, error)

	// 评论状态管理
	UpdateCommentStatus(ctx context.Context, commentID int64, status string) error
	BatchUpdateCommentStatus(ctx context.Context, commentIDs []int64, status string) error

	// 评论计数管理
	IncrementReplyCount(ctx context.Context, commentID int64, delta int32) error
	IncrementLikeCount(ctx context.Context, commentID int64, delta int32) error
	UpdateHotStatus(ctx context.Context, commentID int64, isHot bool) error
	UpdatePinStatus(ctx context.Context, commentID int64, isPinned bool) error

	// 评论统计
	GetCommentStats(ctx context.Context, objectID int64, objectType string) (*model.CommentStats, error)
	GetBatchCommentStats(ctx context.Context, objectIDs []int64, objectType string) ([]*model.CommentStats, error)
	UpdateCommentStats(ctx context.Context, stats *model.CommentStats) error
	IncrementCommentStats(ctx context.Context, objectID int64, objectType string, field string, delta int64) error

	// 审核日志
	CreateModerationLog(ctx context.Context, log *model.CommentModerationLog) error
	GetModerationLogs(ctx context.Context, commentID int64) ([]*model.CommentModerationLog, error)

	// 点赞管理
	AddCommentLike(ctx context.Context, commentID, userID int64) error
	RemoveCommentLike(ctx context.Context, commentID, userID int64) error
	IsCommentLiked(ctx context.Context, commentID, userID int64) (bool, error)
	GetCommentLikeCount(ctx context.Context, commentID int64) (int64, error)

	// 批量操作
	BatchGetComments(ctx context.Context, commentIDs []int64) ([]*model.Comment, error)
	BatchDeleteComments(ctx context.Context, commentIDs []int64) error

	// 管理员操作
	GetPendingComments(ctx context.Context, page, pageSize int32) ([]*model.Comment, int64, error)
	GetCommentsByStatus(ctx context.Context, status string, page, pageSize int32) ([]*model.Comment, int64, error)

	// 清理操作
	CleanDeletedComments(ctx context.Context, beforeTime time.Time) (int64, error)
	CleanOldModerationLogs(ctx context.Context, beforeTime time.Time) (int64, error)
}
