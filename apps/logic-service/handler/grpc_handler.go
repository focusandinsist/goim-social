package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/logic-service/converter"
	"goim-social/apps/logic-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedLogicServiceServer
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    log,
	}
}

// SendMessage 发送消息gRPC接口
func (h *GRPCHandler) SendMessage(ctx context.Context, req *rest.SendLogicMessageRequest) (*rest.SendLogicMessageResponse, error) {
	return h.sendMessageImpl(ctx, req)
}

// HandleMessageAck 处理消息ACK确认gRPC接口
func (h *GRPCHandler) HandleMessageAck(ctx context.Context, req *rest.MessageAckRequest) (*rest.MessageAckResponse, error) {
	return h.handleMessageAckImpl(ctx, req)
}
