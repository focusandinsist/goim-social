package handler

import (
	"context"

	"websocket-server/api/rest"
)

// GRPCService gRPC服务实现
type GRPCService struct {
	rest.UnimplementedConnectServiceServer
	handler *Handler
}

// NewGRPCService 创建gRPC服务
func (h *Handler) NewGRPCService() *GRPCService {
	return &GRPCService{handler: h}
}

// OnlineStatus 查询在线状态
func (g *GRPCService) OnlineStatus(ctx context.Context, req *rest.OnlineStatusRequest) (*rest.OnlineStatusResponse, error) {
	status, err := g.handler.service.OnlineStatus(ctx, req.UserIds)
	if err != nil {
		return &rest.OnlineStatusResponse{
			Status: make(map[int64]bool),
		}, err
	}
	return &rest.OnlineStatusResponse{
		Status: status,
	}, nil
}
