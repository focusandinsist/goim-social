package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/im-gateway-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC协议处理器
type GRPCHandler struct {
	rest.UnimplementedConnectServiceServer
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

// OnlineStatus 查询在线状态
func (g *GRPCHandler) OnlineStatus(ctx context.Context, req *rest.OnlineStatusRequest) (*rest.OnlineStatusResponse, error) {
	status, err := g.svc.OnlineStatus(ctx, req.UserIds)
	if err != nil {
		g.log.Error(ctx, "gRPC OnlineStatus failed", logger.F("error", err.Error()))
		return &rest.OnlineStatusResponse{
			Status: make(map[int64]bool),
		}, err
	}
	return &rest.OnlineStatusResponse{
		Status: status,
	}, nil
}
