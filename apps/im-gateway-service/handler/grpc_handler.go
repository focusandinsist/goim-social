package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/im-gateway-service/converter"
	"goim-social/apps/im-gateway-service/service"
	tracecontext "goim-social/pkg/context"
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
	userIDs := g.converter.OnlineStatusParamsFromProto(req)

	// 将业务信息添加到context（如果有用户ID的话）
	if len(userIDs) > 0 {
		ctx = tracecontext.WithUserID(ctx, userIDs[0]) // 使用第一个用户ID作为主要用户
	}

	status, err := g.svc.OnlineStatus(ctx, userIDs)
	if err != nil {
		g.log.Error(ctx, "gRPC OnlineStatus failed", logger.F("error", err.Error()))
		return g.converter.BuildErrorOnlineStatusResponse(), err
	}
	return g.converter.BuildSuccessOnlineStatusResponse(status), nil
}
