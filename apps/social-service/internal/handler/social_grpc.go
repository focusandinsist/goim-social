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

// getUserSocialInfoImpl 获取用户社交信息汇总实现
func (h *GRPCHandler) getUserSocialInfoImpl(ctx context.Context, req *rest.GetUserSocialInfoRequest) (*rest.GetUserSocialInfoResponse, error) {
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
