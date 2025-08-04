package handler

import (
	"context"
	"time"

	"goim-social/api/rest"
	"goim-social/apps/history-service/converter"
	"goim-social/apps/history-service/model"
	"goim-social/apps/history-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedHistoryServiceServer
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    log,
	}
}

// CreateHistory 创建历史记录
func (h *GRPCHandler) CreateHistory(ctx context.Context, req *rest.CreateHistoryRequest) (*rest.CreateHistoryResponse, error) {
	params := &model.CreateHistoryParams{
		UserID:      req.UserId,
		ActionType:  h.converter.ActionTypeFromProto(req.ActionType),
		ObjectType:  h.converter.ObjectTypeFromProto(req.HistoryObjectType),
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
		return h.converter.BuildErrorCreateHistoryResponse(err.Error()), nil
	}

	return h.converter.BuildCreateHistoryResponse(true, "历史记录创建成功", record), nil
}

// BatchCreateHistory 批量创建历史记录
func (h *GRPCHandler) BatchCreateHistory(ctx context.Context, req *rest.BatchCreateHistoryRequest) (*rest.BatchCreateHistoryResponse, error) {
	var paramsList []*model.CreateHistoryParams
	for _, r := range req.Records {
		params := &model.CreateHistoryParams{
			UserID:      r.UserId,
			ActionType:  h.converter.ActionTypeFromProto(r.ActionType),
			ObjectType:  h.converter.ObjectTypeFromProto(r.HistoryObjectType),
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
		return h.converter.BuildErrorBatchCreateHistoryResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessBatchCreateHistoryResponse(int32(len(records)), records), nil
}

// GetUserHistory 获取用户历史记录
func (h *GRPCHandler) GetUserHistory(ctx context.Context, req *rest.GetUserHistoryRequest) (*rest.GetUserHistoryResponse, error) {
	params := &model.GetUserHistoryParams{
		UserID:     req.UserId,
		ActionType: h.converter.ActionTypeFromProto(req.ActionType),
		ObjectType: h.converter.ObjectTypeFromProto(req.HistoryObjectType),
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
		return h.converter.BuildErrorGetUserHistoryResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetUserHistoryResponse(records, total, req.Page, req.PageSize), nil
}

// GetObjectHistory 获取对象历史记录
func (h *GRPCHandler) GetObjectHistory(ctx context.Context, req *rest.GetObjectHistoryRequest) (*rest.GetObjectHistoryResponse, error) {
	params := &model.GetObjectHistoryParams{
		ObjectType: h.converter.ObjectTypeFromProto(req.HistoryObjectType),
		ObjectID:   req.ObjectId,
		ActionType: h.converter.ActionTypeFromProto(req.ActionType),
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
		return h.converter.BuildErrorGetObjectHistoryResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetObjectHistoryResponse(records, total, req.Page, req.PageSize), nil
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

	return h.converter.BuildSuccessDeleteHistoryResponse(deletedCount), nil
}

// ClearUserHistory 清空用户历史记录
func (h *GRPCHandler) ClearUserHistory(ctx context.Context, req *rest.ClearUserHistoryRequest) (*rest.ClearUserHistoryResponse, error) {
	params := &model.ClearUserHistoryParams{
		UserID:     req.UserId,
		ActionType: h.converter.ActionTypeFromProto(req.ActionType),
		ObjectType: h.converter.ObjectTypeFromProto(req.HistoryObjectType),
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
		return h.converter.BuildErrorClearUserHistoryResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessClearUserHistoryResponse(deletedCount), nil
}

// GetUserActionStats 获取用户行为统计
func (h *GRPCHandler) GetUserActionStats(ctx context.Context, req *rest.GetUserActionStatsRequest) (*rest.GetUserActionStatsResponse, error) {
	actionType := h.converter.ActionTypeFromProto(req.ActionType)
	stats, err := h.svc.GetUserActionStats(ctx, req.UserId, actionType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user action stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return h.converter.BuildErrorGetUserActionStatsResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetUserActionStatsResponse(stats), nil
}

// GetHotObjects 获取热门对象
func (h *GRPCHandler) GetHotObjects(ctx context.Context, req *rest.GetHotObjectsRequest) (*rest.GetHotObjectsResponse, error) {
	params := &model.GetHotObjectsParams{
		ObjectType: h.converter.ObjectTypeFromProto(req.HistoryObjectType),
		TimeRange:  req.TimeRange,
		Limit:      req.Limit,
	}

	objects, err := h.svc.GetHotObjects(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get hot objects via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectType", req.HistoryObjectType))
		return h.converter.BuildErrorGetHotObjectsResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetHotObjectsResponse(objects), nil
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
		return h.converter.BuildErrorGetUserActivityStatsResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetUserActivityStatsResponse(stats), nil
}
