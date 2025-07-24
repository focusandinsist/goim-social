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
