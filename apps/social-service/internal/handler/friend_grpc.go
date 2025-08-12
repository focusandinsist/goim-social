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

// notifyFriendEventImpl 通知好友事件实现
func (h *GRPCHandler) notifyFriendEventImpl(ctx context.Context, req *rest.NotifyFriendEventRequest) (*rest.NotifyFriendEventResponse, error) {
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

// validateFriendshipImpl 验证好友关系实现
func (h *GRPCHandler) validateFriendshipImpl(ctx context.Context, req *rest.ValidateFriendshipRequest) (*rest.ValidateFriendshipResponse, error) {
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
