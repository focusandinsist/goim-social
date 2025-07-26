package model

import (
	"fmt"
	"time"
)

// Interaction 互动记录表
type Interaction struct {
	ID              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID          int64     `json:"user_id" gorm:"not null;index:idx_user_object"`
	ObjectID        int64     `json:"object_id" gorm:"not null;index:idx_user_object,idx_object_type"`
	ObjectType      string    `json:"object_type" gorm:"type:varchar(20);not null;index:idx_object_type"`
	InteractionType string    `json:"interaction_type" gorm:"type:varchar(20);not null;index"`
	Metadata        string    `json:"metadata" gorm:"type:text"` // JSON格式的元数据
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
	ObjectID      int64     `json:"object_id" gorm:"not null;uniqueIndex:idx_object_type"`
	ObjectType    string    `json:"object_type" gorm:"type:varchar(20);not null;uniqueIndex:idx_object_type"`
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

// InteractionCounter 互动计数器（Redis中的数据结构）
type InteractionCounter struct {
	ObjectID    int64            `json:"object_id"`
	ObjectType  string           `json:"object_type"`
	Counters    map[string]int64 `json:"counters"` // interaction_type -> count
	LastUpdated time.Time        `json:"last_updated"`
}

// UserInteractionCache 用户互动缓存（Redis中的数据结构）
type UserInteractionCache struct {
	UserID       int64                     `json:"user_id"`
	Interactions map[string]map[int64]bool `json:"interactions"` // interaction_type -> object_id -> exists
	LastUpdated  time.Time                 `json:"last_updated"`
}

// InteractionQuery 互动查询参数
type InteractionQuery struct {
	UserID          int64
	ObjectID        int64
	ObjectType      string
	InteractionType string
	Page            int32
	PageSize        int32
	SortBy          string
	SortOrder       string
}

// BatchInteractionQuery 批量互动查询参数
type BatchInteractionQuery struct {
	UserID          int64
	ObjectIDs       []int64
	ObjectType      string
	InteractionType string
}

// InteractionStatsQuery 统计查询参数
type InteractionStatsQuery struct {
	ObjectIDs  []int64
	ObjectType string
}

// ValidateInteractionType 验证互动类型
func ValidateInteractionType(interactionType string) bool {
	for _, t := range ValidInteractionTypes {
		if t == interactionType {
			return true
		}
	}
	return false
}

// ValidateObjectType 验证对象类型
func ValidateObjectType(objectType string) bool {
	for _, t := range ValidObjectTypes {
		if t == objectType {
			return true
		}
	}
	return false
}

// GetInteractionKey 生成互动的唯一键
func GetInteractionKey(userID, objectID int64, objectType, interactionType string) string {
	return fmt.Sprintf("%d:%d:%s:%s", userID, objectID, objectType, interactionType)
}

// GetStatsKey 生成统计的缓存键
func GetStatsKey(objectID int64, objectType string) string {
	return fmt.Sprintf("%s:%d:%s", CacheKeyInteractionStats, objectID, objectType)
}

// GetUserInteractionKey 生成用户互动的缓存键
func GetUserInteractionKey(userID int64, objectType, interactionType string) string {
	return fmt.Sprintf("%s:%d:%s:%s", CacheKeyUserInteraction, userID, objectType, interactionType)
}

// GetHotListKey 生成热门列表的缓存键
func GetHotListKey(objectType, interactionType string) string {
	return fmt.Sprintf("%s:%s:%s", CacheKeyObjectHot, objectType, interactionType)
}

// InteractionEvent 互动事件（用于消息队列）
type InteractionEvent struct {
	EventType       string    `json:"event_type"` // "create" or "delete"
	UserID          int64     `json:"user_id"`
	ObjectID        int64     `json:"object_id"`
	ObjectType      string    `json:"object_type"`
	InteractionType string    `json:"interaction_type"`
	Metadata        string    `json:"metadata"`
	Timestamp       time.Time `json:"timestamp"`
}

// InteractionSummary 互动汇总（用于API响应）
type InteractionSummary struct {
	ObjectID         int64           `json:"object_id"`
	ObjectType       string          `json:"object_type"`
	LikeCount        int64           `json:"like_count"`
	FavoriteCount    int64           `json:"favorite_count"`
	ShareCount       int64           `json:"share_count"`
	RepostCount      int64           `json:"repost_count"`
	UserInteractions map[string]bool `json:"user_interactions,omitempty"` // 当前用户的互动状态
}

// HotObject 热门对象（用于排行榜）
type HotObject struct {
	ObjectID         int64     `json:"object_id"`
	ObjectType       string    `json:"object_type"`
	Score            float64   `json:"score"` // 热度分数
	InteractionCount int64     `json:"interaction_count"`
	LastActiveTime   time.Time `json:"last_active_time"`
}
