package converter

import (
	"strconv"
	"time"

	"goim-social/api/rest"
	"goim-social/apps/message-service/internal/model"
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

// ==================== 历史记录相关转换方法 ====================

// ActionTypeToString 将ActionType枚举转换为字符串
func (c *Converter) ActionTypeToString(actionType rest.ActionType) string {
	switch actionType {
	case rest.ActionType_ACTION_TYPE_VIEW:
		return model.ActionTypeView
	case rest.ActionType_ACTION_TYPE_LIKE:
		return model.ActionTypeLike
	case rest.ActionType_ACTION_TYPE_FAVORITE:
		return model.ActionTypeFavorite
	case rest.ActionType_ACTION_TYPE_SHARE:
		return model.ActionTypeShare
	case rest.ActionType_ACTION_TYPE_COMMENT:
		return model.ActionTypeComment
	case rest.ActionType_ACTION_TYPE_FOLLOW:
		return model.ActionTypeFollow
	case rest.ActionType_ACTION_TYPE_LOGIN:
		return model.ActionTypeLogin
	case rest.ActionType_ACTION_TYPE_SEARCH:
		return model.ActionTypeSearch
	case rest.ActionType_ACTION_TYPE_DOWNLOAD:
		return model.ActionTypeDownload
	case rest.ActionType_ACTION_TYPE_PURCHASE:
		return model.ActionTypePurchase
	case rest.ActionType_ACTION_TYPE_MESSAGE:
		return model.ActionTypeMessage
	default:
		return ""
	}
}

// ObjectTypeToString 将ObjectType枚举转换为字符串
func (c *Converter) ObjectTypeToString(objectType rest.HistoryObjectType) string {
	switch objectType {
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_POST:
		return model.ObjectTypePost
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_ARTICLE:
		return model.ObjectTypeArticle
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_VIDEO:
		return model.ObjectTypeVideo
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_USER:
		return model.ObjectTypeUser
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_PRODUCT:
		return model.ObjectTypeProduct
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_GROUP:
		return model.ObjectTypeGroup
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_MESSAGE:
		return model.ObjectTypeMessage
	default:
		return ""
	}
}

// HistoryRecordModelToProto 将历史记录Model转换为Protobuf
func (c *Converter) HistoryRecordModelToProto(record *model.HistoryRecord) *rest.HistoryRecord {
	if record == nil {
		return nil
	}

	// 将ObjectID转换为int64（简化处理，实际项目中可能需要更复杂的ID映射）
	var recordID int64
	if !record.ID.IsZero() {
		// 使用时间戳作为ID的简化方案
		recordID = record.CreatedAt.Unix()
	}

	return &rest.HistoryRecord{
		Id:          recordID,
		UserId:      record.UserID,
		ActionType:  c.stringToActionType(record.ActionType),
		ObjectType:  c.stringToObjectType(record.ObjectType),
		ObjectId:    record.ObjectID,
		ObjectTitle: record.ObjectTitle,
		ObjectUrl:   record.ObjectURL,
		Metadata:    record.Metadata,
		IpAddress:   record.IPAddress,
		UserAgent:   record.UserAgent,
		DeviceInfo:  record.DeviceInfo,
		Location:    record.Location,
		Duration:    record.Duration,
		CreatedAt:   record.CreatedAt.Format(time.RFC3339),
	}
}

// stringToActionType 将字符串转换为ActionType枚举
func (c *Converter) stringToActionType(actionType string) rest.ActionType {
	switch actionType {
	case model.ActionTypeView:
		return rest.ActionType_ACTION_TYPE_VIEW
	case model.ActionTypeLike:
		return rest.ActionType_ACTION_TYPE_LIKE
	case model.ActionTypeFavorite:
		return rest.ActionType_ACTION_TYPE_FAVORITE
	case model.ActionTypeShare:
		return rest.ActionType_ACTION_TYPE_SHARE
	case model.ActionTypeComment:
		return rest.ActionType_ACTION_TYPE_COMMENT
	case model.ActionTypeFollow:
		return rest.ActionType_ACTION_TYPE_FOLLOW
	case model.ActionTypeLogin:
		return rest.ActionType_ACTION_TYPE_LOGIN
	case model.ActionTypeSearch:
		return rest.ActionType_ACTION_TYPE_SEARCH
	case model.ActionTypeDownload:
		return rest.ActionType_ACTION_TYPE_DOWNLOAD
	case model.ActionTypePurchase:
		return rest.ActionType_ACTION_TYPE_PURCHASE
	case model.ActionTypeMessage:
		return rest.ActionType_ACTION_TYPE_MESSAGE
	default:
		return rest.ActionType_ACTION_TYPE_UNSPECIFIED
	}
}

// stringToObjectType 将字符串转换为ObjectType枚举
func (c *Converter) stringToObjectType(objectType string) rest.HistoryObjectType {
	switch objectType {
	case model.ObjectTypePost:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_POST
	case model.ObjectTypeArticle:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_ARTICLE
	case model.ObjectTypeVideo:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_VIDEO
	case model.ObjectTypeUser:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_USER
	case model.ObjectTypeProduct:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_PRODUCT
	case model.ObjectTypeGroup:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_GROUP
	case model.ObjectTypeMessage:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_MESSAGE
	default:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_UNSPECIFIED
	}
}

// ConvertToHistoryRecords 将请求转换为历史记录模型
func (c *Converter) ConvertToHistoryRecords(actions []*rest.RecordUserActionRequest) []*model.HistoryRecord {
	records := make([]*model.HistoryRecord, 0, len(actions))
	for _, action := range actions {
		record := &model.HistoryRecord{
			UserID:      action.UserId,
			ActionType:  c.ActionTypeToString(action.ActionType),
			ObjectType:  c.ObjectTypeToString(action.ObjectType),
			ObjectID:    action.ObjectId,
			ObjectTitle: action.ObjectTitle,
			ObjectURL:   action.ObjectUrl,
			Metadata:    action.Metadata,
			IPAddress:   action.IpAddress,
			UserAgent:   action.UserAgent,
			DeviceInfo:  action.DeviceInfo,
			Location:    action.Location,
			Duration:    action.Duration,
		}
		records = append(records, record)
	}
	return records
}

// ConvertRecordIDs 将int64数组转换为字符串数组
func (c *Converter) ConvertRecordIDs(recordIDs []int64) []string {
	result := make([]string, len(recordIDs))
	for i, id := range recordIDs {
		result[i] = strconv.FormatInt(id, 10)
	}
	return result
}

// ParseTimeRange 解析时间范围
func (c *Converter) ParseTimeRange(startTimeStr, endTimeStr string) (time.Time, time.Time, error) {
	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	return startTime, endTime, nil
}

// ==================== 历史记录响应构建方法 ====================

// BuildSuccessRecordUserActionResponse 构建成功记录用户行为响应
func (c *Converter) BuildSuccessRecordUserActionResponse() *rest.RecordUserActionResponse {
	return &rest.RecordUserActionResponse{
		Success: true,
		Message: "用户行为记录成功",
	}
}

// BuildErrorRecordUserActionResponse 构建错误记录用户行为响应
func (c *Converter) BuildErrorRecordUserActionResponse(message string) *rest.RecordUserActionResponse {
	return &rest.RecordUserActionResponse{
		Success: false,
		Message: message,
	}
}

// BuildBatchRecordUserActionResponse 构建批量记录用户行为响应
func (c *Converter) BuildBatchRecordUserActionResponse(successCount, failedCount int32, errors []string) *rest.BatchRecordUserActionResponse {
	return &rest.BatchRecordUserActionResponse{
		Success:      failedCount == 0,
		Message:      "批量记录完成",
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Errors:       errors,
	}
}

// BuildErrorBatchRecordUserActionResponse 构建错误批量记录用户行为响应
func (c *Converter) BuildErrorBatchRecordUserActionResponse(message string) *rest.BatchRecordUserActionResponse {
	return &rest.BatchRecordUserActionResponse{
		Success: false,
		Message: message,
	}
}

// BuildGetUserHistoryResponse 构建获取用户历史记录响应
func (c *Converter) BuildGetUserHistoryResponse(records []*model.HistoryRecord, total int64, page, pageSize int32) *rest.GetUserHistoryResponse {
	protoRecords := make([]*rest.HistoryRecord, 0, len(records))
	for _, record := range records {
		if protoRecord := c.HistoryRecordModelToProto(record); protoRecord != nil {
			protoRecords = append(protoRecords, protoRecord)
		}
	}

	return &rest.GetUserHistoryResponse{
		Success:  true,
		Message:  "获取用户历史记录成功",
		Records:  protoRecords,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildErrorGetUserHistoryResponse 构建错误获取用户历史记录响应
func (c *Converter) BuildErrorGetUserHistoryResponse(message string) *rest.GetUserHistoryResponse {
	return &rest.GetUserHistoryResponse{
		Success: false,
		Message: message,
	}
}

// BuildDeleteHistoryResponse 构建删除历史记录响应
func (c *Converter) BuildDeleteHistoryResponse(deletedCount int64) *rest.DeleteHistoryResponse {
	return &rest.DeleteHistoryResponse{
		Success:      true,
		Message:      "删除历史记录成功",
		DeletedCount: int32(deletedCount),
	}
}

// BuildErrorDeleteHistoryResponse 构建错误删除历史记录响应
func (c *Converter) BuildErrorDeleteHistoryResponse(message string) *rest.DeleteHistoryResponse {
	return &rest.DeleteHistoryResponse{
		Success: false,
		Message: message,
	}
}

// BuildGetUserActionStatsResponse 构建获取用户行为统计响应
func (c *Converter) BuildGetUserActionStatsResponse(stats []*model.ActionStatItem) *rest.GetUserActionStatsResponse {
	protoStats := make([]*rest.ActionStatItem, 0, len(stats))
	for _, stat := range stats {
		protoStat := &rest.ActionStatItem{
			Date:       stat.Date,
			ActionType: c.stringToActionType(stat.ActionType),
			Count:      stat.Count,
		}
		protoStats = append(protoStats, protoStat)
	}

	return &rest.GetUserActionStatsResponse{
		Success: true,
		Message: "获取用户行为统计成功",
		Stats:   protoStats,
	}
}

// BuildErrorGetUserActionStatsResponse 构建错误获取用户行为统计响应
func (c *Converter) BuildErrorGetUserActionStatsResponse(message string) *rest.GetUserActionStatsResponse {
	return &rest.GetUserActionStatsResponse{
		Success: false,
		Message: message,
	}
}
