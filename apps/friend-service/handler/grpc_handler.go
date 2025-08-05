package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/friend-service/converter"
	"goim-social/apps/friend-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC协议处理器
type GRPCHandler struct {
	rest.UnimplementedFriendEventServiceServer
	svc       *service.Service
	converter *converter.Converter
	log       logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		log:       log,
	}
}

// NotifyFriendEvent 通知好友事件
func (g *GRPCHandler) NotifyFriendEvent(ctx context.Context, req *rest.NotifyFriendEventRequest) (*rest.NotifyFriendEventResponse, error) {
	event := req.GetEvent()
	if event == nil {
		return g.converter.BuildNotifyFriendEventResponse(false, "event is nil"), nil
	}

	switch event.Type {
	case rest.FriendEventType_ADD_FRIEND:
		err := g.svc.AddFriend(ctx, event.UserId, event.FriendId, event.Remark)
		if err != nil {
			return g.converter.BuildNotifyFriendEventResponse(false, err.Error()), nil
		}
	case rest.FriendEventType_DELETE_FRIEND:
		err := g.svc.DeleteFriend(ctx, event.UserId, event.FriendId)
		if err != nil {
			return g.converter.BuildNotifyFriendEventResponse(false, err.Error()), nil
		}
	default:
		return g.converter.BuildNotifyFriendEventResponse(false, "unknown event type"), nil
	}

	return g.converter.BuildNotifyFriendEventResponse(true, "ok"), nil
}
