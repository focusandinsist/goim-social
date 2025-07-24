package handler

import (
	"context"

	"websocket-server/apps/group-service/service"
	"websocket-server/pkg/logger"
	"websocket-server/api/rest"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedGroupServiceServer
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

// GetGroupMemberIDs 获取群组成员ID列表
func (h *GRPCHandler) GetGroupMemberIDs(ctx context.Context, req *rest.GetGroupMemberIDsRequest) (*rest.GetGroupMemberIDsResponse, error) {
	h.logger.Info(ctx, "收到获取群成员ID列表请求", 
		logger.F("groupID", req.GroupId))

	memberIDs, err := h.svc.GetGroupMemberIDs(ctx, req.GroupId)
	if err != nil {
		h.logger.Error(ctx, "获取群成员ID列表失败", 
			logger.F("groupID", req.GroupId),
			logger.F("error", err.Error()))
		return &rest.GetGroupMemberIDsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	h.logger.Info(ctx, "获取群成员ID列表成功", 
		logger.F("groupID", req.GroupId),
		logger.F("memberCount", len(memberIDs)))

	return &rest.GetGroupMemberIDsResponse{
		Success:   true,
		Message:   "获取群成员列表成功",
		MemberIds: memberIDs,
	}, nil
}

// ValidateGroupMember 验证群成员身份
func (h *GRPCHandler) ValidateGroupMember(ctx context.Context, req *rest.ValidateGroupMemberRequest) (*rest.ValidateGroupMemberResponse, error) {
	h.logger.Info(ctx, "收到验证群成员身份请求", 
		logger.F("groupID", req.GroupId),
		logger.F("userID", req.UserId))

	err := h.svc.ValidateGroupMember(ctx, req.GroupId, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "验证群成员身份失败", 
			logger.F("groupID", req.GroupId),
			logger.F("userID", req.UserId),
			logger.F("error", err.Error()))
		return &rest.ValidateGroupMemberResponse{
			Success:  false,
			Message:  err.Error(),
			IsMember: false,
		}, nil
	}

	h.logger.Info(ctx, "验证群成员身份成功", 
		logger.F("groupID", req.GroupId),
		logger.F("userID", req.UserId))

	return &rest.ValidateGroupMemberResponse{
		Success:  true,
		Message:  "用户是群成员",
		IsMember: true,
	}, nil
}
