package handler

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/api/rest"
	"goim-social/apps/social-service/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
	"goim-social/pkg/telemetry"
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
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.grpc.NotifyFriendEvent")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.String("friend.event_type", req.Event.Type.String()),
		attribute.Int64("friend.user_id", req.Event.UserId),
		attribute.Int64("friend.friend_id", req.Event.FriendId),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.Event.UserId)

	h.logger.Info(ctx, "Received friend event notification",
		logger.F("type", req.Event.Type.String()),
		logger.F("userID", req.Event.UserId),
		logger.F("friendID", req.Event.FriendId))

	// 这里可以添加好友事件的处理逻辑，比如发送通知、更新缓存等
	// 目前只是记录日志

	span.SetStatus(codes.Ok, "friend event processed successfully")
	return &rest.NotifyFriendEventResponse{
		Success: true,
		Message: "好友事件处理成功",
	}, nil
}

// GetGroupMemberIDs 获取群组成员ID列表
func (h *GRPCHandler) GetGroupMemberIDs(ctx context.Context, req *rest.GetGroupMemberIDsRequest) (*rest.GetGroupMemberIDsResponse, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.grpc.GetGroupMemberIDs")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("group.id", req.GroupId))

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, req.GroupId)

	memberIDs, err := h.svc.GetMemberIDs(ctx, req.GroupId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get member IDs")
		h.logger.Error(ctx, "Failed to get group member IDs",
			logger.F("error", err.Error()),
			logger.F("groupID", req.GroupId))
		return &rest.GetGroupMemberIDsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	span.SetAttributes(attribute.Int("group.member_count", len(memberIDs)))
	span.SetStatus(codes.Ok, "member IDs retrieved successfully")

	h.logger.Info(ctx, "Group member IDs retrieved successfully",
		logger.F("groupID", req.GroupId),
		logger.F("memberCount", len(memberIDs)))

	return &rest.GetGroupMemberIDsResponse{
		Success:   true,
		Message:   "获取群成员ID列表成功",
		MemberIds: memberIDs,
	}, nil
}

// ValidateGroupMember 验证群成员身份
func (h *GRPCHandler) ValidateGroupMember(ctx context.Context, req *rest.ValidateGroupMemberRequest) (*rest.ValidateGroupMemberResponse, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.grpc.ValidateGroupMember")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("group.id", req.GroupId),
		attribute.Int64("group.user_id", req.UserId),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, req.GroupId)
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	isMember, err := h.svc.ValidateGroupMembership(ctx, req.UserId, req.GroupId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate group membership")
		h.logger.Error(ctx, "Failed to validate group membership",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("groupID", req.GroupId))
		return &rest.ValidateGroupMemberResponse{
			Success:  false,
			Message:  err.Error(),
			IsMember: false,
		}, nil
	}

	span.SetAttributes(attribute.Bool("group.is_member", isMember))
	span.SetStatus(codes.Ok, "group membership validated successfully")

	h.logger.Info(ctx, "Group membership validated successfully",
		logger.F("userID", req.UserId),
		logger.F("groupID", req.GroupId),
		logger.F("isMember", isMember))

	return &rest.ValidateGroupMemberResponse{
		Success:  true,
		Message:  "验证群成员身份成功",
		IsMember: isMember,
	}, nil
}

// ValidateFriendship 验证好友关系
func (h *GRPCHandler) ValidateFriendship(ctx context.Context, req *rest.ValidateFriendshipRequest) (*rest.ValidateFriendshipResponse, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.grpc.ValidateFriendship")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("friend.user_id", req.UserId),
		attribute.Int64("friend.friend_id", req.FriendId),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	isFriend, err := h.svc.ValidateFriendship(ctx, req.UserId, req.FriendId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate friendship")
		h.logger.Error(ctx, "Failed to validate friendship",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		return &rest.ValidateFriendshipResponse{
			Success:  false,
			Message:  err.Error(),
			IsFriend: false,
		}, nil
	}

	span.SetAttributes(attribute.Bool("friend.is_friend", isFriend))
	span.SetStatus(codes.Ok, "friendship validated successfully")

	h.logger.Info(ctx, "Friendship validated successfully",
		logger.F("userID", req.UserId),
		logger.F("friendID", req.FriendId),
		logger.F("isFriend", isFriend))

	return &rest.ValidateFriendshipResponse{
		Success:  true,
		Message:  "验证好友关系成功",
		IsFriend: isFriend,
	}, nil
}

// GetUserSocialInfo 获取用户社交信息汇总
func (h *GRPCHandler) GetUserSocialInfo(ctx context.Context, req *rest.GetUserSocialInfoRequest) (*rest.GetUserSocialInfoResponse, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.grpc.GetUserSocialInfo")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("social.user_id", req.UserId))

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	socialInfo, err := h.svc.GetUserSocialInfo(ctx, req.UserId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user social info")
		h.logger.Error(ctx, "Failed to get user social info",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.GetUserSocialInfoResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	span.SetAttributes(
		attribute.Int("social.friend_count", socialInfo.FriendCount),
		attribute.Int("social.group_count", socialInfo.GroupCount),
	)
	span.SetStatus(codes.Ok, "user social info retrieved successfully")

	h.logger.Info(ctx, "User social info retrieved successfully",
		logger.F("userID", req.UserId),
		logger.F("friendCount", socialInfo.FriendCount),
		logger.F("groupCount", socialInfo.GroupCount))

	return &rest.GetUserSocialInfoResponse{
		Success: true,
		Message: "获取用户社交信息成功",
		SocialInfo: &rest.UserSocialInfo{
			UserId:      socialInfo.UserID,
			FriendCount: int32(socialInfo.FriendCount),
			GroupCount:  int32(socialInfo.GroupCount),
			FriendIds:   socialInfo.FriendIDs,
			GroupIds:    socialInfo.GroupIDs,
		},
	}, nil
}
