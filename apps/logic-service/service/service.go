package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"websocket-server/api/rest"
	"websocket-server/apps/logic-service/model"
	"websocket-server/pkg/gatewayrouter"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/redis"
	"websocket-server/pkg/snowflake"
)

// Service Logic服务 - 业务编排层
type Service struct {
	redis          *redis.RedisClient
	kafka          *kafka.Producer
	logger         logger.Logger
	instanceID     string                 // 服务实例ID
	gatewayRouter  *gatewayrouter.Router  // 网关路由器
	gatewayCleaner *gatewayrouter.Cleaner // 网关清理器（领导者选举）
	groupClient    rest.GroupServiceClient
	messageClient  rest.MessageServiceClient
	friendClient   rest.FriendEventServiceClient
	userClient     rest.UserServiceClient
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

	// 生成服务实例ID
	instanceID := fmt.Sprintf("logic-service-%d", time.Now().UnixNano())

	// 初始化网关路由器
	gatewayRouter := gatewayrouter.NewRouter(redis)

	// 初始化网关清理器（领导者选举）
	gatewayCleaner := gatewayrouter.NewCleaner(redis, instanceID)

	service := &Service{
		redis:          redis,
		kafka:          kafka,
		logger:         log,
		instanceID:     instanceID,
		gatewayRouter:  gatewayRouter,
		gatewayCleaner: gatewayCleaner,
		groupClient:    groupClient,
		messageClient:  messageClient,
		friendClient:   friendClient,
		userClient:     userClient,
	}

	// 启动网关清理器（包含领导者选举）
	gatewayCleaner.Start(context.Background())

	return service, nil
}

// ProcessMessage 处理消息 - 核心业务编排逻辑
func (s *Service) ProcessMessage(ctx context.Context, msg *rest.WSMessage) (*model.MessageResult, error) {
	// 如果MessageID为0，使用Snowflake生成新的MessageID
	if msg.MessageId == 0 {
		msg.MessageId = snowflake.GenerateID()
	}

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
	memberResp, err := s.groupClient.ValidateGroupMember(ctx, &rest.ValidateGroupMemberRequest{
		GroupId: msg.GroupId,
		UserId:  msg.From,
	})
	if err != nil || !memberResp.Success || !memberResp.IsMember {
		s.logger.Error(ctx, "用户不在群组中", logger.F("userID", msg.From), logger.F("groupID", msg.GroupId))
		return &model.MessageResult{
			Success:      false,
			Message:      "您不在该群组中",
			SuccessCount: 0,
			FailureCount: 1,
		}, nil
	}

	// 2. 获取群成员列表
	membersResp, err := s.groupClient.GetGroupMemberIDs(ctx, &rest.GetGroupMemberIDsRequest{
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

	for _, memberID := range membersResp.MemberIds {
		if memberID == msg.From {
			continue // 跳过发送者
		}

		// 通过消息队列异步投递
		err := s.publishMessageToQueue(ctx, memberID, msg)
		if err != nil {
			s.logger.Error(ctx, "消息投递失败",
				logger.F("targetUser", memberID),
				logger.F("error", err.Error()))
			failureCount++
			failedUsers = append(failedUsers, memberID)
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
	// TODO: 实现好友关系验证，暂时跳过验证
	s.logger.Info(ctx, "处理私聊消息，暂时跳过好友关系验证", logger.F("from", msg.From), logger.F("to", msg.To))

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
	err := s.publishMessageToQueue(ctx, msg.To, msg)
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
	// 为群聊消息创建针对特定用户的消息副本
	targetMsg := &rest.WSMessage{
		MessageId:   msg.MessageId,
		From:        msg.From,
		To:          targetUserID,
		GroupId:     msg.GroupId,
		Content:     msg.Content,
		MessageType: msg.MessageType,
		Timestamp:   msg.Timestamp,
		AckId:       msg.AckId,
	}

	// 使用网关路由器找到用户对应的网关实例
	userIDStr := fmt.Sprintf("%d", targetUserID)
	gateway, err := s.gatewayRouter.GetGatewayForUser(userIDStr)
	if err != nil {
		s.logger.Error(ctx, "获取用户网关失败",
			logger.F("userID", targetUserID),
			logger.F("error", err.Error()))
		// 降级：发布到消息队列作为备选方案
		return s.publishToKafkaFallback(ctx, targetMsg)
	}

	s.logger.Info(ctx, "路由消息到网关",
		logger.F("userID", targetUserID),
		logger.F("gatewayID", gateway.ID),
		logger.F("gatewayAddr", gateway.GetAddress()))

	// 直接通过Redis发送消息到特定网关
	return s.forwardMessageToGateway(ctx, gateway, targetMsg)
}

// forwardMessageToGateway 向特定网关转发消息
func (s *Service) forwardMessageToGateway(ctx context.Context, gateway *gatewayrouter.GatewayInstance, msg *rest.WSMessage) error {
	// 通过Redis发布消息到特定网关的频道
	channel := fmt.Sprintf("gateway:%s:user_message", gateway.ID)

	// 构造protobuf消息负载
	gatewayMsg := &rest.GatewayMessage{
		Type:       "user_message",
		Message:    msg,
		TargetUser: msg.To,
		Timestamp:  time.Now().Unix(),
	}

	// 序列化为protobuf二进制数据
	payloadBytes, err := proto.Marshal(gatewayMsg)
	if err != nil {
		s.logger.Error(ctx, "序列化protobuf消息失败",
			logger.F("gatewayID", gateway.ID),
			logger.F("userID", msg.To),
			logger.F("error", err.Error()))
		return err
	}

	// 使用base64编码便于Redis传输
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadBytes)

	// 发布到Redis频道
	err = s.redis.Publish(ctx, channel, payloadBase64)
	if err != nil {
		s.logger.Error(ctx, "发送消息到网关失败",
			logger.F("gatewayID", gateway.ID),
			logger.F("userID", msg.To),
			logger.F("error", err.Error()))
		return err
	}

	s.logger.Info(ctx, "消息已发送到网关",
		logger.F("gatewayID", gateway.ID),
		logger.F("userID", msg.To),
		logger.F("messageID", msg.MessageId))

	return nil
}

// publishToKafkaFallback 降级到Kafka队列的备选方案
func (s *Service) publishToKafkaFallback(ctx context.Context, msg *rest.WSMessage) error {
	s.logger.Warn(ctx, "使用Kafka降级方案发送消息",
		logger.F("userID", msg.To),
		logger.F("messageID", msg.MessageId))

	// 构造protobuf消息事件
	messageEvent := &rest.MessageEvent{
		Type:      "new_message",
		Message:   msg,
		Timestamp: time.Now().Unix(),
	}

	// 发布到下行消息队列
	return s.kafka.PublishMessage("downlink_messages", messageEvent)
}

// GetGatewayRouterStatus 获取网关路由器状态信息
func (s *Service) GetGatewayRouterStatus() map[string]interface{} {
	return s.gatewayRouter.GetStats()
}

// GetActiveGateways 获取所有活跃网关实例
func (s *Service) GetActiveGateways() []*gatewayrouter.GatewayInstance {
	return s.gatewayRouter.GetAllActiveGateways()
}

// GetUserGateway 获取用户对应的网关实例
func (s *Service) GetUserGateway(userID int64) (*gatewayrouter.GatewayInstance, error) {
	userIDStr := fmt.Sprintf("%d", userID)
	return s.gatewayRouter.GetGatewayForUser(userIDStr)
}

// Cleanup 清理服务资源
func (s *Service) Cleanup() {
	s.logger.Info(context.Background(), "开始清理Logic服务资源")

	// 停止网关清理器
	if s.gatewayCleaner != nil {
		s.gatewayCleaner.Stop()
	}

	// 停止网关路由器
	if s.gatewayRouter != nil {
		s.gatewayRouter.Stop()
	}

	s.logger.Info(context.Background(), "Logic服务资源清理完成")
}

// GetCleanerStatus 获取清理器状态
func (s *Service) GetCleanerStatus(ctx context.Context) map[string]interface{} {
	isLeader := s.gatewayCleaner.IsLeader()

	leaderInfo, err := s.gatewayCleaner.GetLeaderInfo(ctx)
	if err != nil {
		leaderInfo = "unknown"
	}

	return map[string]interface{}{
		"instance_id":    s.instanceID,
		"is_leader":      isLeader,
		"current_leader": leaderInfo,
	}
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

// HandleMessageAck 处理消息ACK确认
func (s *Service) HandleMessageAck(ctx context.Context, userID, messageID int64, ackID string) error {
	s.logger.Info(ctx, "Logic服务处理消息ACK",
		logger.F("userID", userID),
		logger.F("messageID", messageID),
		logger.F("ackID", ackID))

	// 1. 业务验证：检查用户是否有权限ACK这条消息
	if err := s.validateAckPermission(ctx, userID, messageID); err != nil {
		s.logger.Error(ctx, "ACK权限验证失败",
			logger.F("error", err.Error()),
			logger.F("userID", userID),
			logger.F("messageID", messageID))
		return fmt.Errorf("ACK权限验证失败: %v", err)
	}

	// 2. 调用Message服务标记消息已读
	markReq := &rest.MarkMessagesReadRequest{
		UserId:     userID,
		MessageIds: []int64{messageID},
	}

	s.logger.Info(ctx, "调用Message服务标记已读",
		logger.F("userID", userID),
		logger.F("messageID", messageID))

	resp, err := s.messageClient.MarkMessagesAsRead(ctx, markReq)
	if err != nil {
		s.logger.Error(ctx, "调用Message服务标记已读失败",
			logger.F("error", err.Error()),
			logger.F("userID", userID),
			logger.F("messageID", messageID))
		return fmt.Errorf("标记消息已读失败: %v", err)
	}

	if !resp.Success {
		s.logger.Error(ctx, "Message服务标记已读失败",
			logger.F("message", resp.Message),
			logger.F("userID", userID),
			logger.F("messageID", messageID))
		return fmt.Errorf("标记消息已读失败: %s", resp.Message)
	}

	// 3. 可能的扩展：发送已读回执通知给发送方
	// TODO: 实现已读回执通知功能
	// s.sendReadReceiptNotification(ctx, messageID, userID)

	s.logger.Info(ctx, "消息ACK处理成功",
		logger.F("userID", userID),
		logger.F("messageID", messageID))

	return nil
}

// validateAckPermission 验证用户是否有权限ACK指定消息
func (s *Service) validateAckPermission(ctx context.Context, userID, messageID int64) error {
	// TODO: 实现更完善的权限验证逻辑
	// 1. 检查消息是否存在
	// 2. 检查用户是否是消息的接收方（单聊）或群成员（群聊）
	// 3. 检查消息是否已经被ACK过

	// 目前简单验证参数有效性
	if userID <= 0 {
		return fmt.Errorf("无效的用户ID: %d", userID)
	}

	if messageID <= 0 {
		return fmt.Errorf("无效的消息ID: %d", messageID)
	}

	s.logger.Debug(ctx, "ACK权限验证通过",
		logger.F("userID", userID),
		logger.F("messageID", messageID))

	return nil
}
