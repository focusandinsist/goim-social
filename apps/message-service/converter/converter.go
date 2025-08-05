package converter

import (
	"goim-social/api/rest"
	"goim-social/apps/message-service/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// MessageModelToProto 将消息Model转换为Protobuf
func (c *Converter) MessageModelToProto(msg *model.Message) *rest.WSMessage {
	if msg == nil {
		return nil
	}
	return &rest.WSMessage{
		MessageId:   msg.MessageID,
		From:        msg.From,
		To:          msg.To,
		GroupId:     msg.GroupID,
		Content:     msg.Content,
		Timestamp:   msg.Timestamp,
		MessageType: int32(msg.MessageType),
		AckId:       msg.AckID,
	}
}

// MessageModelsToProto 将消息Model列表转换为Protobuf列表
func (c *Converter) MessageModelsToProto(messages []*model.Message) []*rest.WSMessage {
	if messages == nil {
		return []*rest.WSMessage{}
	}

	result := make([]*rest.WSMessage, 0, len(messages))
	for _, msg := range messages {
		if protoMsg := c.MessageModelToProto(msg); protoMsg != nil {
			result = append(result, protoMsg)
		}
	}
	return result
}

// 响应构建方法

// BuildSendMessageResponse 构建发送消息响应
func (c *Converter) BuildSendMessageResponse(messageID int64, ackID string) *rest.SendMessageResponse {
	return &rest.SendMessageResponse{
		MessageId: messageID,
		AckId:     ackID,
	}
}

// BuildSendWSMessageResponse 构建发送WebSocket消息响应
func (c *Converter) BuildSendWSMessageResponse(success bool, message string) *rest.SendWSMessageResponse {
	return &rest.SendWSMessageResponse{
		Success: success,
		Message: message,
	}
}

// BuildGetHistoryResponse 构建获取历史消息响应
func (c *Converter) BuildGetHistoryResponse(messages []*model.Message, total int64, page, size int32) *rest.GetHistoryResponse {
	return &rest.GetHistoryResponse{
		Messages: c.MessageModelsToProto(messages),
		Total:    int32(total),
		Page:     page,
		Size:     size,
	}
}

// BuildGetUnreadMessagesResponse 构建获取未读消息响应
func (c *Converter) BuildGetUnreadMessagesResponse(success bool, message string, messages []*model.Message) *rest.GetUnreadMessagesResponse {
	return &rest.GetUnreadMessagesResponse{
		Success:  success,
		Message:  message,
		Messages: c.MessageModelsToProto(messages),
		Total:    int32(len(messages)),
	}
}

// BuildMarkMessagesReadResponse 构建标记消息已读响应
func (c *Converter) BuildMarkMessagesReadResponse(success bool, message string, failedIDs []int64) *rest.MarkMessagesReadResponse {
	return &rest.MarkMessagesReadResponse{
		Success:   success,
		Message:   message,
		FailedIds: failedIDs,
	}
}

// 便捷方法：构建错误响应

// BuildErrorGetHistoryResponse 构建获取历史消息错误响应
func (c *Converter) BuildErrorGetHistoryResponse(page, size int32) *rest.GetHistoryResponse {
	return &rest.GetHistoryResponse{
		Messages: []*rest.WSMessage{},
		Total:    0,
		Page:     page,
		Size:     size,
	}
}

// BuildErrorGetUnreadMessagesResponse 构建获取未读消息错误响应
func (c *Converter) BuildErrorGetUnreadMessagesResponse(message string) *rest.GetUnreadMessagesResponse {
	return &rest.GetUnreadMessagesResponse{
		Success:  false,
		Message:  message,
		Messages: []*rest.WSMessage{},
		Total:    0,
	}
}

// BuildErrorMarkMessagesReadResponse 构建标记消息已读错误响应
func (c *Converter) BuildErrorMarkMessagesReadResponse(message string) *rest.MarkMessagesReadResponse {
	return &rest.MarkMessagesReadResponse{
		Success:   false,
		Message:   message,
		FailedIds: []int64{},
	}
}

// BuildErrorSendWSMessageResponse 构建发送WebSocket消息错误响应
func (c *Converter) BuildErrorSendWSMessageResponse(message string) *rest.SendWSMessageResponse {
	return &rest.SendWSMessageResponse{
		Success: false,
		Message: message,
	}
}

// 便捷方法：构建成功响应

// BuildSuccessSendWSMessageResponse 构建发送WebSocket消息成功响应
func (c *Converter) BuildSuccessSendWSMessageResponse(message string) *rest.SendWSMessageResponse {
	return &rest.SendWSMessageResponse{
		Success: true,
		Message: message,
	}
}

// BuildSuccessGetUnreadMessagesResponse 构建获取未读消息成功响应
func (c *Converter) BuildSuccessGetUnreadMessagesResponse(messages []*model.Message) *rest.GetUnreadMessagesResponse {
	return &rest.GetUnreadMessagesResponse{
		Success:  true,
		Message:  "获取未读消息成功",
		Messages: c.MessageModelsToProto(messages),
		Total:    int32(len(messages)),
	}
}

// BuildSuccessMarkMessagesReadResponse 构建标记消息已读成功响应
func (c *Converter) BuildSuccessMarkMessagesReadResponse(failedIDs []int64) *rest.MarkMessagesReadResponse {
	var message string
	if len(failedIDs) > 0 {
		message = "部分消息标记已读失败"
	} else {
		message = "标记已读成功"
	}
	
	return &rest.MarkMessagesReadResponse{
		Success:   len(failedIDs) == 0,
		Message:   message,
		FailedIds: failedIDs,
	}
}
