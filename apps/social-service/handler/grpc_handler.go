package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/social-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedSocialServiceServer
	svc    *service.Service
	logger logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, logger logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:    svc,
		logger: logger,
	}
}

// NotifyFriendEvent 通知好友事件
func (h *GRPCHandler) NotifyFriendEvent(ctx context.Context, req *rest.NotifyFriendEventRequest) (*rest.NotifyFriendEventResponse, error) {
	return h.notifyFriendEventImpl(ctx, req)
}

// GetGroupMemberIDs 获取群组成员ID列表
func (h *GRPCHandler) GetGroupMemberIDs(ctx context.Context, req *rest.GetGroupMemberIDsRequest) (*rest.GetGroupMemberIDsResponse, error) {
	return h.getGroupMemberIDsImpl(ctx, req)
}

// ValidateGroupMember 验证群成员身份
func (h *GRPCHandler) ValidateGroupMember(ctx context.Context, req *rest.ValidateGroupMemberRequest) (*rest.ValidateGroupMemberResponse, error) {
	return h.validateGroupMemberImpl(ctx, req)
}

// ValidateFriendship 验证好友关系
func (h *GRPCHandler) ValidateFriendship(ctx context.Context, req *rest.ValidateFriendshipRequest) (*rest.ValidateFriendshipResponse, error) {
	return h.validateFriendshipImpl(ctx, req)
}

// GetUserSocialInfo 获取用户社交信息汇总
func (h *GRPCHandler) GetUserSocialInfo(ctx context.Context, req *rest.GetUserSocialInfoRequest) (*rest.GetUserSocialInfoResponse, error) {
	return h.getUserSocialInfoImpl(ctx, req)
}
