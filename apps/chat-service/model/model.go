package model

import (
	"time"
)

// ChatMessage 聊天消息模型
type ChatMessage struct {
	MessageID   int64     `json:"message_id"`
	From        int64     `json:"from"`
	To          int64     `json:"to"`       // 单聊时的目标用户ID
	GroupID     int64     `json:"group_id"` // 群聊时的群组ID
	Content     string    `json:"content"`
	MessageType int32     `json:"message_type"` // 1:文本 2:图片 3:语音 4:视频 5:文件
	ChatType    int32     `json:"chat_type"`    // 1:单聊 2:群聊
	Status      int32     `json:"status"`       // 0:发送中 1:已发送 2:已送达 3:已读 -1:发送失败
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MessageRouteInfo 消息路由信息
type MessageRouteInfo struct {
	ChatType    int32   `json:"chat_type"`    // 1:单聊 2:群聊
	TargetUsers []int64 `json:"target_users"` // 目标用户列表（单聊时只有一个，群聊时是所有成员）
	GroupID     int64   `json:"group_id"`     // 群聊时的群组ID
}

// MessageDistribution 消息分发结果
type MessageDistribution struct {
	MessageID     int64                  `json:"message_id"`
	OriginalMsg   *ChatMessage           `json:"original_msg"`
	Distributions []*PersonalMessageCopy `json:"distributions"`
	FailedUsers   []int64                `json:"failed_users"`
	TotalUsers    int                    `json:"total_users"`
	SuccessCount  int                    `json:"success_count"`
	FailureCount  int                    `json:"failure_count"`
}

// PersonalMessageCopy 个人消息副本（写扩散后的单聊消息）
type PersonalMessageCopy struct {
	MessageID   int64     `json:"message_id"`
	From        int64     `json:"from"`
	To          int64     `json:"to"`       // 接收者ID
	GroupID     int64     `json:"group_id"` // 原群组ID（保留群聊上下文）
	Content     string    `json:"content"`
	MessageType int32     `json:"message_type"`
	ChatType    int32     `json:"chat_type"` // 固定为1（单聊）
	Status      int32     `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RouteResult 路由结果
type RouteResult struct {
	Success     bool    `json:"success"`
	Message     string  `json:"message"`
	MessageID   int64   `json:"message_id"`
	TargetUsers []int64 `json:"target_users"`
	ChatType    int32   `json:"chat_type"`
}

// ForwardResult 转发结果
type ForwardResult struct {
	Success      bool    `json:"success"`
	Message      string  `json:"message"`
	MessageID    int64   `json:"message_id"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	FailedUsers  []int64 `json:"failed_users"`
}

// MessageResult 消息处理结果
type MessageResult struct {
	Success      bool    `json:"success"`
	Message      string  `json:"message"`
	MessageID    int64   `json:"message_id"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	FailedUsers  []int64 `json:"failed_users"`
}
