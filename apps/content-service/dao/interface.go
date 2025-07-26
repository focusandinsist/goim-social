package dao

import (
	"context"

	"websocket-server/apps/content-service/model"
)

// ContentDAO 内容数据访问接口
type ContentDAO interface {
	// 内容管理
	CreateContent(ctx context.Context, content *model.Content) error
	GetContent(ctx context.Context, contentID int64) (*model.Content, error)
	GetContentWithRelations(ctx context.Context, contentID int64) (*model.Content, error)
	UpdateContent(ctx context.Context, content *model.Content) error
	DeleteContent(ctx context.Context, contentID int64) error

	// 内容查询
	SearchContents(ctx context.Context, params *model.SearchContentParams) ([]*model.Content, int64, error)
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
}
