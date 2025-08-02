package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/interaction-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedInteractionServiceServer
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

// DoInteraction 执行互动
func (h *GRPCHandler) DoInteraction(ctx context.Context, req *rest.DoInteractionRequest) (*rest.DoInteractionResponse, error) {
	objectType := convertObjectTypeFromProto(req.InteractionObjectType)
	interactionType := convertInteractionTypeFromProto(req.InteractionType)

	interaction, err := h.svc.DoInteraction(
		ctx,
		req.UserId,
		req.ObjectId,
		objectType,
		interactionType,
		req.Metadata,
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to do interaction via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("objectID", req.ObjectId))
		return &rest.DoInteractionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.DoInteractionResponse{
		Success:     true,
		Message:     "操作成功",
		Interaction: convertInteractionToProto(interaction),
	}, nil
}

// UndoInteraction 取消互动
func (h *GRPCHandler) UndoInteraction(ctx context.Context, req *rest.UndoInteractionRequest) (*rest.UndoInteractionResponse, error) {
	objectType := convertObjectTypeFromProto(req.InteractionObjectType)
	interactionType := convertInteractionTypeFromProto(req.InteractionType)

	err := h.svc.UndoInteraction(
		ctx,
		req.UserId,
		req.ObjectId,
		objectType,
		interactionType,
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to undo interaction via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("objectID", req.ObjectId))
		return &rest.UndoInteractionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.UndoInteractionResponse{
		Success: true,
		Message: "取消成功",
	}, nil
}

// CheckInteraction 检查互动状态
func (h *GRPCHandler) CheckInteraction(ctx context.Context, req *rest.CheckInteractionRequest) (*rest.CheckInteractionResponse, error) {
	objectType := convertObjectTypeFromProto(req.InteractionObjectType)
	interactionType := convertInteractionTypeFromProto(req.InteractionType)

	hasInteraction, interaction, err := h.svc.CheckInteraction(
		ctx,
		req.UserId,
		req.ObjectId,
		objectType,
		interactionType,
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to check interaction via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("objectID", req.ObjectId))
		return &rest.CheckInteractionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	response := &rest.CheckInteractionResponse{
		Success:        true,
		Message:        "查询成功",
		HasInteraction: hasInteraction,
	}

	if hasInteraction && interaction != nil {
		response.Interaction = convertInteractionToProto(interaction)
	}

	return response, nil
}

// BatchCheckInteraction 批量检查互动状态
func (h *GRPCHandler) BatchCheckInteraction(ctx context.Context, req *rest.BatchCheckInteractionRequest) (*rest.BatchCheckInteractionResponse, error) {
	objectType := convertObjectTypeFromProto(req.InteractionObjectType)
	interactionType := convertInteractionTypeFromProto(req.InteractionType)

	interactions, err := h.svc.BatchCheckInteraction(
		ctx,
		req.UserId,
		req.ObjectIds,
		objectType,
		interactionType,
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to batch check interaction via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("objectIDs", req.ObjectIds))
		return &rest.BatchCheckInteractionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.BatchCheckInteractionResponse{
		Success:      true,
		Message:      "查询成功",
		Interactions: interactions,
	}, nil
}

// GetObjectStats 获取对象统计
func (h *GRPCHandler) GetObjectStats(ctx context.Context, req *rest.GetObjectStatsRequest) (*rest.GetObjectStatsResponse, error) {
	objectType := convertObjectTypeFromProto(req.InteractionObjectType)

	stats, err := h.svc.GetObjectStats(ctx, req.ObjectId, objectType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get object stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectId))
		return &rest.GetObjectStatsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.GetObjectStatsResponse{
		Success: true,
		Message: "获取成功",
		Stats:   convertStatsToProto(stats),
	}, nil
}

// GetBatchObjectStats 批量获取对象统计
func (h *GRPCHandler) GetBatchObjectStats(ctx context.Context, req *rest.GetBatchObjectStatsRequest) (*rest.GetBatchObjectStatsResponse, error) {
	objectType := convertObjectTypeFromProto(req.InteractionObjectType)

	statsList, err := h.svc.GetBatchObjectStats(ctx, req.ObjectIds, objectType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get batch object stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectIDs", req.ObjectIds))
		return &rest.GetBatchObjectStatsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoStats []*rest.InteractionStats
	for _, stats := range statsList {
		protoStats = append(protoStats, convertStatsToProto(stats))
	}

	return &rest.GetBatchObjectStatsResponse{
		Success: true,
		Message: "获取成功",
		Stats:   protoStats,
	}, nil
}

// GetUserInteractions 获取用户互动列表
func (h *GRPCHandler) GetUserInteractions(ctx context.Context, req *rest.GetUserInteractionsRequest) (*rest.GetUserInteractionsResponse, error) {
	objectType := convertObjectTypeFromProto(req.InteractionObjectType)
	interactionType := convertInteractionTypeFromProto(req.InteractionType)

	interactions, total, err := h.svc.GetUserInteractions(
		ctx,
		req.UserId,
		objectType,
		interactionType,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user interactions via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.GetUserInteractionsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoInteractions []*rest.Interaction
	for _, interaction := range interactions {
		protoInteractions = append(protoInteractions, convertInteractionToProto(interaction))
	}

	return &rest.GetUserInteractionsResponse{
		Success:      true,
		Message:      "获取成功",
		Interactions: protoInteractions,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}, nil
}

// GetObjectInteractions 获取对象互动列表
func (h *GRPCHandler) GetObjectInteractions(ctx context.Context, req *rest.GetObjectInteractionsRequest) (*rest.GetObjectInteractionsResponse, error) {
	objectType := convertObjectTypeFromProto(req.InteractionObjectType)
	interactionType := convertInteractionTypeFromProto(req.InteractionType)

	interactions, total, err := h.svc.GetObjectInteractions(
		ctx,
		req.ObjectId,
		objectType,
		interactionType,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to get object interactions via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectId))
		return &rest.GetObjectInteractionsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoInteractions []*rest.Interaction
	for _, interaction := range interactions {
		protoInteractions = append(protoInteractions, convertInteractionToProto(interaction))
	}

	return &rest.GetObjectInteractionsResponse{
		Success:      true,
		Message:      "获取成功",
		Interactions: protoInteractions,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}, nil
}
