package model

import (
	"time"
)

// HistoryRecord 历史记录模型
type HistoryRecord struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      int64     `json:"user_id" gorm:"not null;index:idx_user_time"`                   // 用户ID
	ActionType  string    `json:"action_type" gorm:"type:varchar(20);not null;index"`            // 行为类型
	ObjectType  string    `json:"object_type" gorm:"type:varchar(20);not null;index"`            // 对象类型
	ObjectID    int64     `json:"object_id" gorm:"not null;index:idx_object"`                    // 对象ID
	ObjectTitle string    `json:"object_title" gorm:"type:varchar(500)"`                         // 对象标题（冗余字段）
	ObjectURL   string    `json:"object_url" gorm:"type:varchar(1000)"`                          // 对象URL
	Metadata    string    `json:"metadata" gorm:"type:text"`                                     // 扩展数据（JSON格式）
	IPAddress   string    `json:"ip_address" gorm:"type:varchar(45)"`                            // IP地址
	UserAgent   string    `json:"user_agent" gorm:"type:text"`                                   // 用户代理
	DeviceInfo  string    `json:"device_info" gorm:"type:varchar(200)"`                          // 设备信息
	Location    string    `json:"location" gorm:"type:varchar(200)"`                             // 地理位置
	Duration    int64     `json:"duration" gorm:"default:0"`                                     // 持续时间（秒）
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;index:idx_user_time,idx_time"` // 创建时间
}

// TableName .
func (HistoryRecord) TableName() string {
	return "history_records"
}

// UserActionStats 用户行为统计模型
type UserActionStats struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         int64     `json:"user_id" gorm:"not null;uniqueIndex:idx_user_action"`
	ActionType     string    `json:"action_type" gorm:"type:varchar(20);not null;uniqueIndex:idx_user_action"`
	TotalCount     int64     `json:"total_count" gorm:"default:0"`     // 总次数
	TodayCount     int64     `json:"today_count" gorm:"default:0"`     // 今日次数
	WeekCount      int64     `json:"week_count" gorm:"default:0"`      // 本周次数
	MonthCount     int64     `json:"month_count" gorm:"default:0"`     // 本月次数
	LastActionTime time.Time `json:"last_action_time"`                 // 最后行为时间
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"` // 更新时间
}

// TableName .
func (UserActionStats) TableName() string {
	return "user_action_stats"
}

// ObjectHotStats 对象热度统计模型
type ObjectHotStats struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ObjectType     string    `json:"object_type" gorm:"type:varchar(20);not null;uniqueIndex:idx_object_hot"`
	ObjectID       int64     `json:"object_id" gorm:"not null;uniqueIndex:idx_object_hot"`
	ObjectTitle    string    `json:"object_title" gorm:"type:varchar(500)"`               // 对象标题
	ViewCount      int64     `json:"view_count" gorm:"default:0"`                         // 浏览次数
	LikeCount      int64     `json:"like_count" gorm:"default:0"`                         // 点赞次数
	FavoriteCount  int64     `json:"favorite_count" gorm:"default:0"`                     // 收藏次数
	ShareCount     int64     `json:"share_count" gorm:"default:0"`                        // 分享次数
	CommentCount   int64     `json:"comment_count" gorm:"default:0"`                      // 评论次数
	HotScore       float64   `json:"hot_score" gorm:"type:decimal(10,2);default:0;index"` // 热度分数
	LastActiveTime time.Time `json:"last_active_time"`                                    // 最后活跃时间
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`                    // 更新时间
}

// TableName .
func (ObjectHotStats) TableName() string {
	return "object_hot_stats"
}

// UserActivityStats 用户活跃度统计模型
type UserActivityStats struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         int64     `json:"user_id" gorm:"not null;uniqueIndex:idx_user_date"`
	Date           time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_user_date;index:idx_date"` // 日期
	TotalActions   int64     `json:"total_actions" gorm:"default:0"`                                          // 总行为次数
	UniqueObjects  int64     `json:"unique_objects" gorm:"default:0"`                                         // 互动的唯一对象数
	OnlineDuration int64     `json:"online_duration" gorm:"default:0"`                                        // 在线时长（分钟）
	ActivityScore  float64   `json:"activity_score" gorm:"type:decimal(5,2);default:0"`                       // 活跃度分数
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`                                        // 更新时间
}

// TableName .
func (UserActivityStats) TableName() string {
	return "user_activity_stats"
}

// 查询参数结构体

// GetUserHistoryParams 获取用户历史记录参数
type GetUserHistoryParams struct {
	UserID     int64     `json:"user_id"`
	ActionType string    `json:"action_type"`
	ObjectType string    `json:"object_type"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Page       int32     `json:"page"`
	PageSize   int32     `json:"page_size"`
}

// GetObjectHistoryParams 获取对象历史记录参数
type GetObjectHistoryParams struct {
	ObjectType string    `json:"object_type"`
	ObjectID   int64     `json:"object_id"`
	ActionType string    `json:"action_type"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Page       int32     `json:"page"`
	PageSize   int32     `json:"page_size"`
}

// CreateHistoryParams 创建历史记录参数
type CreateHistoryParams struct {
	UserID      int64  `json:"user_id"`
	ActionType  string `json:"action_type"`
	ObjectType  string `json:"object_type"`
	ObjectID    int64  `json:"object_id"`
	ObjectTitle string `json:"object_title"`
	ObjectURL   string `json:"object_url"`
	Metadata    string `json:"metadata"`
	IPAddress   string `json:"ip_address"`
	UserAgent   string `json:"user_agent"`
	DeviceInfo  string `json:"device_info"`
	Location    string `json:"location"`
	Duration    int64  `json:"duration"`
}

// DeleteHistoryParams 删除历史记录参数
type DeleteHistoryParams struct {
	UserID    int64   `json:"user_id"`
	RecordIDs []int64 `json:"record_ids"`
}

// ClearUserHistoryParams 清空用户历史记录参数
type ClearUserHistoryParams struct {
	UserID     int64     `json:"user_id"`
	ActionType string    `json:"action_type"`
	ObjectType string    `json:"object_type"`
	BeforeTime time.Time `json:"before_time"`
}

// GetHotObjectsParams 获取热门对象参数
type GetHotObjectsParams struct {
	ObjectType string `json:"object_type"`
	TimeRange  string `json:"time_range"`
	Limit      int32  `json:"limit"`
}

// GetUserActivityStatsParams 获取用户活跃度统计参数
type GetUserActivityStatsParams struct {
	UserID    int64     `json:"user_id"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// 辅助方法

// IsValidActionType 检查行为类型是否有效
func IsValidActionType(actionType string) bool {
	validTypes := []string{
		ActionTypeView, ActionTypeLike, ActionTypeFavorite,
		ActionTypeShare, ActionTypeComment, ActionTypeFollow,
		ActionTypeLogin, ActionTypeSearch, ActionTypeDownload,
		ActionTypePurchase,
	}
	for _, t := range validTypes {
		if t == actionType {
			return true
		}
	}
	return false
}

// IsValidObjectType 检查对象类型是否有效
func IsValidObjectType(objectType string) bool {
	validTypes := []string{
		ObjectTypePost, ObjectTypeArticle, ObjectTypeVideo,
		ObjectTypeUser, ObjectTypeProduct, ObjectTypeGroup,
	}
	for _, t := range validTypes {
		if t == objectType {
			return true
		}
	}
	return false
}

// IsValidTimeRange 检查时间范围是否有效
func IsValidTimeRange(timeRange string) bool {
	validRanges := []string{
		TimeRangeToday, TimeRangeWeek, TimeRangeMonth, TimeRangeAll,
	}
	for _, r := range validRanges {
		if r == timeRange {
			return true
		}
	}
	return false
}

// CalculateHotScore 计算热度分数
func (o *ObjectHotStats) CalculateHotScore() float64 {
	score := float64(o.ViewCount)*ViewWeight +
		float64(o.LikeCount)*LikeWeight +
		float64(o.FavoriteCount)*FavoriteWeight +
		float64(o.ShareCount)*ShareWeight +
		float64(o.CommentCount)*CommentWeight

	// 时间衰减因子
	daysSinceActive := time.Since(o.LastActiveTime).Hours() / 24
	if daysSinceActive > 0 {
		decayFactor := 1.0 / (1.0 + daysSinceActive/float64(HotScoreDecayDays))
		score *= decayFactor
	}

	return score
}

// CalculateActivityScore 计算活跃度分数
func (u *UserActivityStats) CalculateActivityScore() float64 {
	// 基础分数：基于行为次数
	baseScore := float64(u.TotalActions) * 0.1

	// 多样性加分：基于互动对象数
	diversityBonus := float64(u.UniqueObjects) * 0.5

	// 时长加分：基于在线时长
	durationBonus := float64(u.OnlineDuration) * 0.01

	totalScore := baseScore + diversityBonus + durationBonus

	// 限制在最大分数范围内
	if totalScore > MaxActivityScore {
		totalScore = MaxActivityScore
	}

	return totalScore
}
