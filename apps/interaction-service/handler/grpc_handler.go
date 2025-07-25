package handler

import (
	"context"
	"time"

	"websocket-server/api/rest"
	"websocket-server/apps/interaction-service/model"
	"websocket-server/apps/interaction-service/service"
	"websocket-server/pkg/logger"
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

// 转换函数

// convertInteractionToProto 将互动模型转换为protobuf格式
func convertInteractionToProto(interaction *model.Interaction) *rest.Interaction {
	if interaction == nil {
		return nil
	}

	return &rest.Interaction{
		Id:                    interaction.ID,
		UserId:                interaction.UserID,
		ObjectId:              interaction.ObjectID,
		InteractionObjectType: convertObjectTypeToProto(interaction.ObjectType),
		InteractionType:       convertInteractionTypeToProto(interaction.InteractionType),
		Metadata:              interaction.Metadata,
		CreatedAt:             interaction.CreatedAt.Format(time.RFC3339),
	}
}

// convertStatsToProto 将统计模型转换为protobuf格式
func convertStatsToProto(stats *model.InteractionStats) *rest.InteractionStats {
	if stats == nil {
		return nil
	}

	return &rest.InteractionStats{
		ObjectId:              stats.ObjectID,
		InteractionObjectType: convertObjectTypeToProto(stats.ObjectType),
		LikeCount:             stats.LikeCount,
		FavoriteCount:         stats.FavoriteCount,
		ShareCount:            stats.ShareCount,
		RepostCount:           stats.RepostCount,
	}
}

// convertObjectTypeToProto 将对象类型转换为protobuf枚举
func convertObjectTypeToProto(objectType string) rest.InteractionObjectType {
	switch objectType {
	case model.ObjectTypePost:
		return rest.InteractionObjectType_OBJECT_TYPE_POST
	case model.ObjectTypeComment:
		return rest.InteractionObjectType_OBJECT_TYPE_COMMENT
	case model.ObjectTypeUser:
		return rest.InteractionObjectType_OBJECT_TYPE_USER
	default:
		return rest.InteractionObjectType_OBJECT_TYPE_UNSPECIFIED
	}
}

// convertObjectTypeFromProto 将protobuf枚举转换为对象类型
func convertObjectTypeFromProto(objectType rest.InteractionObjectType) string {
	switch objectType {
	case rest.InteractionObjectType_OBJECT_TYPE_POST:
		return model.ObjectTypePost
	case rest.InteractionObjectType_OBJECT_TYPE_COMMENT:
		return model.ObjectTypeComment
	case rest.InteractionObjectType_OBJECT_TYPE_USER:
		return model.ObjectTypeUser
	default:
		return ""
	}
}

// convertInteractionTypeToProto 将互动类型转换为protobuf枚举
func convertInteractionTypeToProto(interactionType string) rest.InteractionType {
	switch interactionType {
	case model.InteractionTypeLike:
		return rest.InteractionType_INTERACTION_TYPE_LIKE
	case model.InteractionTypeFavorite:
		return rest.InteractionType_INTERACTION_TYPE_FAVORITE
	case model.InteractionTypeShare:
		return rest.InteractionType_INTERACTION_TYPE_SHARE
	case model.InteractionTypeRepost:
		return rest.InteractionType_INTERACTION_TYPE_REPOST
	default:
		return rest.InteractionType_INTERACTION_TYPE_UNSPECIFIED
	}
}

// convertInteractionTypeFromProto 将protobuf枚举转换为互动类型
func convertInteractionTypeFromProto(interactionType rest.InteractionType) string {
	switch interactionType {
	case rest.InteractionType_INTERACTION_TYPE_LIKE:
		return model.InteractionTypeLike
	case rest.InteractionType_INTERACTION_TYPE_FAVORITE:
		return model.InteractionTypeFavorite
	case rest.InteractionType_INTERACTION_TYPE_SHARE:
		return model.InteractionTypeShare
	case rest.InteractionType_INTERACTION_TYPE_REPOST:
		return model.InteractionTypeRepost
	default:
		return ""
	}
}
