package handler

import (
	"context"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// sendWSMessageImpl 发送并持久化消息实现
func (g *GRPCHandler) sendWSMessageImpl(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
	msg := req.Msg

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, msg.From)
	if msg.GroupId > 0 {
		ctx = tracecontext.WithGroupID(ctx, msg.GroupId)
	}

	g.logger.Info(ctx, "Message服务接收消息",
		logger.F("from", msg.From),
		logger.F("to", msg.To),
		logger.F("groupID", msg.GroupId),
		logger.F("content", msg.Content))

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
		g.logger.Error(ctx, "消息持久化失败", logger.F("error", err.Error()))
		return g.converter.BuildErrorSendWSMessageResponse("消息保存失败"), nil
	}

	g.logger.Info(ctx, "消息持久化成功", logger.F("messageID", msg.MessageId))
	return g.converter.BuildSuccessSendWSMessageResponse("消息保存成功"), nil
}

// getHistoryMessagesImpl 获取历史消息实现
func (g *GRPCHandler) getHistoryMessagesImpl(ctx context.Context, req *rest.GetHistoryRequest) (*rest.GetHistoryResponse, error) {
	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)
	if req.GroupId > 0 {
		ctx = tracecontext.WithGroupID(ctx, req.GroupId)
	}

	g.logger.Info(ctx, "获取历史消息",
		logger.F("userID", req.UserId),
		logger.F("groupID", req.GroupId),
		logger.F("page", req.Page),
		logger.F("size", req.Size))

	msgs, total, err := g.service.GetMessageHistory(ctx, req.UserId, req.GroupId, int(req.Page), int(req.Size))
	if err != nil {
		g.logger.Error(ctx, "获取历史消息失败", logger.F("error", err.Error()))
		return g.converter.BuildErrorGetHistoryResponse(req.Page, req.Size), err
	}

	g.logger.Info(ctx, "获取历史消息成功", logger.F("count", len(msgs)))
	return g.converter.BuildGetHistoryResponse(msgs, total, req.Page, req.Size), nil
}

// markMessagesAsReadImpl 标记消息已读实现
func (g *GRPCHandler) markMessagesAsReadImpl(ctx context.Context, req *rest.MarkMessagesReadRequest) (*rest.MarkMessagesReadResponse, error) {
	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	g.logger.Info(ctx, "Message服务接收标记已读请求",
		logger.F("userID", req.UserId),
		logger.F("messageIDs", req.MessageIds))

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
		g.logger.Error(ctx, "Message服务标记已读失败", logger.F("error", err.Error()))
	} else {
		g.logger.Info(ctx, "Message服务标记已读成功",
			logger.F("userID", req.UserId),
			logger.F("successCount", len(req.MessageIds)-len(failedIDs)),
			logger.F("failedCount", len(failedIDs)))
	}

	return response, nil
}
