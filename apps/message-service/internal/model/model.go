package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 消息状态常量
const (
	MessageStatusSent      = "sent"      // 已发送
	MessageStatusDelivered = "delivered" // 已投递
	MessageStatusRead      = "read"      // 已读
	MessageStatusRevoked   = "revoked"   // 已撤回
)

type Message struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MessageID   int64              `bson:"message_id" json:"message_id"` // 唯一消息ID
	From        int64              `bson:"from" json:"from"`
	To          int64              `bson:"to" json:"to"`
	GroupID     int64              `bson:"group_id" json:"group_id"`
	Content     string             `bson:"content" json:"content"`
	MessageType int                `bson:"message_type" json:"message_type"`         // 消息类型
	Timestamp   int64              `bson:"timestamp" json:"timestamp"`               // 时间戳
	AckID       string             `bson:"ack_id,omitempty" json:"ack_id,omitempty"` // 确认ID，可选存储
	Status      string             `bson:"status" json:"status"`                     // 消息状态：sent/delivered/read/revoked
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type WSMessage struct {
	MessageID   int64  `json:"message_id"`
	From        int64  `json:"from"`
	To          int64  `json:"to"`
	GroupID     int64  `json:"group_id"`
	Content     string `json:"content"`
	Timestamp   int64  `json:"timestamp"`
	MessageType int32  `json:"message_type"`
	AckID       string `json:"ack_id"`
}

type HistoryMessage struct {
	ID        int64     `bson:"_id" json:"id"`
	From      int64     `bson:"from" json:"from"`
	To        int64     `bson:"to" json:"to"`
	GroupID   int64     `bson:"group_id" json:"group_id"`
	Content   string    `bson:"content" json:"content"`
	MsgType   int32     `bson:"msg_type" json:"msg_type"`
	AckID     string    `bson:"ack_id" json:"ack_id"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	Status    int32     `bson:"status" json:"status"` // 0:未读 1:已读 2:撤回等
}

// ==================== 历史记录相关模型 ====================

// 行为类型常量
const (
	ActionTypeView     = "view"
	ActionTypeLike     = "like"
	ActionTypeFavorite = "favorite"
	ActionTypeShare    = "share"
	ActionTypeComment  = "comment"
	ActionTypeFollow   = "follow"
	ActionTypeLogin    = "login"
	ActionTypeSearch   = "search"
	ActionTypeDownload = "download"
	ActionTypePurchase = "purchase"
	ActionTypeMessage  = "message" // 新增：消息发送
)

// 对象类型常量
const (
	ObjectTypePost    = "post"
	ObjectTypeArticle = "article"
	ObjectTypeVideo   = "video"
	ObjectTypeUser    = "user"
	ObjectTypeProduct = "product"
	ObjectTypeGroup   = "group"
	ObjectTypeMessage = "message" // 新增：消息对象
)

// HistoryRecord 历史记录模型（使用MongoDB存储）
type HistoryRecord struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      int64              `bson:"user_id" json:"user_id"`
	ActionType  string             `bson:"action_type" json:"action_type"`
	ObjectType  string             `bson:"object_type" json:"object_type"`
	ObjectID    int64              `bson:"object_id" json:"object_id"`
	ObjectTitle string             `bson:"object_title" json:"object_title"`
	ObjectURL   string             `bson:"object_url" json:"object_url"`
	Metadata    string             `bson:"metadata" json:"metadata"`
	IPAddress   string             `bson:"ip_address" json:"ip_address"`
	UserAgent   string             `bson:"user_agent" json:"user_agent"`
	DeviceInfo  string             `bson:"device_info" json:"device_info"`
	Location    string             `bson:"location" json:"location"`
	Duration    int64              `bson:"duration" json:"duration"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// UserActionStats 用户行为统计模型（使用MongoDB存储）
type UserActionStats struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         int64              `bson:"user_id" json:"user_id"`
	ActionType     string             `bson:"action_type" json:"action_type"`
	TotalCount     int64              `bson:"total_count" json:"total_count"`
	TodayCount     int64              `bson:"today_count" json:"today_count"`
	WeekCount      int64              `bson:"week_count" json:"week_count"`
	MonthCount     int64              `bson:"month_count" json:"month_count"`
	LastActionTime time.Time          `bson:"last_action_time" json:"last_action_time"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// ObjectHotStats 对象热度统计模型（使用MongoDB存储）
type ObjectHotStats struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ObjectType     string             `bson:"object_type" json:"object_type"`
	ObjectID       int64              `bson:"object_id" json:"object_id"`
	ViewCount      int64              `bson:"view_count" json:"view_count"`
	LikeCount      int64              `bson:"like_count" json:"like_count"`
	FavoriteCount  int64              `bson:"favorite_count" json:"favorite_count"`
	ShareCount     int64              `bson:"share_count" json:"share_count"`
	CommentCount   int64              `bson:"comment_count" json:"comment_count"`
	HotScore       float64            `bson:"hot_score" json:"hot_score"`
	LastUpdateTime time.Time          `bson:"last_update_time" json:"last_update_time"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
}

// ActionStatItem 行为统计项
type ActionStatItem struct {
	Date       string `json:"date"`
	ActionType string `json:"action_type"`
	Count      int64  `json:"count"`
}

// 分页参数
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// 时间分组常量
const (
	GroupByDay   = "day"
	GroupByWeek  = "week"
	GroupByMonth = "month"
)
