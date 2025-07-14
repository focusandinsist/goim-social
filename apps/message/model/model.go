package model

import "time"

type Message struct {
	ID        int64     `bson:"_id" json:"id"`
	From      int64     `bson:"from" json:"from"`
	To        int64     `bson:"to" json:"to"`
	GroupID   int64     `bson:"group_id" json:"group_id"`
	Content   string    `bson:"content" json:"content"`
	MsgType   int32     `bson:"msg_type" json:"msg_type"`
	AckID     string    `bson:"ack_id" json:"ack_id"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
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
