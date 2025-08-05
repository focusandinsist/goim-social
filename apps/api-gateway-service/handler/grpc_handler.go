package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/api-gateway-service/converter"
	"goim-social/apps/api-gateway-service/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
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

// OnlineStatus gRPC在线状态查询
func (h *GRPCHandler) OnlineStatus(ctx context.Context, req *rest.OnlineStatusRequest) (*rest.OnlineStatusResponse, error) {
	// 将业务信息添加到context（如果有用户ID的话）
	if len(req.UserIds) > 0 {
		ctx = tracecontext.WithUserID(ctx, req.UserIds[0])
	}

	h.log.Info(ctx, "gRPC OnlineStatus request", logger.F("userIDs", req.UserIds))

	status, err := h.svc.OnlineStatus(ctx, req.UserIds)
	if err != nil {
		h.log.Error(ctx, "gRPC OnlineStatus failed", logger.F("error", err.Error()))
		return h.converter.BuildEmptyOnlineStatusResponse(), err
	}

	return h.converter.BuildOnlineStatusResponse(status), nil
}
