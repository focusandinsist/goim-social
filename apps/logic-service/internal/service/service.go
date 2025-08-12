package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"goim-social/api/rest"
	"goim-social/apps/logic-service/internal/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/sessionlocator"
	"goim-social/pkg/snowflake"
	"goim-social/pkg/telemetry"
)

// Service Logic服务 - 业务编排层
type Service struct {
	redis          *redis.RedisClient
	kafka          *kafka.Producer         // 异步Producer（用于推送）
	reliableKafka  *kafka.ReliableProducer // 同步Producer（用于持久化保障）
	logger         logger.Logger
	instanceID     string                  // 服务实例ID
	sessionLocator *sessionlocator.Locator // 会话定位器
	gatewayCleaner *sessionlocator.Cleaner // 网关清理器（领导者选举）
	socialClient   rest.SocialServiceClient
	messageClient  rest.MessageServiceClient
	userClient     rest.UserServiceClient
}

// NewService 创建Logic服务实例
func NewService(redis *redis.RedisClient, kafkaProducer *kafka.Producer, log logger.Logger, kafkaBrokers []string, socialAddr, messageAddr, userAddr string) (*Service, error) {
	// 初始化高可靠性同步Producer（用于持久化保障）
	reliableKafka, err := kafka.InitReliableProducer(kafkaBrokers)
	if err != nil {
		return nil, fmt.Errorf("初始化可靠Kafka Producer失败: %v", err)
	}
	// 连接Social服务（合并了原来的Group和Friend服务）
	socialConn, err := grpc.NewClient(socialAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接Social服务失败: %v", err)
	}
	socialClient := rest.NewSocialServiceClient(socialConn)

	// 连接Message服务
	messageConn, err := grpc.NewClient(messageAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接Message服务失败: %v", err)
	}
	messageClient := rest.NewMessageServiceClient(messageConn)

	// 连接User服务
	userConn, err := grpc.NewClient(userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接User服务失败: %v", err)
	}
	userClient := rest.NewUserServiceClient(userConn)

	// 生成服务实例ID
	instanceID := fmt.Sprintf("logic-service-%d", time.Now().UnixNano())

	// 初始化会话定位器
	sessionLocator := sessionlocator.NewLocator(redis)

	// 初始化网关清理器（领导者选举）
	gatewayCleaner := sessionlocator.NewCleaner(redis, instanceID)

	service := &Service{
		redis:          redis,
		kafka:          kafkaProducer,
		reliableKafka:  reliableKafka,
		logger:         log,
		instanceID:     instanceID,
		sessionLocator: sessionLocator,
		gatewayCleaner: gatewayCleaner,
		socialClient:   socialClient,
		messageClient:  messageClient,
		userClient:     userClient,
	}

	// 启动网关清理器（包含领导者选举）
	gatewayCleaner.Start(context.Background())

	return service, nil
}

// ProcessMessage 处理消息 - 核心业务编排逻辑
func (s *Service) ProcessMessage(ctx context.Context, msg *rest.WSMessage) (*model.MessageResult, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "logic.service.ProcessMessage")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("message.id", msg.MessageId),
		attribute.Int64("message.from", msg.From),
		attribute.Int64("message.to", msg.To),
		attribute.Int64("message.group_id", msg.GroupId),
		attribute.String("message.content", msg.Content),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, msg.From)
	if msg.GroupId > 0 {
		ctx = tracecontext.WithGroupID(ctx, msg.GroupId)
	}

	// 如果MessageID为0，使用Snowflake生成新的MessageID
	if msg.MessageId == 0 {
		msg.MessageId = snowflake.GenerateID()
		span.SetAttributes(attribute.Int64("message.generated_id", msg.MessageId))
	}

	s.logger.Info(ctx, "Logic服务开始处理消息",
		logger.F("messageID", msg.MessageId),
		logger.F("from", msg.From),
		logger.F("to", msg.To),
		logger.F("groupID", msg.GroupId))

	// 1. 消息路由决策
	var result *model.MessageResult
	var err error

	if msg.GroupId > 0 {
		// 群聊消息
		span.SetAttributes(attribute.String("message.type", "group"))
		result, err = s.processGroupMessage(ctx, msg)
	} else if msg.To > 0 {
		// 单聊消息
		span.SetAttributes(attribute.String("message.type", "private"))
		result, err = s.processPrivateMessage(ctx, msg)
	} else {
		span.SetStatus(codes.Error, "invalid message target")
		return nil, fmt.Errorf("无效的消息目标")
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to process message")
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("result.success_count", result.SuccessCount),
		attribute.Int("result.failure_count", result.FailureCount),
	)
	span.SetStatus(codes.Ok, "message processed successfully")
	return result, nil
}

// processGroupMessage 处理群聊消息
func (s *Service) processGroupMessage(ctx context.Context, msg *rest.WSMessage) (*model.MessageResult, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "logic.service.processGroupMessage")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("group.id", msg.GroupId),
		attribute.Int64("message.from", msg.From),
	)

	s.logger.Info(ctx, "处理群聊消息", logger.F("groupID", msg.GroupId))

	// 1. 权限验证 - 检查用户是否在群组中
	memberResp, err := s.socialClient.ValidateGroupMember(ctx, &rest.ValidateGroupMemberRequest{
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
	membersResp, err := s.socialClient.GetGroupMemberIDs(ctx, &rest.GetGroupMemberIDsRequest{
		GroupId: msg.GroupId,
	})
	if err != nil {
		s.logger.Error(ctx, "获取群成员失败", logger.F("error", err.Error()))
		return nil, err
	}

	// 3. 消息持久化保障 - 同步写入Kafka确保安全落地
	if err := s.ensureMessagePersistence(ctx, msg); err != nil {
		s.logger.Error(ctx, "消息持久化保障失败",
			logger.F("messageID", msg.MessageId),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("消息持久化失败: %v", err)
	}

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
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "logic.service.processPrivateMessage")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("message.to", msg.To),
		attribute.Int64("message.from", msg.From),
	)

	s.logger.Info(ctx, "处理私聊消息", logger.F("to", msg.To))

	// 1. 权限验证 - 检查好友关系
	friendResp, err := s.socialClient.ValidateFriendship(ctx, &rest.ValidateFriendshipRequest{
		UserId:   msg.From,
		FriendId: msg.To,
	})
	if err != nil || !friendResp.Success || !friendResp.IsFriend {
		s.logger.Error(ctx, "用户不是好友关系", logger.F("from", msg.From), logger.F("to", msg.To))
		return &model.MessageResult{
			Success:      false,
			Message:      "您与对方不是好友关系",
			SuccessCount: 0,
			FailureCount: 1,
		}, nil
	}
	s.logger.Info(ctx, "好友关系验证通过", logger.F("from", msg.From), logger.F("to", msg.To))

	// 2. 消息持久化保障 - 同步写入Kafka确保安全落地
	if err := s.ensureMessagePersistence(ctx, msg); err != nil {
		s.logger.Error(ctx, "消息持久化保障失败",
			logger.F("messageID", msg.MessageId),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("消息持久化失败: %v", err)
	}

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

	// 使用会话定位器找到用户对应的网关实例
	userIDStr := fmt.Sprintf("%d", targetUserID)
	gateway, err := s.sessionLocator.GetGatewayForUser(userIDStr)
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
func (s *Service) forwardMessageToGateway(ctx context.Context, gateway *sessionlocator.GatewayInstance, msg *rest.WSMessage) error {
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

// GetSessionLocatorStatus 获取会话定位器状态信息
func (s *Service) GetSessionLocatorStatus() map[string]interface{} {
	return s.sessionLocator.GetStats()
}

// GetActiveGateways 获取所有活跃网关实例
func (s *Service) GetActiveGateways() []*sessionlocator.GatewayInstance {
	return s.sessionLocator.GetAllActiveGateways()
}

// GetUserGateway 获取用户对应的网关实例
func (s *Service) GetUserGateway(userID int64) (*sessionlocator.GatewayInstance, error) {
	userIDStr := fmt.Sprintf("%d", userID)
	return s.sessionLocator.GetGatewayForUser(userIDStr)
}

// Cleanup 清理服务资源
func (s *Service) Cleanup() {
	s.logger.Info(context.Background(), "开始清理Logic服务资源")

	// 停止网关清理器
	if s.gatewayCleaner != nil {
		s.gatewayCleaner.Stop()
	}

	// 停止会话定位器
	if s.sessionLocator != nil {
		s.sessionLocator.Stop()
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
		UserId: userID,
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
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "logic.service.GetMessageHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("user.id", userID),
		attribute.Int64("target.id", targetID),
		attribute.Int64("group.id", groupID),
		attribute.Int("page", int(page)),
		attribute.Int("size", int(size)),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)
	if groupID > 0 {
		ctx = tracecontext.WithGroupID(ctx, groupID)
	}

	// 直接调用Message服务获取历史消息
	resp, err := s.messageClient.GetHistoryMessages(ctx, &rest.GetHistoryRequest{
		UserId:  userID,
		GroupId: groupID,
		Page:    page,
		Size:    size,
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get message history")
		return nil, err
	}

	span.SetAttributes(attribute.Int("result.message_count", len(resp.Messages)))
	span.SetStatus(codes.Ok, "message history retrieved successfully")
	return resp, nil
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

// ensureMessagePersistence 确保消息持久化, 同步写入专门的持久化Topic
func (s *Service) ensureMessagePersistence(ctx context.Context, msg *rest.WSMessage) error {
	s.logger.Info(ctx, "开始消息持久化保障",
		logger.F("messageID", msg.MessageId),
		logger.F("from", msg.From),
		logger.F("to", msg.To),
		logger.F("groupID", msg.GroupId))

	// 构造持久化归档命令
	persistenceCommand := &rest.MessageEvent{
		Type:      "archive_message", // 归档命令类型
		Message:   msg,
		Timestamp: time.Now().Unix(),
	}

	// 使用高可靠性同步Producer写入专门的持久化Topic
	// 这个Topic有独立的配置：更高副本数、更长保留期、独立监控
	if err := s.reliableKafka.PublishMessageSync("message_persistence_log", persistenceCommand); err != nil {
		s.logger.Error(ctx, "消息持久化保障失败",
			logger.F("messageID", msg.MessageId),
			logger.F("topic", "message_persistence_log"),
			logger.F("error", err.Error()))
		return fmt.Errorf("消息持久化保障失败: %v", err)
	}

	s.logger.Info(ctx, "消息归档命令已安全写入持久化Topic（已确认）",
		logger.F("messageID", msg.MessageId),
		logger.F("topic", "message_persistence_log"))

	return nil
}
