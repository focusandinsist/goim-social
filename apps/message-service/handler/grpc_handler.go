package handler

import (
	"context"
	"log"

	"goim-social/api/rest"
	"goim-social/apps/message-service/converter"
	"goim-social/apps/message-service/service"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedMessageServiceServer
	service   *service.Service
	converter *converter.Converter
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(service *service.Service) *GRPCHandler {
	return &GRPCHandler{
		service:   service,
		converter: converter.NewConverter(),
	}
}

// SendWSMessage 发送并持久化消息
func (g *GRPCHandler) SendWSMessage(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
	msg := req.Msg
	log.Printf("Message服务接收消息: From=%d, To=%d, GroupID=%d, Content=%s",
		msg.From, msg.To, msg.GroupId, msg.Content)

	// 1. 数据验证
	if msg.From <= 0 {
		return g.converter.BuildErrorSendWSMessageResponse("发送者ID无效"), nil
	}

	if msg.To <= 0 && msg.GroupId <= 0 {
		return g.converter.BuildErrorSendWSMessageResponse("接收者或群组ID必须指定一个"), nil
	}

	if msg.Content == "" {
		return g.converter.BuildErrorSendWSMessageResponse("消息内容不能为空"), nil
	}

	// 2. 调用service层保存消息
	err := g.service.SaveWSMessage(ctx, msg)
	if err != nil {
		log.Printf("消息持久化失败: %v", err)
		return g.converter.BuildErrorSendWSMessageResponse("消息保存失败"), nil
	}

	log.Printf("消息持久化成功: MessageID=%d", msg.MessageId)
	return g.converter.BuildSuccessSendWSMessageResponse("消息保存成功"), nil
}

// GetHistoryMessages 获取历史消息
func (g *GRPCHandler) GetHistoryMessages(ctx context.Context, req *rest.GetHistoryRequest) (*rest.GetHistoryResponse, error) {
	log.Printf("获取历史消息: UserID=%d, GroupID=%d, Page=%d, Size=%d",
		req.UserId, req.GroupId, req.Page, req.Size)

	msgs, total, err := g.service.GetMessageHistory(ctx, req.UserId, req.GroupId, int(req.Page), int(req.Size))
	if err != nil {
		log.Printf("获取历史消息失败: %v", err)
		return g.converter.BuildErrorGetHistoryResponse(req.Page, req.Size), err
	}

	log.Printf("获取历史消息成功: 共%d条消息", len(msgs))
	return g.converter.BuildGetHistoryResponse(msgs, total, req.Page, req.Size), nil
}

// MarkMessagesAsRead 标记消息已读gRPC接口
func (g *GRPCHandler) MarkMessagesAsRead(ctx context.Context, req *rest.MarkMessagesReadRequest) (*rest.MarkMessagesReadResponse, error) {
	log.Printf("Message服务接收标记已读请求: UserID=%d, MessageIDs=%v", req.UserId, req.MessageIds)

	// 调用service层标记消息已读
	failedIDs, err := g.service.MarkMessagesAsRead(ctx, req.UserId, req.MessageIds)

	response := &rest.MarkMessagesReadResponse{
		Success: err == nil && len(failedIDs) == 0,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			if len(failedIDs) > 0 {
				return "部分消息标记已读失败"
			}
			return "标记已读成功"
		}(),
		FailedIds: failedIDs,
	}

	if err != nil {
		log.Printf("Message服务标记已读失败: %v", err)
	} else {
		log.Printf("Message服务标记已读成功: UserID=%d, 成功数量=%d, 失败数量=%d",
			req.UserId, len(req.MessageIds)-len(failedIDs), len(failedIDs))
	}

	return response, nil
}
