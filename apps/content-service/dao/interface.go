package dao

import (
	"context"

	"goim-social/apps/content-service/model"
)

// ContentDAO 内容数据访问接口（合并了评论和互动功能）
type ContentDAO interface {
	// 内容管理
	CreateContent(ctx context.Context, content *model.Content) error
	GetContent(ctx context.Context, contentID int64) (*model.Content, error)
	GetContentWithRelations(ctx context.Context, contentID int64) (*model.Content, error)
	UpdateContent(ctx context.Context, content *model.Content) error
	DeleteContent(ctx context.Context, contentID int64) error

	// 内容查询
	GetUserContents(ctx context.Context, authorID int64, status string, page, pageSize int32) ([]*model.Content, int64, error)
	GetContentsByStatus(ctx context.Context, status string, page, pageSize int32) ([]*model.Content, int64, error)

	// 内容统计
	GetContentStats(ctx context.Context, authorID int64) (*model.ContentStats, error)
	IncrementViewCount(ctx context.Context, contentID int64) error

	// 媒体文件管理
	CreateMediaFile(ctx context.Context, mediaFile *model.ContentMediaFile) error
	GetMediaFiles(ctx context.Context, contentID int64) ([]*model.ContentMediaFile, error)
	DeleteMediaFiles(ctx context.Context, contentID int64) error

	// 标签管理
	CreateTag(ctx context.Context, tag *model.ContentTag) error
	GetTag(ctx context.Context, tagID int64) (*model.ContentTag, error)
	GetTagByName(ctx context.Context, name string) (*model.ContentTag, error)
	GetTags(ctx context.Context, keyword string, page, pageSize int32) ([]*model.ContentTag, int64, error)
	UpdateTagUsageCount(ctx context.Context, tagID int64, delta int64) error

	// 话题管理
	CreateTopic(ctx context.Context, topic *model.ContentTopic) error
	GetTopic(ctx context.Context, topicID int64) (*model.ContentTopic, error)
	GetTopicByName(ctx context.Context, name string) (*model.ContentTopic, error)
	GetTopics(ctx context.Context, keyword string, hotOnly bool, page, pageSize int32) ([]*model.ContentTopic, int64, error)
	UpdateTopicContentCount(ctx context.Context, topicID int64, delta int64) error

	// 关联关系管理
	AddContentTags(ctx context.Context, contentID int64, tagIDs []int64) error
	RemoveContentTags(ctx context.Context, contentID int64) error
	AddContentTopics(ctx context.Context, contentID int64, topicIDs []int64) error
	RemoveContentTopics(ctx context.Context, contentID int64) error

	// 状态日志
	CreateStatusLog(ctx context.Context, log *model.ContentStatusLog) error
	GetStatusLogs(ctx context.Context, contentID int64) ([]*model.ContentStatusLog, error)

	// ==================== 评论相关方法 ====================

	// 评论基础操作
	CreateComment(ctx context.Context, comment *model.Comment) error
	GetComment(ctx context.Context, commentID int64) (*model.Comment, error)
	UpdateComment(ctx context.Context, commentID int64, content string) error
	DeleteComment(ctx context.Context, commentID int64) error

	// 评论查询
	GetComments(ctx context.Context, targetID int64, targetType string, parentID int64, sortBy, sortOrder string, page, pageSize int32) ([]*model.Comment, int64, error)
	GetCommentReplies(ctx context.Context, commentID int64, sortBy, sortOrder string, page, pageSize int32) ([]*model.Comment, int64, error)
	GetCommentsByUser(ctx context.Context, userID int64, page, pageSize int32) ([]*model.Comment, int64, error)

	// 评论统计
	UpdateCommentReplyCount(ctx context.Context, commentID int64, delta int32) error
	UpdateTargetCommentCount(ctx context.Context, targetID int64, targetType string, delta int64) error

	// ==================== 互动相关方法 ====================

	// 互动基础操作
	CreateInteraction(ctx context.Context, interaction *model.Interaction) error
	DeleteInteraction(ctx context.Context, userID, targetID int64, targetType, interactionType string) error
	GetInteraction(ctx context.Context, userID, targetID int64, targetType, interactionType string) (*model.Interaction, error)

	// 批量互动查询
	BatchCheckInteractions(ctx context.Context, userID int64, targetIDs []int64, targetType, interactionType string) (map[int64]bool, error)
	GetUserInteractions(ctx context.Context, userID int64, targetType, interactionType string, page, pageSize int32) ([]*model.Interaction, int64, error)

	// 互动统计
	GetInteractionStats(ctx context.Context, targetID int64, targetType string) (*model.InteractionStats, error)
	BatchGetInteractionStats(ctx context.Context, targetIDs []int64, targetType string) ([]*model.InteractionStats, error)
	UpdateInteractionStats(ctx context.Context, targetID int64, targetType, interactionType string, delta int64) error

	// 互动计数
	IncrementInteractionCount(ctx context.Context, targetID int64, targetType, interactionType string) error
	DecrementInteractionCount(ctx context.Context, targetID int64, targetType, interactionType string) error

	// ==================== 聚合查询方法 ====================

	// 内容详情聚合查询（包含评论和互动）
	GetContentWithDetails(ctx context.Context, contentID, userID int64, commentLimit int32) (*model.Content, []*model.Comment, *model.InteractionStats, map[string]bool, error)

	// 内容流聚合查询
	GetContentFeed(ctx context.Context, userID int64, contentType, sortBy string, page, pageSize int32) ([]*model.Content, []*model.InteractionStats, map[int64]map[string]bool, error)

	// 热门内容查询
	GetTrendingContent(ctx context.Context, timeRange, contentType string, limit int32) ([]*model.Content, []*model.InteractionStats, error)

	// ==================== 事务操作方法 ====================

	// 删除内容及其相关数据（评论、互动）
	DeleteContentWithRelated(ctx context.Context, contentID int64) error

	// 批量更新统计数据
	BatchUpdateStats(ctx context.Context, updates []model.StatsUpdate) error
}
