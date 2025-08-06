package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/interaction-service/converter"
	"goim-social/apps/interaction-service/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedInteractionServiceServer
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

// DoInteraction 执行互动
func (h *GRPCHandler) DoInteraction(ctx context.Context, req *rest.DoInteractionRequest) (*rest.DoInteractionResponse, error) {
	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

	h.logger.Info(ctx, "gRPC DoInteraction request",
		logger.F("userID", req.UserId),
		logger.F("objectID", req.ObjectId),
		logger.F("interactionType", interactionType))

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
			logger.F("objectID", req.ObjectId),
			logger.F("interactionType", interactionType))
		return h.converter.BuildErrorDoInteractionResponse(err.Error()), nil
	}

	h.logger.Info(ctx, "gRPC DoInteraction successful",
		logger.F("interactionID", interaction.ID),
		logger.F("userID", req.UserId),
		logger.F("objectID", req.ObjectId),
		logger.F("interactionType", interactionType))

	return h.converter.BuildSuccessDoInteractionResponse(interaction), nil
}

// UndoInteraction 取消互动
func (h *GRPCHandler) UndoInteraction(ctx context.Context, req *rest.UndoInteractionRequest) (*rest.UndoInteractionResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

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
		return h.converter.BuildErrorUndoInteractionResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessUndoInteractionResponse(), nil
}

// CheckInteraction 检查互动状态
func (h *GRPCHandler) CheckInteraction(ctx context.Context, req *rest.CheckInteractionRequest) (*rest.CheckInteractionResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

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
		return h.converter.BuildErrorCheckInteractionResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessCheckInteractionResponse(hasInteraction, interaction), nil
}

// BatchCheckInteraction 批量检查互动状态
func (h *GRPCHandler) BatchCheckInteraction(ctx context.Context, req *rest.BatchCheckInteractionRequest) (*rest.BatchCheckInteractionResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

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
		return h.converter.BuildErrorBatchCheckInteractionResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessBatchCheckInteractionResponse(interactions), nil
}

// GetObjectStats 获取对象统计
func (h *GRPCHandler) GetObjectStats(ctx context.Context, req *rest.GetObjectStatsRequest) (*rest.GetObjectStatsResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)

	stats, err := h.svc.GetObjectStats(ctx, req.ObjectId, objectType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get object stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectId))
		return h.converter.BuildErrorGetObjectStatsResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetObjectStatsResponse(stats), nil
}

// GetBatchObjectStats 批量获取对象统计
func (h *GRPCHandler) GetBatchObjectStats(ctx context.Context, req *rest.GetBatchObjectStatsRequest) (*rest.GetBatchObjectStatsResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)

	statsList, err := h.svc.GetBatchObjectStats(ctx, req.ObjectIds, objectType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get batch object stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectIDs", req.ObjectIds))
		return h.converter.BuildErrorGetBatchObjectStatsResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetBatchObjectStatsResponse(statsList), nil
}

// GetUserInteractions 获取用户互动列表
func (h *GRPCHandler) GetUserInteractions(ctx context.Context, req *rest.GetUserInteractionsRequest) (*rest.GetUserInteractionsResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

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
		return h.converter.BuildErrorGetUserInteractionsResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetUserInteractionsResponse(interactions, total, req.Page, req.PageSize), nil
}

// GetObjectInteractions 获取对象互动列表
func (h *GRPCHandler) GetObjectInteractions(ctx context.Context, req *rest.GetObjectInteractionsRequest) (*rest.GetObjectInteractionsResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

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
		return h.converter.BuildErrorGetObjectInteractionsResponse(err.Error()), nil
	}

	return h.converter.BuildSuccessGetObjectInteractionsResponse(interactions, total, req.Page, req.PageSize), nil
}
