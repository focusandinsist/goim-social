package dao

import (
	"context"
	"time"

	"goim-social/apps/history-service/model"
)

// HistoryDAO 历史记录数据访问接口
type HistoryDAO interface {
	// 基础历史记录操作
	CreateHistory(ctx context.Context, record *model.HistoryRecord) error
	BatchCreateHistory(ctx context.Context, records []*model.HistoryRecord) error
	GetUserHistory(ctx context.Context, params *model.GetUserHistoryParams) ([]*model.HistoryRecord, int64, error)
	GetObjectHistory(ctx context.Context, params *model.GetObjectHistoryParams) ([]*model.HistoryRecord, int64, error)
	DeleteHistory(ctx context.Context, userID int64, recordIDs []int64) (int32, error)
	ClearUserHistory(ctx context.Context, params *model.ClearUserHistoryParams) (int32, error)

	// 用户行为统计
	GetUserActionStats(ctx context.Context, userID int64, actionType string) (*model.UserActionStats, error)
	GetAllUserActionStats(ctx context.Context, userID int64) ([]*model.UserActionStats, error)
	UpdateUserActionStats(ctx context.Context, stats *model.UserActionStats) error
	IncrementUserActionStats(ctx context.Context, userID int64, actionType string) error

	// 对象热度统计
	GetObjectHotStats(ctx context.Context, objectType string, objectID int64) (*model.ObjectHotStats, error)
	GetHotObjects(ctx context.Context, params *model.GetHotObjectsParams) ([]*model.ObjectHotStats, error)
	UpdateObjectHotStats(ctx context.Context, stats *model.ObjectHotStats) error
	IncrementObjectHotStats(ctx context.Context, objectType string, objectID int64, actionType string) error

	// 用户活跃度统计
	GetUserActivityStats(ctx context.Context, params *model.GetUserActivityStatsParams) ([]*model.UserActivityStats, error)
	UpdateUserActivityStats(ctx context.Context, stats *model.UserActivityStats) error
	IncrementUserActivityStats(ctx context.Context, userID int64, date time.Time, actionCount int64, objectID int64) error

	// 数据清理
	CleanOldRecords(ctx context.Context, beforeTime time.Time) (int64, error)
	CleanOldStats(ctx context.Context, beforeTime time.Time) (int64, error)

	// 批量操作
	BatchGetUserHistory(ctx context.Context, userIDs []int64, actionType string, limit int32) (map[int64][]*model.HistoryRecord, error)
	BatchGetObjectStats(ctx context.Context, objectType string, objectIDs []int64) (map[int64]*model.ObjectHotStats, error)

	// 统计查询
	GetUserActionCount(ctx context.Context, userID int64, actionType string, startTime, endTime time.Time) (int64, error)
	GetObjectActionCount(ctx context.Context, objectType string, objectID int64, actionType string, startTime, endTime time.Time) (int64, error)
	GetTopActiveUsers(ctx context.Context, startTime, endTime time.Time, limit int32) ([]*model.UserActivityStats, error)
	GetUserActionTrend(ctx context.Context, userID int64, actionType string, days int32) (map[string]int64, error)

	// 实时统计
	GetRealtimeStats(ctx context.Context, objectType string, objectID int64) (map[string]int64, error)
	UpdateRealtimeStats(ctx context.Context, objectType string, objectID int64, actionType string, delta int64) error
}
