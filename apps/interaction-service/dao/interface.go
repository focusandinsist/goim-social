package dao

import (
	"context"

	"goim-social/apps/interaction-service/model"
)

// InteractionDAO 互动数据访问接口
type InteractionDAO interface {
	// 基础互动操作
	CreateInteraction(ctx context.Context, interaction *model.Interaction) error
	DeleteInteraction(ctx context.Context, userID, objectID int64, objectType, interactionType string) error
	GetInteraction(ctx context.Context, userID, objectID int64, objectType, interactionType string) (*model.Interaction, error)
	
	// 批量操作
	BatchCheckInteractions(ctx context.Context, query *model.BatchInteractionQuery) (map[int64]bool, error)
	BatchGetInteractions(ctx context.Context, userID int64, objectIDs []int64, objectType, interactionType string) ([]*model.Interaction, error)
	
	// 查询操作
	GetUserInteractions(ctx context.Context, query *model.InteractionQuery) ([]*model.Interaction, int64, error)
	GetObjectInteractions(ctx context.Context, query *model.InteractionQuery) ([]*model.Interaction, int64, error)
	
	// 统计操作
	GetInteractionStats(ctx context.Context, objectID int64, objectType string) (*model.InteractionStats, error)
	BatchGetInteractionStats(ctx context.Context, query *model.InteractionStatsQuery) ([]*model.InteractionStats, error)
	UpdateInteractionStats(ctx context.Context, objectID int64, objectType, interactionType string, delta int64) error
	
	// 计数操作
	GetInteractionCount(ctx context.Context, objectID int64, objectType, interactionType string) (int64, error)
	IncrementInteractionCount(ctx context.Context, objectID int64, objectType, interactionType string) error
	DecrementInteractionCount(ctx context.Context, objectID int64, objectType, interactionType string) error
	
	// 热门数据
	GetHotObjects(ctx context.Context, objectType, interactionType string, limit int32) ([]*model.HotObject, error)
	UpdateHotScore(ctx context.Context, objectID int64, objectType string, score float64) error
}
