package handler

import (
	"context"
	"time"

	"goim-social/api/rest"
	"goim-social/apps/history-service/model"
	"goim-social/apps/history-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedHistoryServiceServer
	svc    *service.Service
	logger logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:    svc,
		logger: log,
	}
}

// CreateHistory 创建历史记录
func (h *GRPCHandler) CreateHistory(ctx context.Context, req *rest.CreateHistoryRequest) (*rest.CreateHistoryResponse, error) {
	params := &model.CreateHistoryParams{
		UserID:      req.UserId,
		ActionType:  convertActionTypeFromProto(req.ActionType),
		ObjectType:  convertObjectTypeFromProto(req.HistoryObjectType),
		ObjectID:    req.ObjectId,
		ObjectTitle: req.ObjectTitle,
		ObjectURL:   req.ObjectUrl,
		Metadata:    req.Metadata,
		IPAddress:   req.IpAddress,
		UserAgent:   req.UserAgent,
		DeviceInfo:  req.DeviceInfo,
		Location:    req.Location,
		Duration:    req.Duration,
	}

	record, err := h.svc.CreateHistory(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to create history via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.CreateHistoryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.CreateHistoryResponse{
		Success: true,
		Message: "历史记录创建成功",
		Record:  convertHistoryRecordToProto(record),
	}, nil
}

// BatchCreateHistory 批量创建历史记录
func (h *GRPCHandler) BatchCreateHistory(ctx context.Context, req *rest.BatchCreateHistoryRequest) (*rest.BatchCreateHistoryResponse, error) {
	var paramsList []*model.CreateHistoryParams
	for _, r := range req.Records {
		params := &model.CreateHistoryParams{
			UserID:      r.UserId,
			ActionType:  convertActionTypeFromProto(r.ActionType),
			ObjectType:  convertObjectTypeFromProto(r.HistoryObjectType),
			ObjectID:    r.ObjectId,
			ObjectTitle: r.ObjectTitle,
			ObjectURL:   r.ObjectUrl,
			Metadata:    r.Metadata,
			IPAddress:   r.IpAddress,
			UserAgent:   r.UserAgent,
			DeviceInfo:  r.DeviceInfo,
			Location:    r.Location,
			Duration:    r.Duration,
		}
		paramsList = append(paramsList, params)
	}

	records, err := h.svc.BatchCreateHistory(ctx, paramsList)
	if err != nil {
		h.logger.Error(ctx, "Failed to batch create history via gRPC",
			logger.F("error", err.Error()),
			logger.F("count", len(req.Records)))
		return &rest.BatchCreateHistoryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoRecords []*rest.HistoryRecord
	for _, record := range records {
		protoRecords = append(protoRecords, convertHistoryRecordToProto(record))
	}

	return &rest.BatchCreateHistoryResponse{
		Success:      true,
		Message:      "批量创建成功",
		CreatedCount: int32(len(records)),
		Records:      protoRecords,
	}, nil
}

// GetUserHistory 获取用户历史记录
func (h *GRPCHandler) GetUserHistory(ctx context.Context, req *rest.GetUserHistoryRequest) (*rest.GetUserHistoryResponse, error) {
	params := &model.GetUserHistoryParams{
		UserID:     req.UserId,
		ActionType: convertActionTypeFromProto(req.ActionType),
		ObjectType: convertObjectTypeFromProto(req.HistoryObjectType),
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	// 解析时间参数
	if req.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			params.StartTime = startTime
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			params.EndTime = endTime
		}
	}

	records, total, err := h.svc.GetUserHistory(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user history via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.GetUserHistoryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoRecords []*rest.HistoryRecord
	for _, record := range records {
		protoRecords = append(protoRecords, convertHistoryRecordToProto(record))
	}

	return &rest.GetUserHistoryResponse{
		Success:  true,
		Message:  "获取成功",
		Records:  protoRecords,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetObjectHistory 获取对象历史记录
func (h *GRPCHandler) GetObjectHistory(ctx context.Context, req *rest.GetObjectHistoryRequest) (*rest.GetObjectHistoryResponse, error) {
	params := &model.GetObjectHistoryParams{
		ObjectType: convertObjectTypeFromProto(req.HistoryObjectType),
		ObjectID:   req.ObjectId,
		ActionType: convertActionTypeFromProto(req.ActionType),
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	// 解析时间参数
	if req.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			params.StartTime = startTime
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			params.EndTime = endTime
		}
	}

	records, total, err := h.svc.GetObjectHistory(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get object history via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectType", req.HistoryObjectType),
			logger.F("objectID", req.ObjectId))
		return &rest.GetObjectHistoryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoRecords []*rest.HistoryRecord
	for _, record := range records {
		protoRecords = append(protoRecords, convertHistoryRecordToProto(record))
	}

	return &rest.GetObjectHistoryResponse{
		Success:  true,
		Message:  "获取成功",
		Records:  protoRecords,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteHistory 删除历史记录
func (h *GRPCHandler) DeleteHistory(ctx context.Context, req *rest.DeleteHistoryRequest) (*rest.DeleteHistoryResponse, error) {
	params := &model.DeleteHistoryParams{
		UserID:    req.UserId,
		RecordIDs: req.RecordIds,
	}

	deletedCount, err := h.svc.DeleteHistory(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete history via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.DeleteHistoryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.DeleteHistoryResponse{
		Success:      true,
		Message:      "删除成功",
		DeletedCount: deletedCount,
	}, nil
}

// ClearUserHistory 清空用户历史记录
func (h *GRPCHandler) ClearUserHistory(ctx context.Context, req *rest.ClearUserHistoryRequest) (*rest.ClearUserHistoryResponse, error) {
	params := &model.ClearUserHistoryParams{
		UserID:     req.UserId,
		ActionType: convertActionTypeFromProto(req.ActionType),
		ObjectType: convertObjectTypeFromProto(req.HistoryObjectType),
	}

	// 解析时间参数
	if req.BeforeTime != "" {
		if beforeTime, err := time.Parse(time.RFC3339, req.BeforeTime); err == nil {
			params.BeforeTime = beforeTime
		}
	}

	deletedCount, err := h.svc.ClearUserHistory(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to clear user history via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.ClearUserHistoryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.ClearUserHistoryResponse{
		Success:      true,
		Message:      "清空成功",
		DeletedCount: deletedCount,
	}, nil
}

// GetUserActionStats 获取用户行为统计
func (h *GRPCHandler) GetUserActionStats(ctx context.Context, req *rest.GetUserActionStatsRequest) (*rest.GetUserActionStatsResponse, error) {
	actionType := convertActionTypeFromProto(req.ActionType)
	stats, err := h.svc.GetUserActionStats(ctx, req.UserId, actionType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user action stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.GetUserActionStatsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoStats []*rest.UserActionStats
	for _, stat := range stats {
		protoStats = append(protoStats, convertUserActionStatsToProto(stat))
	}

	return &rest.GetUserActionStatsResponse{
		Success: true,
		Message: "获取成功",
		Stats:   protoStats,
	}, nil
}

// GetHotObjects 获取热门对象
func (h *GRPCHandler) GetHotObjects(ctx context.Context, req *rest.GetHotObjectsRequest) (*rest.GetHotObjectsResponse, error) {
	params := &model.GetHotObjectsParams{
		ObjectType: convertObjectTypeFromProto(req.HistoryObjectType),
		TimeRange:  req.TimeRange,
		Limit:      req.Limit,
	}

	objects, err := h.svc.GetHotObjects(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get hot objects via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectType", req.HistoryObjectType))
		return &rest.GetHotObjectsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoObjects []*rest.ObjectHotStats
	for _, obj := range objects {
		protoObjects = append(protoObjects, convertObjectHotStatsToProto(obj))
	}

	return &rest.GetHotObjectsResponse{
		Success: true,
		Message: "获取成功",
		Objects: protoObjects,
	}, nil
}

// GetUserActivityStats 获取用户活跃度统计
func (h *GRPCHandler) GetUserActivityStats(ctx context.Context, req *rest.GetUserActivityStatsRequest) (*rest.GetUserActivityStatsResponse, error) {
	params := &model.GetUserActivityStatsParams{
		UserID: req.UserId,
	}

	// 解析日期参数
	if req.StartDate != "" {
		if startDate, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			params.StartDate = startDate
		}
	}
	if req.EndDate != "" {
		if endDate, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			params.EndDate = endDate
		}
	}

	stats, err := h.svc.GetUserActivityStats(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user activity stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.GetUserActivityStatsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoStats []*rest.UserActivityStats
	for _, stat := range stats {
		protoStats = append(protoStats, convertUserActivityStatsToProto(stat))
	}

	return &rest.GetUserActivityStatsResponse{
		Success: true,
		Message: "获取成功",
		Stats:   protoStats,
	}, nil
}

// 转换函数

// convertHistoryRecordToProto 将历史记录模型转换为protobuf格式
func convertHistoryRecordToProto(record *model.HistoryRecord) *rest.HistoryRecord {
	if record == nil {
		return nil
	}

	return &rest.HistoryRecord{
		Id:                record.ID,
		UserId:            record.UserID,
		ActionType:        convertActionTypeToProto(record.ActionType),
		HistoryObjectType: convertObjectTypeToProto(record.ObjectType),
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

// convertUserActionStatsToProto 将用户行为统计模型转换为protobuf格式
func convertUserActionStatsToProto(stats *model.UserActionStats) *rest.UserActionStats {
	if stats == nil {
		return nil
	}

	return &rest.UserActionStats{
		UserId:         stats.UserID,
		ActionType:     convertActionTypeToProto(stats.ActionType),
		TotalCount:     stats.TotalCount,
		TodayCount:     stats.TodayCount,
		WeekCount:      stats.WeekCount,
		MonthCount:     stats.MonthCount,
		LastActionTime: stats.LastActionTime.Format(time.RFC3339),
	}
}

// convertObjectHotStatsToProto 将对象热度统计模型转换为protobuf格式
func convertObjectHotStatsToProto(stats *model.ObjectHotStats) *rest.ObjectHotStats {
	if stats == nil {
		return nil
	}

	return &rest.ObjectHotStats{
		HistoryObjectType: convertObjectTypeToProto(stats.ObjectType),
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

// convertUserActivityStatsToProto 将用户活跃度统计模型转换为protobuf格式
func convertUserActivityStatsToProto(stats *model.UserActivityStats) *rest.UserActivityStats {
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

// convertActionTypeToProto 将行为类型转换为protobuf枚举
func convertActionTypeToProto(actionType string) rest.ActionType {
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

// convertActionTypeFromProto 将protobuf枚举转换为行为类型
func convertActionTypeFromProto(actionType rest.ActionType) string {
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

// convertObjectTypeToProto 将对象类型转换为protobuf枚举
func convertObjectTypeToProto(objectType string) rest.HistoryObjectType {
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

// convertObjectTypeFromProto 将protobuf枚举转换为对象类型
func convertObjectTypeFromProto(objectType rest.HistoryObjectType) string {
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
