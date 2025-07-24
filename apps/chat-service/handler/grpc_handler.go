package handler

import (
	"context"
	"io"

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

// MessageStream 双向流消息处理（与Connect服务的实时通信）
func (h *GRPCHandler) MessageStream(stream rest.ChatService_MessageStreamServer) error {
	h.logger.Info(stream.Context(), "Chat服务建立双向流连接")

	// 创建错误通道
	errChan := make(chan error, 2)

	// 启动接收goroutine
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					h.logger.Info(stream.Context(), "客户端关闭连接")
				} else {
					h.logger.Error(stream.Context(), "接收消息失败", logger.F("error", err.Error()))
				}
				errChan <- err
				return
			}

			h.logger.Info(stream.Context(), "Chat服务收到流消息",
				logger.F("from", msg.From),
				logger.F("to", msg.To),
				logger.F("groupID", msg.GroupId),
				logger.F("content", msg.Content))

			// 处理消息
			result, err := h.svc.ProcessMessage(stream.Context(), msg)
			if err != nil {
				h.logger.Error(stream.Context(), "处理流消息失败",
					logger.F("error", err.Error()))

				// 发送错误响应
				errorMsg := &rest.WSMessage{
					MessageId: msg.MessageId,
					From:      -1, // 系统消息
					To:        msg.From,
					Content:   "消息处理失败: " + err.Error(),
					Timestamp: msg.Timestamp,
				}

				if sendErr := stream.Send(errorMsg); sendErr != nil {
					h.logger.Error(stream.Context(), "发送错误消息失败",
						logger.F("error", sendErr.Error()))
					errChan <- sendErr
					return
				}
				continue
			}

			// 发送成功响应（可选，根据业务需求）
			if result.Success {
				h.logger.Info(stream.Context(), "流消息处理成功",
					logger.F("messageID", result.MessageID),
					logger.F("successCount", result.SuccessCount))
			}
		}
	}()

	// 等待错误或连接关闭
	select {
	case err := <-errChan:
		h.logger.Error(stream.Context(), "双向流连接出错", logger.F("error", err.Error()))
		return err
	case <-stream.Context().Done():
		h.logger.Info(stream.Context(), "Chat服务双向流连接关闭")
		return nil
	}
}
