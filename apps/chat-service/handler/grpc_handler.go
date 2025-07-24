package handler

import (
	"context"

	"websocket-server/api/rest"
	"websocket-server/apps/chat-service/service"
	"websocket-server/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedChatServiceServer
	svc    *service.Service
	logger logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:    svc,
		logger: log,
	}
}

// SendMessage 发送消息gRPC接口
func (h *GRPCHandler) SendMessage(ctx context.Context, req *rest.SendChatMessageRequest) (*rest.SendChatMessageResponse, error) {
	h.logger.Info(ctx, "收到gRPC发送消息请求",
		logger.F("from", req.Msg.From),
		logger.F("to", req.Msg.To),
		logger.F("groupID", req.Msg.GroupId),
		logger.F("content", req.Msg.Content))

	// 处理消息
	result, err := h.svc.ProcessMessage(ctx, req.Msg)
	if err != nil {
		h.logger.Error(ctx, "gRPC处理消息失败",
			logger.F("error", err.Error()))
		return &rest.SendChatMessageResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	h.logger.Info(ctx, "gRPC消息处理成功",
		logger.F("messageID", result.MessageID),
		logger.F("successCount", result.SuccessCount))

	return &rest.SendChatMessageResponse{
		Success:      result.Success,
		Message:      result.Message,
		MessageId:    result.MessageID,
		SuccessCount: int32(result.SuccessCount),
		FailureCount: int32(result.FailureCount),
		FailedUsers:  result.FailedUsers,
	}, nil
}
