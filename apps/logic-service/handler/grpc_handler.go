package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/logic-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedLogicServiceServer
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
func (h *GRPCHandler) SendMessage(ctx context.Context, req *rest.SendLogicMessageRequest) (*rest.SendLogicMessageResponse, error) {
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
		return &rest.SendLogicMessageResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	h.logger.Info(ctx, "gRPC消息处理成功",
		logger.F("messageID", result.MessageID),
		logger.F("successCount", result.SuccessCount))

	return &rest.SendLogicMessageResponse{
		Success:      result.Success,
		Message:      result.Message,
		MessageId:    result.MessageID,
		SuccessCount: int32(result.SuccessCount),
		FailureCount: int32(result.FailureCount),
		FailedUsers:  result.FailedUsers,
	}, nil
}

// HandleMessageAck 处理消息ACK确认gRPC接口
func (h *GRPCHandler) HandleMessageAck(ctx context.Context, req *rest.MessageAckRequest) (*rest.MessageAckResponse, error) {
	h.logger.Info(ctx, "收到gRPC消息ACK请求",
		logger.F("userID", req.UserId),
		logger.F("messageID", req.MessageId),
		logger.F("ackID", req.AckId))

	// 处理ACK
	err := h.svc.HandleMessageAck(ctx, req.UserId, req.MessageId, req.AckId)
	if err != nil {
		h.logger.Error(ctx, "gRPC处理消息ACK失败",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("messageID", req.MessageId))
		return &rest.MessageAckResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	h.logger.Info(ctx, "gRPC消息ACK处理成功",
		logger.F("userID", req.UserId),
		logger.F("messageID", req.MessageId))

	return &rest.MessageAckResponse{
		Success: true,
		Message: "ACK处理成功",
	}, nil
}
