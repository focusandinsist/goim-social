package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/message-service/converter"
	"goim-social/apps/message-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedMessageServiceServer
	service   *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(service *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		service:   service,
		converter: converter.NewConverter(),
		logger:    log,
	}
}

// SendWSMessage 发送并持久化消息
func (g *GRPCHandler) SendWSMessage(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
	return g.sendWSMessageImpl(ctx, req)
}

// GetHistoryMessages 获取历史消息
func (g *GRPCHandler) GetHistoryMessages(ctx context.Context, req *rest.GetHistoryRequest) (*rest.GetHistoryResponse, error) {
	return g.getHistoryMessagesImpl(ctx, req)
}

// MarkMessagesAsRead 标记消息已读gRPC接口
func (g *GRPCHandler) MarkMessagesAsRead(ctx context.Context, req *rest.MarkMessagesReadRequest) (*rest.MarkMessagesReadResponse, error) {
	return g.markMessagesAsReadImpl(ctx, req)
}
