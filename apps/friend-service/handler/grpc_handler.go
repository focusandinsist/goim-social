package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/friend-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC协议处理器
type GRPCHandler struct {
	rest.UnimplementedFriendEventServiceServer
	svc *service.Service
	log logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc: svc,
		log: log,
	}
}

// NotifyFriendEvent 通知好友事件
func (g *GRPCHandler) NotifyFriendEvent(ctx context.Context, req *rest.NotifyFriendEventRequest) (*rest.NotifyFriendEventResponse, error) {
	event := req.GetEvent()
	if event == nil {
		return &rest.NotifyFriendEventResponse{
			Success: false,
			Message: "event is nil",
		}, nil
	}

	switch event.Type {
	case rest.FriendEventType_ADD_FRIEND:
		err := g.svc.AddFriend(ctx, event.UserId, event.FriendId, event.Remark)
		if err != nil {
			return &rest.NotifyFriendEventResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}
	case rest.FriendEventType_DELETE_FRIEND:
		err := g.svc.DeleteFriend(ctx, event.UserId, event.FriendId)
		if err != nil {
			return &rest.NotifyFriendEventResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}
	default:
		return &rest.NotifyFriendEventResponse{
			Success: false,
			Message: "unknown event type",
		}, nil
	}

	return &rest.NotifyFriendEventResponse{
		Success: true,
		Message: "ok",
	}, nil
}
