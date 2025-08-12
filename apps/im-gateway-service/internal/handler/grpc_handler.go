package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/im-gateway-service/internal/converter"
	"goim-social/apps/im-gateway-service/internal/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC协议处理器
type GRPCHandler struct {
	rest.UnimplementedConnectServiceServer
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

// OnlineStatus 查询在线状态
func (g *GRPCHandler) OnlineStatus(ctx context.Context, req *rest.OnlineStatusRequest) (*rest.OnlineStatusResponse, error) {
	return g.onlineStatusImpl(ctx, req)
}
