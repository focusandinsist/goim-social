package model

// ChatType 聊天类型常量
const (
	ChatTypePrivate = 1 // 单聊
	ChatTypeGroup   = 2 // 群聊
)

// MessageType 消息类型常量
const (
	MessageTypeText  = 1 // 文本消息
	MessageTypeImage = 2 // 图片消息
	MessageTypeAudio = 3 // 语音消息
	MessageTypeVideo = 4 // 视频消息
	MessageTypeFile  = 5 // 文件消息
)

// MessageStatus 消息状态常量
const (
	MessageStatusFailed    = -1 // 发送失败
	MessageStatusSending   = 0  // 发送中
	MessageStatusSent      = 1  // 已发送
	MessageStatusDelivered = 2  // 已送达
	MessageStatusRead      = 3  // 已读
)
