package converter

import (
	"time"

	"goim-social/api/rest"
	"goim-social/apps/logic-service/internal/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// LogicMessageModelToProto 将逻辑消息Model转换为WSMessage Protobuf
func (c *Converter) LogicMessageModelToProto(msg *model.LogicMessage) *rest.WSMessage {
	if msg == nil {
		return nil
	}

	return &rest.WSMessage{
		MessageId:   msg.MessageID,
		From:        msg.From,
		To:          msg.To,
		GroupId:     msg.GroupID,
		Content:     msg.Content,
		MessageType: msg.MessageType,
		Timestamp:   msg.CreatedAt.Unix(),
		AckId:       "", // Logic服务不处理AckID
	}
}

// LogicMessageModelsToProto 将逻辑消息Model列表转换为WSMessage Protobuf列表
func (c *Converter) LogicMessageModelsToProto(messages []*model.LogicMessage) []*rest.WSMessage {
	if messages == nil {
		return []*rest.WSMessage{}
	}

	result := make([]*rest.WSMessage, 0, len(messages))
	for _, msg := range messages {
		if protoMsg := c.LogicMessageModelToProto(msg); protoMsg != nil {
			result = append(result, protoMsg)
		}
	}
	return result
}

// PersonalMessageCopyModelToProto 将个人消息副本Model转换为WSMessage Protobuf
func (c *Converter) PersonalMessageCopyModelToProto(msg *model.PersonalMessageCopy) *rest.WSMessage {
	if msg == nil {
		return nil
	}

	return &rest.WSMessage{
		MessageId:   msg.MessageID,
		From:        msg.From,
		To:          msg.To,
		GroupId:     msg.GroupID,
		Content:     msg.Content,
		MessageType: msg.MessageType,
		Timestamp:   msg.CreatedAt.Unix(),
		AckId:       "", // Logic服务不处理AckID
	}
}

// PersonalMessageCopyModelsToProto 将个人消息副本Model列表转换为WSMessage Protobuf列表
func (c *Converter) PersonalMessageCopyModelsToProto(messages []*model.PersonalMessageCopy) []*rest.WSMessage {
	if messages == nil {
		return []*rest.WSMessage{}
	}

	result := make([]*rest.WSMessage, 0, len(messages))
	for _, msg := range messages {
		if protoMsg := c.PersonalMessageCopyModelToProto(msg); protoMsg != nil {
			result = append(result, protoMsg)
		}
	}
	return result
}

// WSMessageProtoToLogicMessage 将WSMessage Protobuf转换为逻辑消息Model
func (c *Converter) WSMessageProtoToLogicMessage(msg *rest.WSMessage) *model.LogicMessage {
	if msg == nil {
		return nil
	}

	now := time.Now()
	return &model.LogicMessage{
		MessageID:   msg.MessageId,
		From:        msg.From,
		To:          msg.To,
		GroupID:     msg.GroupId,
		Content:     msg.Content,
		MessageType: msg.MessageType,
		ChatType:    c.determineChatType(msg),
		Status:      0, // 默认发送中状态
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// determineChatType 根据WSMessage确定聊天类型
func (c *Converter) determineChatType(msg *rest.WSMessage) int32 {
	if msg.GroupId > 0 {
		return 2 // 群聊
	}
	return 1 // 单聊
}

// 响应构建方法

// BuildSendLogicMessageResponse 构建发送逻辑消息响应
func (c *Converter) BuildSendLogicMessageResponse(success bool, message string, messageID int64, successCount, failureCount int32, failedUsers []int64) *rest.SendLogicMessageResponse {
	return &rest.SendLogicMessageResponse{
		Success:      success,
		Message:      message,
		MessageId:    messageID,
		SuccessCount: successCount,
		FailureCount: failureCount,
		FailedUsers:  failedUsers,
	}
}

// BuildMessageAckResponse 构建消息ACK响应
func (c *Converter) BuildMessageAckResponse(success bool, message string) *rest.MessageAckResponse {
	return &rest.MessageAckResponse{
		Success: success,
		Message: message,
	}
}

// 便捷方法：构建错误响应

// BuildErrorSendLogicMessageResponse 构建发送逻辑消息错误响应
func (c *Converter) BuildErrorSendLogicMessageResponse(message string) *rest.SendLogicMessageResponse {
	return c.BuildSendLogicMessageResponse(false, message, 0, 0, 0, nil)
}

// BuildErrorMessageAckResponse 构建消息ACK错误响应
func (c *Converter) BuildErrorMessageAckResponse(message string) *rest.MessageAckResponse {
	return c.BuildMessageAckResponse(false, message)
}

// 便捷方法：构建成功响应

// BuildSuccessSendLogicMessageResponse 构建发送逻辑消息成功响应
func (c *Converter) BuildSuccessSendLogicMessageResponse(result *model.MessageResult) *rest.SendLogicMessageResponse {
	return c.BuildSendLogicMessageResponse(
		result.Success,
		result.Message,
		result.MessageID,
		int32(result.SuccessCount),
		int32(result.FailureCount),
		result.FailedUsers,
	)
}

// BuildSuccessMessageAckResponse 构建消息ACK成功响应
func (c *Converter) BuildSuccessMessageAckResponse() *rest.MessageAckResponse {
	return c.BuildMessageAckResponse(true, "ACK处理成功")
}

// 从ForwardResult构建响应的便捷方法

// BuildSendLogicMessageResponseFromForwardResult 从ForwardResult构建发送逻辑消息响应
func (c *Converter) BuildSendLogicMessageResponseFromForwardResult(result *model.ForwardResult) *rest.SendLogicMessageResponse {
	return c.BuildSendLogicMessageResponse(
		result.Success,
		result.Message,
		result.MessageID,
		int32(result.SuccessCount),
		int32(result.FailureCount),
		result.FailedUsers,
	)
}

// 从RouteResult构建响应的便捷方法

// BuildSendLogicMessageResponseFromRouteResult 从RouteResult构建发送逻辑消息响应
func (c *Converter) BuildSendLogicMessageResponseFromRouteResult(result *model.RouteResult) *rest.SendLogicMessageResponse {
	if result.Success {
		return c.BuildSendLogicMessageResponse(true, result.Message, result.MessageID, 1, 0, nil)
	}
	return c.BuildSendLogicMessageResponse(false, result.Message, result.MessageID, 0, 1, result.TargetUsers)
}

// HTTP响应构建方法（用于测试接口）

// BuildHTTPRouteMessageResponse 构建HTTP路由消息响应
func (c *Converter) BuildHTTPRouteMessageResponse(result *model.MessageResult) map[string]interface{} {
	return map[string]interface{}{
		"success":       result.Success,
		"message":       result.Message,
		"message_id":    result.MessageID,
		"success_count": result.SuccessCount,
		"failure_count": result.FailureCount,
		"failed_users":  result.FailedUsers,
	}
}

// BuildHTTPErrorResponse 构建HTTP错误响应
func (c *Converter) BuildHTTPErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": message,
	}
}

// BuildHTTPHealthResponse 构建HTTP健康检查响应
func (c *Converter) BuildHTTPHealthResponse(service string, timestamp int64) map[string]interface{} {
	return map[string]interface{}{
		"status":    "ok",
		"service":   service,
		"timestamp": timestamp,
	}
}
