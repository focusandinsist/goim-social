package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"websocket-server/api/rest"
	"websocket-server/apps/chat-service/model"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/redis"
)

// Service 聊天服务
type Service struct {
	redis         *redis.RedisClient
	kafka         *kafka.Producer
	logger        logger.Logger
	groupClient   rest.GroupServiceClient
	messageClient rest.MessageServiceClient
	messageStream rest.MessageService_MessageStreamClient // Message服务双向流连接
}

// NewService 创建聊天服务实例
func NewService(redis *redis.RedisClient, kafka *kafka.Producer, log logger.Logger, groupAddr, messageAddr string) (*Service, error) {
	// 连接Group服务
	groupConn, err := grpc.NewClient(groupAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接Group服务失败: %v", err)
	}
	groupClient := rest.NewGroupServiceClient(groupConn)

	// 连接Message服务
	messageConn, err := grpc.NewClient(messageAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接Message服务失败: %v", err)
	}
	messageClient := rest.NewMessageServiceClient(messageConn)

	service := &Service{
		redis:         redis,
		kafka:         kafka,
		logger:        log,
		groupClient:   groupClient,
		messageClient: messageClient,
	}

	return service, nil
}

// StartMessageStream 启动与Message服务的双向流连接
func (s *Service) StartMessageStream() {
	s.logger.Info(context.Background(), "开始连接Message服务双向流...")

	// 重试连接Message服务
	for i := 0; i < 10; i++ {
		ctx := context.Background()
		if i == 0 {
			s.logger.Info(ctx, "尝试连接Message服务双向流...", logger.F("attempt", i+1))
		} else {
			s.logger.Info(ctx, "重试连接Message服务双向流...", logger.F("attempt", i+1))
		}

		if s.messageClient == nil {
			s.logger.Warn(ctx, "Message服务客户端未初始化，等待2秒后重试...")
			time.Sleep(2 * time.Second)
			continue
		}

		// 创建双向流连接
		stream, err := s.messageClient.MessageStream(ctx)
		if err != nil {
			s.logger.Error(ctx, "创建Message服务双向流失败", logger.F("error", err.Error()))
			if i < 9 {
				s.logger.Info(ctx, "等待2秒后重试...")
			}
			time.Sleep(2 * time.Second)
			continue
		}

		s.messageStream = stream // 保存stream连接
		s.logger.Info(ctx, "成功连接到Message服务双向流")

		// 发送订阅请求
		subscribeReq := &rest.MessageStreamRequest{
			RequestType: &rest.MessageStreamRequest_Subscribe{
				Subscribe: &rest.SubscribeRequest{
					ConnectServiceId: "chat-service", // Chat服务标识
				},
			},
		}

		err = stream.Send(subscribeReq)
		if err != nil {
			s.logger.Error(ctx, "发送订阅请求失败", logger.F("error", err.Error()))
			stream = nil
			time.Sleep(2 * time.Second)
			continue
		}

		// 启动接收goroutine
		go s.handleMessageStreamMessages(stream)

		s.logger.Info(ctx, "Message服务双向流连接建立完成")
		return
	}

	s.logger.Error(context.Background(), "连接Message服务双向流失败，已重试10次")
}

// handleMessageStreamMessages 处理来自Message服务的流消息
func (s *Service) handleMessageStreamMessages(stream rest.MessageService_MessageStreamClient) {
	ctx := context.Background()
	for {
		resp, err := stream.Recv()
		if err != nil {
			s.logger.Error(ctx, "接收Message服务流消息失败", logger.F("error", err.Error()))
			s.messageStream = nil // 清空连接

			// 重新连接
			go func() {
				time.Sleep(5 * time.Second)
				s.StartMessageStream()
			}()
			return
		}

		s.logger.Info(ctx, "收到Message服务流响应", logger.F("response", resp))

		// 处理来自Message服务的响应
		switch resp.ResponseType.(type) {
		case *rest.MessageStreamResponse_PushEvent:
			pushEvent := resp.GetPushEvent()
			s.logger.Info(ctx, "收到推送事件",
				logger.F("targetUserID", pushEvent.TargetUserId),
				logger.F("eventType", pushEvent.EventType),
				logger.F("messageFrom", pushEvent.Message.From))

		case *rest.MessageStreamResponse_Failure:
			failureEvent := resp.GetFailure()
			s.logger.Error(ctx, "收到消息失败事件",
				logger.F("originalSender", failureEvent.OriginalSender),
				logger.F("failureReason", failureEvent.FailureReason))

		case *rest.MessageStreamResponse_AckConfirm:
			ackConfirm := resp.GetAckConfirm()
			s.logger.Info(ctx, "收到ACK确认",
				logger.F("messageID", ackConfirm.MessageId),
				logger.F("confirmed", ackConfirm.Confirmed))

		default:
			s.logger.Info(ctx, "收到未知类型的Message服务响应")
		}
	}
}

// RouteMessage 消息路由,判断消息类型并进行相应处理
func (s *Service) RouteMessage(ctx context.Context, msg *model.ChatMessage) (*model.RouteResult, error) {
	s.logger.Info(ctx, "开始路由消息",
		logger.F("messageID", msg.MessageID),
		logger.F("from", msg.From),
		logger.F("to", msg.To),
		logger.F("groupID", msg.GroupID),
		logger.F("chatType", msg.ChatType))

	// 生成消息ID（如果没有）
	if msg.MessageID == 0 {
		msg.MessageID = s.generateMessageID()
	}

	var targetUsers []int64
	var err error

	switch msg.ChatType {
	case model.ChatTypePrivate:
		// 单聊消息路由
		targetUsers, err = s.routePrivateMessage(ctx, msg)
	case model.ChatTypeGroup:
		// 群聊消息路由（写扩散）
		targetUsers, err = s.routeGroupMessage(ctx, msg)
	default:
		return &model.RouteResult{
			Success: false,
			Message: "不支持的聊天类型",
		}, fmt.Errorf("不支持的聊天类型: %d", msg.ChatType)
	}

	if err != nil {
		s.logger.Error(ctx, "消息路由失败", logger.F("messageID", msg.MessageID), logger.F("error", err.Error()))
		return &model.RouteResult{
			Success: false,
			Message: err.Error(),
		}, err
	}

	s.logger.Info(ctx, "消息路由成功",
		logger.F("messageID", msg.MessageID),
		logger.F("targetUsers", targetUsers),
		logger.F("userCount", len(targetUsers)))

	return &model.RouteResult{
		Success:     true,
		Message:     "消息路由成功",
		MessageID:   msg.MessageID,
		TargetUsers: targetUsers,
		ChatType:    msg.ChatType,
	}, nil
}

// routePrivateMessage 单聊消息路由
func (s *Service) routePrivateMessage(ctx context.Context, msg *model.ChatMessage) ([]int64, error) {
	if msg.To <= 0 {
		return nil, fmt.Errorf("单聊消息缺少目标用户ID")
	}

	// 单聊消息直接转发给目标用户
	targetUsers := []int64{msg.To}

	// 调用Message服务存储和推送
	err := s.forwardToMessageService(ctx, msg, targetUsers)
	if err != nil {
		return nil, fmt.Errorf("转发单聊消息失败: %v", err)
	}

	return targetUsers, nil
}

// routeGroupMessage 群聊消息路由（写扩散）
func (s *Service) routeGroupMessage(ctx context.Context, msg *model.ChatMessage) ([]int64, error) {
	if msg.GroupID <= 0 {
		return nil, fmt.Errorf("群聊消息缺少群组ID")
	}

	// 1. 验证发送者是否为群成员
	validateReq := &rest.ValidateGroupMemberRequest{
		GroupId: msg.GroupID,
		UserId:  msg.From,
	}
	validateResp, err := s.groupClient.ValidateGroupMember(ctx, validateReq)
	if err != nil {
		return nil, fmt.Errorf("验证群成员身份失败: %v", err)
	}
	if !validateResp.Success || !validateResp.IsMember {
		return nil, fmt.Errorf("用户不是群成员，无权发送群消息")
	}

	// 2. 获取群成员列表
	memberReq := &rest.GetGroupMemberIDsRequest{GroupId: msg.GroupID}
	memberResp, err := s.groupClient.GetGroupMemberIDs(ctx, memberReq)
	if err != nil {
		return nil, fmt.Errorf("获取群成员列表失败: %v", err)
	}
	if !memberResp.Success {
		return nil, fmt.Errorf("获取群成员列表失败: %s", memberResp.Message)
	}

	// 3. 写扩散：为每个群成员创建个人消息副本
	targetUsers := make([]int64, 0, len(memberResp.MemberIds))
	for _, memberID := range memberResp.MemberIds {
		if memberID == msg.From {
			continue // 跳过自己
		}
		targetUsers = append(targetUsers, memberID)
	}

	if len(targetUsers) == 0 {
		s.logger.Warn(ctx, "群组没有其他成员", logger.F("groupID", msg.GroupID))
		return targetUsers, nil
	}

	// 4. 调用Message服务存储和推送
	err = s.forwardToMessageService(ctx, msg, targetUsers)
	if err != nil {
		return nil, fmt.Errorf("转发群聊消息失败: %v", err)
	}

	s.logger.Info(ctx, "群聊消息写扩散完成",
		logger.F("groupID", msg.GroupID),
		logger.F("memberCount", len(memberResp.MemberIds)),
		logger.F("targetCount", len(targetUsers)))

	return targetUsers, nil
}

// forwardToMessageService 转发消息到Message服务
func (s *Service) forwardToMessageService(ctx context.Context, msg *model.ChatMessage, targetUsers []int64) error {
	s.logger.Info(ctx, "开始转发消息到Message服务", logger.F("messageID", msg.MessageID), logger.F("targetUsers", targetUsers))

	var (
		successCount = 0
		failureCount = 0
		failedUsers  []int64
	)

	for _, targetUserID := range targetUsers {
		// 为每个目标用户创建消息副本
		wsMsg := &rest.WSMessage{
			MessageId:   msg.MessageID,
			From:        msg.From,
			To:          targetUserID,
			GroupId:     msg.GroupID,
			Content:     msg.Content,
			MessageType: msg.MessageType,
			Timestamp:   time.Now().Unix(),
		}

		// 优先使用双向流，如果不可用则使用单向调用
		err := s.sendMessageViaStream(ctx, wsMsg)
		if err != nil {
			s.logger.Error(ctx, "转发消息到Message服务失败",
				logger.F("messageID", msg.MessageID),
				logger.F("targetUser", targetUserID),
				logger.F("error", err))
			failureCount++
			failedUsers = append(failedUsers, targetUserID)
		} else {
			successCount++
		}
	}

	s.logger.Info(ctx, "消息转发完成",
		logger.F("messageID", msg.MessageID),
		logger.F("successCount", successCount),
		logger.F("failureCount", failureCount),
		logger.F("failedUsers", failedUsers))

	if failureCount > 0 {
		return fmt.Errorf("部分消息转发失败: 成功%d, 失败%d", successCount, failureCount)
	}

	return nil
}

// generateMessageID 生成消息ID
func (s *Service) generateMessageID() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// ProcessMessage 处理来自Connect服务的消息（主入口）
func (s *Service) ProcessMessage(ctx context.Context, wsMsg *rest.WSMessage) (*model.ForwardResult, error) {
	s.logger.Info(ctx, "Chat服务接收到消息",
		logger.F("from", wsMsg.From),
		logger.F("to", wsMsg.To),
		logger.F("groupID", wsMsg.GroupId),
		logger.F("content", wsMsg.Content))

	// 转换为内部消息模型
	chatMsg := &model.ChatMessage{
		MessageID:   wsMsg.MessageId,
		From:        wsMsg.From,
		To:          wsMsg.To,
		GroupID:     wsMsg.GroupId,
		Content:     wsMsg.Content,
		MessageType: wsMsg.MessageType,
		ChatType:    s.determineChatType(wsMsg),
		Status:      model.MessageStatusSending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 路由消息
	routeResult, err := s.RouteMessage(ctx, chatMsg)
	if err != nil {
		return &model.ForwardResult{
			Success: false,
			Message: err.Error(),
		}, err
	}

	return &model.ForwardResult{
		Success:      routeResult.Success,
		Message:      routeResult.Message,
		MessageID:    routeResult.MessageID,
		SuccessCount: len(routeResult.TargetUsers),
		FailureCount: 0,
		FailedUsers:  []int64{},
	}, nil
}

// determineChatType 判断聊天类型
func (s *Service) determineChatType(wsMsg *rest.WSMessage) int32 {
	if wsMsg.GroupId > 0 {
		return model.ChatTypeGroup
	}
	return model.ChatTypePrivate
}

// sendMessageViaStream 通过双向流发送消息到Message服务
func (s *Service) sendMessageViaStream(ctx context.Context, wsMsg *rest.WSMessage) error {
	// 优先使用双向流
	if s.messageStream != nil {
		s.logger.Info(ctx, "通过Message服务双向流发送消息",
			logger.F("messageID", wsMsg.MessageId),
			logger.F("from", wsMsg.From),
			logger.F("to", wsMsg.To))

		// 构造发送请求
		req := &rest.MessageStreamRequest{
			RequestType: &rest.MessageStreamRequest_SendMessage{
				SendMessage: &rest.SendWSMessageRequest{
					Msg: wsMsg,
				},
			},
		}

		err := s.messageStream.Send(req)
		if err != nil {
			s.logger.Error(ctx, "Message服务双向流发送失败", logger.F("error", err.Error()))
			s.messageStream = nil // 清空连接

			// 降级到单向调用
			return s.sendMessageViaUnaryCall(ctx, wsMsg)
		}

		s.logger.Info(ctx, "Message服务双向流发送成功")
		return nil
	}

	// 双向流不可用，使用单向调用
	return s.sendMessageViaUnaryCall(ctx, wsMsg)
}

// sendMessageViaUnaryCall 通过单向调用发送消息到Message服务（降级方案）
func (s *Service) sendMessageViaUnaryCall(ctx context.Context, wsMsg *rest.WSMessage) error {
	s.logger.Info(ctx, "通过Message服务单向调用发送消息",
		logger.F("messageID", wsMsg.MessageId),
		logger.F("from", wsMsg.From),
		logger.F("to", wsMsg.To))

	req := &rest.SendWSMessageRequest{
		Msg: wsMsg,
	}

	resp, err := s.messageClient.SendWSMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("Message服务单向调用失败: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("Message服务处理失败: %s", resp.Message)
	}

	s.logger.Info(ctx, "Message服务单向调用发送成功")
	return nil
}
