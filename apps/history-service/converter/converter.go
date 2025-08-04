package converter

import (
	"time"

	"goim-social/api/rest"
	"goim-social/apps/history-service/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// HistoryRecordModelToProto 将历史记录Model转换为Protobuf
func (c *Converter) HistoryRecordModelToProto(record *model.HistoryRecord) *rest.HistoryRecord {
	if record == nil {
		return nil
	}

	return &rest.HistoryRecord{
		Id:                record.ID,
		UserId:            record.UserID,
		ActionType:        c.ActionTypeToProto(record.ActionType),
		HistoryObjectType: c.ObjectTypeToProto(record.ObjectType),
		ObjectId:          record.ObjectID,
		ObjectTitle:       record.ObjectTitle,
		ObjectUrl:         record.ObjectURL,
		Metadata:          record.Metadata,
		IpAddress:         record.IPAddress,
		UserAgent:         record.UserAgent,
		DeviceInfo:        record.DeviceInfo,
		Location:          record.Location,
		Duration:          record.Duration,
		CreatedAt:         record.CreatedAt.Format(time.RFC3339),
	}
}

// HistoryRecordModelsToProto 将历史记录Model列表转换为Protobuf列表
func (c *Converter) HistoryRecordModelsToProto(records []*model.HistoryRecord) []*rest.HistoryRecord {
	if records == nil {
		return []*rest.HistoryRecord{}
	}

	result := make([]*rest.HistoryRecord, 0, len(records))
	for _, record := range records {
		if protoRecord := c.HistoryRecordModelToProto(record); protoRecord != nil {
			result = append(result, protoRecord)
		}
	}
	return result
}

// UserActionStatsModelToProto 将用户行为统计Model转换为Protobuf
func (c *Converter) UserActionStatsModelToProto(stats *model.UserActionStats) *rest.UserActionStats {
	if stats == nil {
		return nil
	}

	return &rest.UserActionStats{
		UserId:         stats.UserID,
		ActionType:     c.ActionTypeToProto(stats.ActionType),
		TotalCount:     stats.TotalCount,
		TodayCount:     stats.TodayCount,
		WeekCount:      stats.WeekCount,
		MonthCount:     stats.MonthCount,
		LastActionTime: stats.LastActionTime.Format(time.RFC3339),
	}
}

// UserActionStatsModelsToProto 将用户行为统计Model列表转换为Protobuf列表
func (c *Converter) UserActionStatsModelsToProto(statsList []*model.UserActionStats) []*rest.UserActionStats {
	if statsList == nil {
		return []*rest.UserActionStats{}
	}

	result := make([]*rest.UserActionStats, 0, len(statsList))
	for _, stats := range statsList {
		if protoStats := c.UserActionStatsModelToProto(stats); protoStats != nil {
			result = append(result, protoStats)
		}
	}
	return result
}

// ObjectHotStatsModelToProto 将对象热度统计Model转换为Protobuf
func (c *Converter) ObjectHotStatsModelToProto(stats *model.ObjectHotStats) *rest.ObjectHotStats {
	if stats == nil {
		return nil
	}

	return &rest.ObjectHotStats{
		HistoryObjectType: c.ObjectTypeToProto(stats.ObjectType),
		ObjectId:          stats.ObjectID,
		ObjectTitle:       stats.ObjectTitle,
		ViewCount:         stats.ViewCount,
		LikeCount:         stats.LikeCount,
		FavoriteCount:     stats.FavoriteCount,
		ShareCount:        stats.ShareCount,
		CommentCount:      stats.CommentCount,
		HotScore:          stats.HotScore,
		LastActiveTime:    stats.LastActiveTime.Format(time.RFC3339),
	}
}

// ObjectHotStatsModelsToProto 将对象热度统计Model列表转换为Protobuf列表
func (c *Converter) ObjectHotStatsModelsToProto(statsList []*model.ObjectHotStats) []*rest.ObjectHotStats {
	if statsList == nil {
		return []*rest.ObjectHotStats{}
	}

	result := make([]*rest.ObjectHotStats, 0, len(statsList))
	for _, stats := range statsList {
		if protoStats := c.ObjectHotStatsModelToProto(stats); protoStats != nil {
			result = append(result, protoStats)
		}
	}
	return result
}

// UserActivityStatsModelToProto 将用户活跃度统计Model转换为Protobuf
func (c *Converter) UserActivityStatsModelToProto(stats *model.UserActivityStats) *rest.UserActivityStats {
	if stats == nil {
		return nil
	}

	return &rest.UserActivityStats{
		UserId:         stats.UserID,
		Date:           stats.Date.Format("2006-01-02"),
		TotalActions:   stats.TotalActions,
		UniqueObjects:  stats.UniqueObjects,
		OnlineDuration: stats.OnlineDuration,
		ActivityScore:  stats.ActivityScore,
	}
}

// UserActivityStatsModelsToProto 将用户活跃度统计Model列表转换为Protobuf列表
func (c *Converter) UserActivityStatsModelsToProto(statsList []*model.UserActivityStats) []*rest.UserActivityStats {
	if statsList == nil {
		return []*rest.UserActivityStats{}
	}

	result := make([]*rest.UserActivityStats, 0, len(statsList))
	for _, stats := range statsList {
		if protoStats := c.UserActivityStatsModelToProto(stats); protoStats != nil {
			result = append(result, protoStats)
		}
	}
	return result
}

// 枚举转换方法

// ActionTypeToProto 将行为类型转换为protobuf枚举
func (c *Converter) ActionTypeToProto(actionType string) rest.ActionType {
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
	default:
		return rest.ActionType_ACTION_TYPE_UNSPECIFIED
	}
}

// ActionTypeFromProto 将protobuf枚举转换为行为类型
func (c *Converter) ActionTypeFromProto(actionType rest.ActionType) string {
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
	default:
		return ""
	}
}

// ObjectTypeToProto 将对象类型转换为protobuf枚举
func (c *Converter) ObjectTypeToProto(objectType string) rest.HistoryObjectType {
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
	default:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_UNSPECIFIED
	}
}

// ObjectTypeFromProto 将protobuf枚举转换为对象类型
func (c *Converter) ObjectTypeFromProto(objectType rest.HistoryObjectType) string {
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
	default:
		return ""
	}
}

// 响应构建方法

// BuildCreateHistoryResponse 构建创建历史记录响应
func (c *Converter) BuildCreateHistoryResponse(success bool, message string, record *model.HistoryRecord) *rest.CreateHistoryResponse {
	return &rest.CreateHistoryResponse{
		Success: success,
		Message: message,
		Record:  c.HistoryRecordModelToProto(record),
	}
}

// BuildBatchCreateHistoryResponse 构建批量创建历史记录响应
func (c *Converter) BuildBatchCreateHistoryResponse(success bool, message string, createdCount int32, records []*model.HistoryRecord) *rest.BatchCreateHistoryResponse {
	return &rest.BatchCreateHistoryResponse{
		Success:      success,
		Message:      message,
		CreatedCount: createdCount,
		Records:      c.HistoryRecordModelsToProto(records),
	}
}

// BuildGetUserHistoryResponse 构建获取用户历史记录响应
func (c *Converter) BuildGetUserHistoryResponse(success bool, message string, records []*model.HistoryRecord, total int64, page, pageSize int32) *rest.GetUserHistoryResponse {
	return &rest.GetUserHistoryResponse{
		Success:  success,
		Message:  message,
		Records:  c.HistoryRecordModelsToProto(records),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildGetObjectHistoryResponse 构建获取对象历史记录响应
func (c *Converter) BuildGetObjectHistoryResponse(success bool, message string, records []*model.HistoryRecord, total int64, page, pageSize int32) *rest.GetObjectHistoryResponse {
	return &rest.GetObjectHistoryResponse{
		Success:  success,
		Message:  message,
		Records:  c.HistoryRecordModelsToProto(records),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildDeleteHistoryResponse 构建删除历史记录响应
func (c *Converter) BuildDeleteHistoryResponse(success bool, message string, deletedCount int32) *rest.DeleteHistoryResponse {
	return &rest.DeleteHistoryResponse{
		Success:      success,
		Message:      message,
		DeletedCount: deletedCount,
	}
}

// BuildClearUserHistoryResponse 构建清空用户历史记录响应
func (c *Converter) BuildClearUserHistoryResponse(success bool, message string, deletedCount int32) *rest.ClearUserHistoryResponse {
	return &rest.ClearUserHistoryResponse{
		Success:      success,
		Message:      message,
		DeletedCount: deletedCount,
	}
}

// BuildGetUserActionStatsResponse 构建获取用户行为统计响应
func (c *Converter) BuildGetUserActionStatsResponse(success bool, message string, stats []*model.UserActionStats) *rest.GetUserActionStatsResponse {
	return &rest.GetUserActionStatsResponse{
		Success: success,
		Message: message,
		Stats:   c.UserActionStatsModelsToProto(stats),
	}
}

// BuildGetHotObjectsResponse 构建获取热门对象响应
func (c *Converter) BuildGetHotObjectsResponse(success bool, message string, objects []*model.ObjectHotStats) *rest.GetHotObjectsResponse {
	return &rest.GetHotObjectsResponse{
		Success: success,
		Message: message,
		Objects: c.ObjectHotStatsModelsToProto(objects),
	}
}

// BuildGetUserActivityStatsResponse 构建获取用户活跃度统计响应
func (c *Converter) BuildGetUserActivityStatsResponse(success bool, message string, stats []*model.UserActivityStats) *rest.GetUserActivityStatsResponse {
	return &rest.GetUserActivityStatsResponse{
		Success: success,
		Message: message,
		Stats:   c.UserActivityStatsModelsToProto(stats),
	}
}

// 便捷方法：构建错误响应

// BuildErrorCreateHistoryResponse 构建创建历史记录错误响应
func (c *Converter) BuildErrorCreateHistoryResponse(message string) *rest.CreateHistoryResponse {
	return c.BuildCreateHistoryResponse(false, message, nil)
}

// BuildErrorBatchCreateHistoryResponse 构建批量创建历史记录错误响应
func (c *Converter) BuildErrorBatchCreateHistoryResponse(message string) *rest.BatchCreateHistoryResponse {
	return c.BuildBatchCreateHistoryResponse(false, message, 0, nil)
}

// BuildErrorGetUserHistoryResponse 构建获取用户历史记录错误响应
func (c *Converter) BuildErrorGetUserHistoryResponse(message string) *rest.GetUserHistoryResponse {
	return c.BuildGetUserHistoryResponse(false, message, nil, 0, 0, 0)
}

// BuildErrorGetObjectHistoryResponse 构建获取对象历史记录错误响应
func (c *Converter) BuildErrorGetObjectHistoryResponse(message string) *rest.GetObjectHistoryResponse {
	return c.BuildGetObjectHistoryResponse(false, message, nil, 0, 0, 0)
}

// BuildErrorDeleteHistoryResponse 构建删除历史记录错误响应
func (c *Converter) BuildErrorDeleteHistoryResponse(message string) *rest.DeleteHistoryResponse {
	return c.BuildDeleteHistoryResponse(false, message, 0)
}

// BuildErrorClearUserHistoryResponse 构建清空用户历史记录错误响应
func (c *Converter) BuildErrorClearUserHistoryResponse(message string) *rest.ClearUserHistoryResponse {
	return c.BuildClearUserHistoryResponse(false, message, 0)
}

// BuildErrorGetUserActionStatsResponse 构建获取用户行为统计错误响应
func (c *Converter) BuildErrorGetUserActionStatsResponse(message string) *rest.GetUserActionStatsResponse {
	return c.BuildGetUserActionStatsResponse(false, message, nil)
}

// BuildErrorGetHotObjectsResponse 构建获取热门对象错误响应
func (c *Converter) BuildErrorGetHotObjectsResponse(message string) *rest.GetHotObjectsResponse {
	return c.BuildGetHotObjectsResponse(false, message, nil)
}

// BuildErrorGetUserActivityStatsResponse 构建获取用户活跃度统计错误响应
func (c *Converter) BuildErrorGetUserActivityStatsResponse(message string) *rest.GetUserActivityStatsResponse {
	return c.BuildGetUserActivityStatsResponse(false, message, nil)
}

// 便捷方法：构建成功响应

// BuildSuccessCreateHistoryResponse 构建创建历史记录成功响应
func (c *Converter) BuildSuccessCreateHistoryResponse(record *model.HistoryRecord) *rest.CreateHistoryResponse {
	return c.BuildCreateHistoryResponse(true, "创建成功", record)
}

// BuildSuccessBatchCreateHistoryResponse 构建批量创建历史记录成功响应
func (c *Converter) BuildSuccessBatchCreateHistoryResponse(createdCount int32, records []*model.HistoryRecord) *rest.BatchCreateHistoryResponse {
	return c.BuildBatchCreateHistoryResponse(true, "批量创建成功", createdCount, records)
}

// BuildSuccessGetUserHistoryResponse 构建获取用户历史记录成功响应
func (c *Converter) BuildSuccessGetUserHistoryResponse(records []*model.HistoryRecord, total int64, page, pageSize int32) *rest.GetUserHistoryResponse {
	return c.BuildGetUserHistoryResponse(true, "获取成功", records, total, page, pageSize)
}

// BuildSuccessGetObjectHistoryResponse 构建获取对象历史记录成功响应
func (c *Converter) BuildSuccessGetObjectHistoryResponse(records []*model.HistoryRecord, total int64, page, pageSize int32) *rest.GetObjectHistoryResponse {
	return c.BuildGetObjectHistoryResponse(true, "获取成功", records, total, page, pageSize)
}

// BuildSuccessDeleteHistoryResponse 构建删除历史记录成功响应
func (c *Converter) BuildSuccessDeleteHistoryResponse(deletedCount int32) *rest.DeleteHistoryResponse {
	return c.BuildDeleteHistoryResponse(true, "删除成功", deletedCount)
}

// BuildSuccessClearUserHistoryResponse 构建清空用户历史记录成功响应
func (c *Converter) BuildSuccessClearUserHistoryResponse(deletedCount int32) *rest.ClearUserHistoryResponse {
	return c.BuildClearUserHistoryResponse(true, "清空成功", deletedCount)
}

// BuildSuccessGetUserActionStatsResponse 构建获取用户行为统计成功响应
func (c *Converter) BuildSuccessGetUserActionStatsResponse(stats []*model.UserActionStats) *rest.GetUserActionStatsResponse {
	return c.BuildGetUserActionStatsResponse(true, "获取成功", stats)
}

// BuildSuccessGetHotObjectsResponse 构建获取热门对象成功响应
func (c *Converter) BuildSuccessGetHotObjectsResponse(objects []*model.ObjectHotStats) *rest.GetHotObjectsResponse {
	return c.BuildGetHotObjectsResponse(true, "获取成功", objects)
}

// BuildSuccessGetUserActivityStatsResponse 构建获取用户活跃度统计成功响应
func (c *Converter) BuildSuccessGetUserActivityStatsResponse(stats []*model.UserActivityStats) *rest.GetUserActivityStatsResponse {
	return c.BuildGetUserActivityStatsResponse(true, "获取成功", stats)
}
