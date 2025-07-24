package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"websocket-server/api/rest"
	"websocket-server/apps/chat-service/model"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/redis"
)

// Service Logic服务 - 业务编排层
type Service struct {
	redis         *redis.RedisClient
	kafka         *kafka.Producer
	logger        logger.Logger
	groupClient   rest.GroupServiceClient
	messageClient rest.MessageServiceClient
	friendClient  rest.FriendEventServiceClient
	userClient    rest.UserServiceClient
}

// NewService 创建Logic服务实例
func NewService(redis *redis.RedisClient, kafka *kafka.Producer, log logger.Logger, groupAddr, messageAddr, friendAddr, userAddr string) (*Service, error) {
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

	// 连接Friend服务
	friendConn, err := grpc.NewClient(friendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接Friend服务失败: %v", err)
	}
	friendClient := rest.NewFriendEventServiceClient(friendConn)

	// 连接User服务
	userConn, err := grpc.NewClient(userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接User服务失败: %v", err)
	}
	userClient := rest.NewUserServiceClient(userConn)

	service := &Service{
		redis:         redis,
		kafka:         kafka,
		logger:        log,
		groupClient:   groupClient,
		messageClient: messageClient,
		friendClient:  friendClient,
		userClient:    userClient,
	}

	return service, nil
}

// ProcessMessage 处理消息 - 核心业务编排逻辑
func (s *Service) ProcessMessage(ctx context.Context, msg *rest.WSMessage) (*model.MessageResult, error) {
	s.logger.Info(ctx, "Logic服务开始处理消息",
		logger.F("messageID", msg.MessageId),
		logger.F("from", msg.From),
		logger.F("to", msg.To),
		logger.F("groupID", msg.GroupId))

	// 1. 消息路由决策
	if msg.GroupId > 0 {
		// 群聊消息
		return s.processGroupMessage(ctx, msg)
	} else if msg.To > 0 {
		// 单聊消息
		return s.processPrivateMessage(ctx, msg)
	} else {
		return nil, fmt.Errorf("无效的消息目标")
	}
}

// processGroupMessage 处理群聊消息
func (s *Service) processGroupMessage(ctx context.Context, msg *rest.WSMessage) (*model.MessageResult, error) {
	s.logger.Info(ctx, "处理群聊消息", logger.F("groupID", msg.GroupId))

	// 1. 权限验证 - 检查用户是否在群组中
	memberResp, err := s.groupClient.GetMember(ctx, &rest.GetMemberRequest{
		GroupId: msg.GroupId,
		UserId:  msg.From,
	})
	if err != nil || !memberResp.Success {
		s.logger.Error(ctx, "用户不在群组中", logger.F("userID", msg.From), logger.F("groupID", msg.GroupId))
		return &model.MessageResult{
			Success:      false,
			Message:      "您不在该群组中",
			SuccessCount: 0,
			FailureCount: 1,
		}, nil
	}

	// 2. 获取群成员列表
	membersResp, err := s.groupClient.GetMembers(ctx, &rest.GetMembersRequest{
		GroupId: msg.GroupId,
	})
	if err != nil {
		s.logger.Error(ctx, "获取群成员失败", logger.F("error", err.Error()))
		return nil, err
	}

	// 3. 消息持久化 (异步)
	go func() {
		_, err := s.messageClient.SendWSMessage(context.Background(), &rest.SendWSMessageRequest{
			Msg: msg,
		})
		if err != nil {
			s.logger.Error(context.Background(), "消息持久化失败", logger.F("error", err.Error()))
		}
	}()

	// 4. 消息扇出 - 发送给所有群成员
	successCount := 0
	failureCount := 0
	var failedUsers []int64

	for _, member := range membersResp.Members {
		if member.UserId == msg.From {
			continue // 跳过发送者
		}

		// 通过消息队列异步投递
		err := s.publishMessageToQueue(ctx, member.UserId, msg)
		if err != nil {
			s.logger.Error(ctx, "消息投递失败",
				logger.F("targetUser", member.UserId),
				logger.F("error", err.Error()))
			failureCount++
			failedUsers = append(failedUsers, member.UserId)
		} else {
			successCount++
		}
	}

	return &model.MessageResult{
		Success:      successCount > 0,
		Message:      fmt.Sprintf("群消息发送完成，成功: %d, 失败: %d", successCount, failureCount),
		MessageID:    msg.MessageId,
		SuccessCount: successCount,
		FailureCount: failureCount,
		FailedUsers:  failedUsers,
	}, nil
}

// processPrivateMessage 处理私聊消息
func (s *Service) processPrivateMessage(ctx context.Context, msg *rest.WSMessage) (*model.MessageResult, error) {
	s.logger.Info(ctx, "处理私聊消息", logger.F("to", msg.To))

	// 1. 权限验证 - 检查好友关系
	friendResp, err := s.friendClient.GetFriend(ctx, &rest.GetFriendRequest{
		UserId:   msg.From,
		FriendId: msg.To,
	})
	if err != nil || !friendResp.Success {
		s.logger.Error(ctx, "用户不是好友关系", logger.F("from", msg.From), logger.F("to", msg.To))
		return &model.MessageResult{
			Success:      false,
			Message:      "您与对方不是好友关系",
			SuccessCount: 0,
			FailureCount: 1,
		}, nil
	}

	// 2. 消息持久化 (异步)
	go func() {
		_, err := s.messageClient.SendWSMessage(context.Background(), &rest.SendWSMessageRequest{
			Msg: msg,
		})
		if err != nil {
			s.logger.Error(context.Background(), "消息持久化失败", logger.F("error", err.Error()))
		}
	}()

	// 3. 消息投递
	err = s.publishMessageToQueue(ctx, msg.To, msg)
	if err != nil {
		s.logger.Error(ctx, "私聊消息投递失败", logger.F("error", err.Error()))
		return &model.MessageResult{
			Success:      false,
			Message:      "消息发送失败",
			SuccessCount: 0,
			FailureCount: 1,
			FailedUsers:  []int64{msg.To},
		}, nil
	}

	return &model.MessageResult{
		Success:      true,
		Message:      "私聊消息发送成功",
		MessageID:    msg.MessageId,
		SuccessCount: 1,
		FailureCount: 0,
	}, nil
}

// publishMessageToQueue 将消息发布到消息队列
func (s *Service) publishMessageToQueue(ctx context.Context, targetUserID int64, msg *rest.WSMessage) error {
	// 构造投递消息
	deliveryMsg := map[string]interface{}{
		"target_user_id": targetUserID,
		"message_id":     msg.MessageId,
		"from":           msg.From,
		"to":             msg.To,
		"group_id":       msg.GroupId,
		"content":        msg.Content,
		"message_type":   msg.MessageType,
		"timestamp":      msg.Timestamp,
		"ack_id":         msg.AckId,
	}

	// 发布到下行消息队列
	return s.kafka.Publish("downlink_messages", deliveryMsg)
}

// ValidateUserPermission 验证用户权限
func (s *Service) ValidateUserPermission(ctx context.Context, userID int64) error {
	// 调用User服务验证用户状态
	userResp, err := s.userClient.GetUser(ctx, &rest.GetUserRequest{
		UserId: fmt.Sprintf("%d", userID),
	})
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %v", err)
	}

	if !userResp.Success {
		return fmt.Errorf("用户不存在或已被禁用")
	}

	return nil
}

// GetMessageHistory 获取消息历史
func (s *Service) GetMessageHistory(ctx context.Context, userID, targetID, groupID int64, page, size int32) (*rest.GetHistoryResponse, error) {
	// 直接调用Message服务获取历史消息
	return s.messageClient.GetHistoryMessages(ctx, &rest.GetHistoryRequest{
		UserId:  userID,
		GroupId: groupID,
		Page:    page,
		Size:    size,
	})
}
