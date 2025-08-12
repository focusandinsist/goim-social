package handler

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
	"goim-social/pkg/telemetry"
)

// getGroupMemberIDsImpl 获取群组成员ID列表实现
func (h *GRPCHandler) getGroupMemberIDsImpl(ctx context.Context, req *rest.GetGroupMemberIDsRequest) (*rest.GetGroupMemberIDsResponse, error) {
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

// validateGroupMemberImpl 验证群成员身份实现
func (h *GRPCHandler) validateGroupMemberImpl(ctx context.Context, req *rest.ValidateGroupMemberRequest) (*rest.ValidateGroupMemberResponse, error) {
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
