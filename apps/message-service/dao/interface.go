package dao

import (
	"context"
	"time"

	"goim-social/apps/message-service/model"
)

// MessageDAO 消息数据访问接口
type MessageDAO interface {
	// 消息相关
	SaveMessage(ctx context.Context, message *model.Message) error
	GetMessage(ctx context.Context, messageID int64) (*model.Message, error)
	GetMessageHistory(ctx context.Context, userID, targetID int64, isGroup bool, limit int32, offset int32) ([]*model.HistoryMessage, error)
	UpdateMessageStatus(ctx context.Context, messageID int64, status string) error
	DeleteMessage(ctx context.Context, messageID int64) error
	
	// 历史记录相关
	RecordUserAction(ctx context.Context, record *model.HistoryRecord) error
	BatchRecordUserAction(ctx context.Context, records []*model.HistoryRecord) error
	GetUserHistory(ctx context.Context, userID int64, actionType, objectType string, startTime, endTime time.Time, page, pageSize int32) ([]*model.HistoryRecord, int64, error)
	DeleteUserHistory(ctx context.Context, userID int64, recordIDs []string) (int64, error)
	
	// 统计相关
	GetUserActionStats(ctx context.Context, userID int64, actionType string, startTime, endTime time.Time, groupBy string) ([]*model.ActionStatItem, error)
	UpdateUserActionStats(ctx context.Context, userID int64, actionType string) error
	GetObjectHotStats(ctx context.Context, objectType string, objectID int64) (*model.ObjectHotStats, error)
	UpdateObjectHotStats(ctx context.Context, objectType string, objectID int64, actionType string, delta int64) error
}
