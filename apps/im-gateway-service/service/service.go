package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"websocket-server/api/rest"
	"websocket-server/apps/im-gateway-service/model"
	"websocket-server/pkg/auth"
	"websocket-server/pkg/config"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"
)

// ConnectionManager 连接管理器，封装本地WebSocket连接和Redis状态
type ConnectionManager struct {
	localConnections map[int64]*websocket.Conn // 本地WebSocket连接
	redis            *redis.RedisClient        // Redis客户端
	config           *config.Config            // 配置
	mutex            sync.RWMutex              // 读写锁
}

// 创建连接管理器
func NewConnectionManager(redis *redis.RedisClient, cfg *config.Config) *ConnectionManager {
	return &ConnectionManager{
		localConnections: make(map[int64]*websocket.Conn),
		redis:            redis,
		config:           cfg,
	}
}

// AddConnection 原子式添加连接，同时更新本地连接和Redis状态
func (cm *ConnectionManager) AddConnection(ctx context.Context, userID int64, conn *websocket.Conn, connID string, serverID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 检查是否已存在连接，如果有则关闭旧连接
	if existingConn, exists := cm.localConnections[userID]; exists {
		log.Printf("用户 %d 已有WebSocket连接，将替换旧连接", userID)
		existingConn.Close()
	}

	// 添加到本地连接管理
	cm.localConnections[userID] = conn

	// 添加到在线用户集合
	if err := cm.redis.SAdd(ctx, "online_users", userID); err != nil {
		delete(cm.localConnections, userID)
		return fmt.Errorf("添加Redis在线状态失败: %v", err)
	}

	// 添加连接信息到Redis Hash
	key := fmt.Sprintf("conn:%d:%s", userID, connID)
	connInfo := map[string]interface{}{
		"userID":        userID,
		"connID":        connID,
		"serverID":      serverID,
		"clientType":    cm.getDefaultClientType(),
		"timestamp":     time.Now().Unix(),
		"lastHeartbeat": time.Now().Unix(),
	}

	if err := cm.redis.HMSet(ctx, key, connInfo); err != nil {
		delete(cm.localConnections, userID)
		cm.redis.SRem(ctx, "online_users", userID)
		return fmt.Errorf("添加Redis连接信息失败: %v", err)
	}

	// 设置连接过期时间
	expireTime := time.Duration(cm.config.Connect.Connection.ExpireTime) * time.Hour
	if err := cm.redis.Expire(ctx, key, expireTime); err != nil {
		log.Printf("设置连接过期时间失败: %v", err)
	}

	totalConnections := len(cm.localConnections)
	log.Printf("用户 %d 连接已添加，当前总连接数: %d", userID, totalConnections)
	return nil
}

// getDefaultClientType 获取默认客户端类型
func (cm *ConnectionManager) getDefaultClientType() string {
	return cm.config.Connect.Connection.ClientType
}

// RemoveConnection 原子式移除连接，同时清理本地连接和Redis状态
func (cm *ConnectionManager) RemoveConnection(ctx context.Context, userID int64, connID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 从本地连接管理中移除并关闭连接
	if conn, exists := cm.localConnections[userID]; exists {
		conn.Close()
		delete(cm.localConnections, userID)
		log.Printf("用户 %d 的本地WebSocket连接已关闭并移除", userID)
	}

	// 从Redis在线用户集合中移除
	if err := cm.redis.SRem(ctx, "online_users", userID); err != nil {
		log.Printf("从Redis移除用户 %d 在线状态失败: %v", userID, err)
	} else {
		log.Printf("用户 %d 已从Redis在线用户列表中移除", userID)
	}

	// 删除Redis中的连接信息
	if connID != "" {
		key := fmt.Sprintf("conn:%d:%s", userID, connID)
		if err := cm.redis.Del(ctx, key); err != nil {
			log.Printf("删除Redis连接信息失败: %v", err)
		} else {
			log.Printf("用户 %d 的Redis连接信息已删除", userID)
		}
	}

	totalConnections := len(cm.localConnections)
	log.Printf("用户 %d 连接已完全清理，剩余连接数: %d", userID, totalConnections)
	return nil
}

// 获取本地连接
func (cm *ConnectionManager) GetConnection(userID int64) (*websocket.Conn, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	conn, exists := cm.localConnections[userID]
	return conn, exists
}

// 检查用户是否在线（检查Redis状态）
func (cm *ConnectionManager) IsUserOnline(ctx context.Context, userID int64) (bool, error) {
	return cm.redis.SIsMember(ctx, "online_users", userID)
}

// 获取所有在线用户
func (cm *ConnectionManager) GetOnlineUsers(ctx context.Context) ([]int64, error) {
	members, err := cm.redis.SMembers(ctx, "online_users")
	if err != nil {
		return nil, err
	}

	var userIDs []int64
	for _, member := range members {
		if userID, err := strconv.ParseInt(member, 10, 64); err == nil {
			userIDs = append(userIDs, userID)
		}
	}
	return userIDs, nil
}

// 获取连接统计信息
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return map[string]interface{}{
		"local_connections": len(cm.localConnections),
		"connection_list":   cm.getConnectionList(),
	}
}

// 获取连接列表（用于调试）
func (cm *ConnectionManager) getConnectionList() []int64 {
	var users []int64
	for userID := range cm.localConnections {
		users = append(users, userID)
	}
	return users
}

// CleanupAll 清理所有本地连接（服务关闭时调用）
func (cm *ConnectionManager) CleanupAll() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	log.Printf("🧹 开始清理所有本地WebSocket连接...")

	// 关闭所有连接
	for userID, conn := range cm.localConnections {
		if conn != nil {
			conn.Close()
			log.Printf("已关闭用户 %d 的WebSocket连接", userID)
		}
	}

	// 清空连接map
	cm.localConnections = make(map[int64]*websocket.Conn)

	log.Printf("所有本地连接已清理完成")
}

type Service struct {
	db          *database.MongoDB
	redis       *redis.RedisClient
	kafka       *kafka.Producer
	config      *config.Config          // 配置
	instanceID  string                  // Connect服务实例ID
	logicClient rest.LogicServiceClient // Logic服务客户端
	connMgr     *ConnectionManager      // 统一连接管理器
}

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer, cfg *config.Config) *Service {
	service := &Service{
		db:         db,
		redis:      redis,
		kafka:      kafka,
		config:     cfg,
		instanceID: fmt.Sprintf("im-gateway-%d", time.Now().UnixNano()), // 生成唯一实例ID
		connMgr:    NewConnectionManager(redis, cfg),                    // 初始化连接管理器
	}

	// 初始化Logic服务客户端
	if err := service.initLogicClient(); err != nil {
		log.Printf("Logic服务客户端初始化失败: %v", err)
	}

	// 注册服务实例
	if err := service.registerInstance(); err != nil {
		log.Printf("服务实例注册失败: %v", err)
	}

	// 启动时清理旧的连接数据
	go service.cleanupOnStartup()

	// 启动Redis订阅 connect_forward 频道
	go service.subscribeConnectForward()

	return service
}

// initLogicClient 初始化Logic服务客户端
func (s *Service) initLogicClient() error {
	// Logic服务地址
	logicAddr := fmt.Sprintf("%s:%d", s.config.Connect.LogicService.Host, s.config.Connect.LogicService.Port)

	// 建立gRPC连接
	conn, err := grpc.Dial(logicAddr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("连接Logic服务失败: %v", err)
	}

	// 创建Logic服务客户端
	s.logicClient = rest.NewLogicServiceClient(conn)

	log.Printf("Logic服务客户端初始化成功，地址: %s", logicAddr)
	return nil
}

// cleanupOnStartup 启动时清理本实例的旧连接数据
func (s *Service) cleanupOnStartup() {
	ctx := context.Background()

	// 查找并清理本实例的连接数据
	pattern := "conn:*"
	keys, err := s.redis.Keys(ctx, pattern)
	if err != nil {
		log.Printf("查询连接keys失败: %v", err)
		return
	}

	cleanedCount := 0
	for _, key := range keys {
		// 获取连接信息
		connInfo, err := s.redis.HGetAll(ctx, key)
		if err != nil {
			continue
		}

		// 检查是否是本实例的连接
		if serverID, exists := connInfo["serverID"]; exists && serverID == s.instanceID {
			// 删除连接信息
			if err := s.redis.Del(ctx, key); err == nil {
				cleanedCount++
			}

			// 从在线用户集合中移除
			if userIDStr, exists := connInfo["userID"]; exists {
				s.redis.SRem(ctx, "online_users", userIDStr)
			}
		}
	}

	log.Printf("启动时清理完成: 清理了 %d 个旧连接", cleanedCount)
}

// setupGracefulShutdown 设置优雅退出
func (s *Service) setupGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("收到退出信号，开始优雅关闭...")

	s.cleanup()
	os.Exit(0)
}

// cleanup 清理资源
func (s *Service) cleanup() {
	ctx := context.Background()

	log.Printf("开始清理实例资源: %s", s.instanceID)

	// 清理实例注册信息
	instanceKey := fmt.Sprintf("connect_instances:%s", s.instanceID)
	if err := s.redis.Del(ctx, instanceKey); err != nil {
		log.Printf("清理实例信息失败: %v", err)
	}

	// 2. 清理本实例的所有连接
	pattern := "conn:*"
	keys, err := s.redis.Keys(ctx, pattern)
	if err != nil {
		log.Printf("查询连接keys失败: %v", err)
		return
	}

	cleanedConnections := 0
	cleanedUsers := make(map[string]bool)

	for _, key := range keys {
		// 获取连接信息
		connInfo, err := s.redis.HGetAll(ctx, key)
		if err != nil {
			continue
		}

		// 检查是否是本实例的连接
		if serverID, exists := connInfo["serverID"]; exists && serverID == s.instanceID {
			// 删除连接信息
			if err := s.redis.Del(ctx, key); err == nil {
				cleanedConnections++
			}

			// 记录需要从在线用户集合中移除的用户
			if userIDStr, exists := connInfo["userID"]; exists {
				cleanedUsers[userIDStr] = true
			}
		}
	}

	// 3. 从在线用户集合中移除用户
	for userID := range cleanedUsers {
		s.redis.SRem(ctx, "online_users", userID)
	}

	log.Printf("清理完成: 实例信息已删除, 清理了 %d 个连接, %d 个用户下线",
		cleanedConnections, len(cleanedUsers))
}

// GetInstanceID 获取实例ID
func (s *Service) GetInstanceID() string {
	return s.instanceID
}

// registerInstance 注册服务实例到Redis
func (s *Service) registerInstance() error {
	ctx := context.Background()

	// 服务实例信息
	instanceInfo := map[string]interface{}{
		"instance_id": s.instanceID,
		"host":        s.config.Connect.Instance.Host,
		"port":        s.config.Connect.Instance.Port,
		"status":      "active",
		"started_at":  time.Now().Unix(),
		"last_ping":   time.Now().Unix(),
	}

	// 注册到Redis Hash
	key := fmt.Sprintf("connect_instances:%s", s.instanceID)
	if err := s.redis.HMSet(ctx, key, instanceInfo); err != nil {
		return fmt.Errorf("注册实例信息失败: %v", err)
	}

	// 设置过期时间（心跳机制）
	expireTime := time.Duration(s.config.Connect.Heartbeat.Timeout) * time.Second
	if err := s.redis.Expire(ctx, key, expireTime); err != nil {
		log.Printf("设置实例过期时间失败: %v", err)
	}

	// 添加到实例列表
	if err := s.redis.SAdd(ctx, "connect_instances_list", s.instanceID); err != nil {
		log.Printf("添加到实例列表失败: %v", err)
	}

	log.Printf("Connect服务实例已注册: %s", s.instanceID)

	// 启动心跳
	go s.startHeartbeat()

	// 启动优雅退出监听
	go s.setupGracefulShutdown()

	return nil
}

// startHeartbeat 启动心跳机制
func (s *Service) startHeartbeat() {
	interval := time.Duration(s.config.Connect.Heartbeat.Interval) * time.Second
	timeout := time.Duration(s.config.Connect.Heartbeat.Timeout) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		key := fmt.Sprintf("connect_instances:%s", s.instanceID)

		// 更新心跳时间
		if err := s.redis.HSet(ctx, key, "last_ping", time.Now().Unix()); err != nil {
			log.Printf("更新心跳失败: %v", err)
			continue
		}

		// 续期
		if err := s.redis.Expire(ctx, key, timeout); err != nil {
			log.Printf("续期失败: %v", err)
		}
	}
}

// Connect 处理连接，写入 redis hash，并维护在线用户 set
func (s *Service) Connect(ctx context.Context, userID int64, token string, serverID, clientType string) (*model.Connection, error) {
	if token == "" {
		return nil, fmt.Errorf("token required")
	}
	timestamp := time.Now().Unix()
	connID := fmt.Sprintf("conn-%d-%d", userID, timestamp)
	conn := &model.Connection{
		UserID:        userID,
		ConnID:        connID,
		ServerID:      serverID,
		Timestamp:     timestamp,
		LastHeartbeat: timestamp,
		ClientType:    clientType,
		Online:        true,
	}
	key := fmt.Sprintf("conn:%d:%s", userID, connID)
	fields := map[string]interface{}{
		"userID":        userID,
		"connID":        connID,
		"serverID":      serverID,
		"timestamp":     timestamp,
		"lastHeartbeat": timestamp,
		"clientType":    clientType,
	}
	if err := s.redis.HMSet(ctx, key, fields); err != nil {
		return nil, err
	}
	expireTime := time.Duration(s.config.Connect.Connection.ExpireTime) * time.Hour
	_ = s.redis.Expire(ctx, key, expireTime)
	_ = s.redis.SAdd(ctx, "online_users", userID)
	return conn, nil
}

// Disconnect 处理断开，删除 redis hash，并维护在线用户 set
func (s *Service) Disconnect(ctx context.Context, userID int64, connID string) error {
	key := fmt.Sprintf("conn:%d:%s", userID, connID)
	err := s.redis.Del(ctx, key)
	_ = s.redis.SRem(ctx, "online_users", userID)
	return err
}

// Heartbeat 心跳，更新 lastHeartbeat 字段
func (s *Service) Heartbeat(ctx context.Context, userID int64, connID string) error {
	key := fmt.Sprintf("conn:%d:%s", userID, connID)
	timestamp := time.Now().Unix()
	if err := s.redis.HSet(ctx, key, "lastHeartbeat", timestamp); err != nil {
		return err
	}
	// 刷新过期时间
	expireTime := time.Duration(s.config.Connect.Connection.ExpireTime) * time.Hour
	return s.redis.Expire(ctx, key, expireTime)
}

// OnlineStatus 查询用户是否有活跃连接
func (s *Service) OnlineStatus(ctx context.Context, userIDs []int64) (map[int64]bool, error) {
	status := make(map[int64]bool)
	for _, uid := range userIDs {
		pattern := fmt.Sprintf("conn:%d:*", uid)
		keys, err := s.redis.Keys(ctx, pattern)
		if err != nil {
			status[uid] = false
			continue
		}
		status[uid] = len(keys) > 0
	}
	return status, nil
}

// ForwardMessageToLogicService 通过 gRPC 转发消息到 Logic 微服务
func (s *Service) ForwardMessageToLogicService(ctx context.Context, wsMsg *rest.WSMessage) error {
	log.Printf("Connect服务转发消息: From=%d, To=%d, Content=%s", wsMsg.From, wsMsg.To, wsMsg.Content)

	// 使用Chat服务的单向调用
	return s.sendMessageViaUnaryCall(ctx, wsMsg)
}

// HandleHeartbeat 处理心跳包
func (s *Service) HandleHeartbeat(ctx context.Context, wsMsg *rest.WSMessage, conn interface{}) error {
	// 这里假设 Content 字段存储 ConnID
	connID := wsMsg.Content
	if connID == "" {
		return fmt.Errorf("心跳包缺少 ConnID")
	}
	return s.Heartbeat(ctx, wsMsg.From, connID)
}

// ValidateToken 校验 JWT token
func (s *Service) ValidateToken(token string) bool {
	return auth.ValidateToken(token)
}

// HandleMessageACK 处理客户端的消息ACK确认
func (s *Service) HandleMessageACK(ctx context.Context, wsMsg *rest.WSMessage) error {
	// 从WebSocket消息中提取用户ID和消息ID
	userID := wsMsg.From // 客户端发送ACK时，From字段是自己的用户ID
	messageID := wsMsg.MessageId
	ackID := wsMsg.AckId

	log.Printf("收到客户端ACK: UserID=%d, MessageID=%d, AckID=%s", userID, messageID, ackID)

	// 检查消息ID是否存在
	if messageID == 0 {
		log.Printf("ACK消息ID为空: UserID=%d", userID)
		return fmt.Errorf("MessageID不能为0")
	}

	// 检查Logic服务客户端是否已初始化
	if s.logicClient == nil {
		log.Printf("Logic服务客户端未初始化，无法处理ACK: UserID=%d, MessageID=%d", userID, messageID)
		return fmt.Errorf("Logic服务客户端未初始化")
	}

	// 调用Logic服务处理ACK
	req := &rest.MessageAckRequest{
		UserId:    userID,
		MessageId: messageID,
		AckId:     ackID,
	}

	resp, err := s.logicClient.HandleMessageAck(ctx, req)
	if err != nil {
		log.Printf("调用Logic服务处理ACK失败: UserID=%d, MessageID=%d, Error=%v", userID, messageID, err)
		return fmt.Errorf("处理消息ACK失败: %v", err)
	}

	if !resp.Success {
		log.Printf("Logic服务处理ACK失败: UserID=%d, MessageID=%d, Message=%s", userID, messageID, resp.Message)
		return fmt.Errorf("处理消息ACK失败: %s", resp.Message)
	}

	log.Printf("消息ACK处理成功: UserID=%d, MessageID=%d", userID, messageID)
	return nil
}

// sendMessageViaUnaryCall 通过Logic服务单向调用发送消息
func (s *Service) sendMessageViaUnaryCall(ctx context.Context, wsMsg *rest.WSMessage) error {
	if s.logicClient == nil {
		return fmt.Errorf("Logic服务客户端未初始化")
	}

	log.Printf("通过Logic服务单向调用发送消息: From=%d, To=%d, GroupID=%d, Content=%s",
		wsMsg.From, wsMsg.To, wsMsg.GroupId, wsMsg.Content)

	// 调用Logic服务的SendMessage方法
	req := &rest.SendLogicMessageRequest{
		Msg: wsMsg,
	}

	resp, err := s.logicClient.SendMessage(ctx, req)
	if err != nil {
		log.Printf("Logic服务单向调用发送消息失败: %v", err)
		return err
	}

	if !resp.Success {
		log.Printf("Logic服务处理消息失败: %s", resp.Message)
		return fmt.Errorf("Logic服务处理消息失败: %s", resp.Message)
	}

	log.Printf("Logic服务单向调用发送消息成功: MessageID=%d, SuccessCount=%d",
		resp.MessageId, resp.SuccessCount)
	return nil
}

// AddWebSocketConnection 添加WebSocket连接（兼容旧接口）
func (s *Service) AddWebSocketConnection(userID int64, conn *websocket.Conn) {
	// 生成连接ID
	timestamp := time.Now().Unix()
	connID := fmt.Sprintf("conn-%d-%d", userID, timestamp)

	// 使用新的连接管理器
	ctx := context.Background()
	if err := s.connMgr.AddConnection(ctx, userID, conn, connID, s.instanceID); err != nil {
		log.Printf("添加WebSocket连接失败: %v", err)
	}
}

// RemoveWebSocketConnection 移除WebSocket连接（兼容旧接口）
func (s *Service) RemoveWebSocketConnection(userID int64) {
	ctx := context.Background()
	if err := s.connMgr.RemoveConnection(ctx, userID, ""); err != nil {
		log.Printf("移除WebSocket连接失败: %v", err)
	}
}

// 订阅 connect_forward 频道并分发消息到本地连接，在线才会这样推送，不在线就登陆时拉取未读消息
func (s *Service) subscribeConnectForward() {
	ctx := context.Background()
	channel := "connect_forward:" + s.instanceID
	log.Printf("订阅Redis频道: %s", channel)
	pubsub := s.redis.Subscribe(ctx, channel)
	ch := pubsub.Channel()
	for msg := range ch {
		log.Printf("收到推送消息: %s", msg.Payload)
		// 解析消息
		var pushMsg map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Payload), &pushMsg); err != nil {
			log.Printf("推送消息解析失败: %v", err)
			continue
		}
		// 获取目标用户ID和消息内容
		targetUser, ok := pushMsg["target_user"].(float64)
		if !ok {
			log.Printf("推送消息缺少 target_user 字段")
			continue
		}
		userID := int64(targetUser)

		// 直接从Redis消息中反序列化Protobuf二进制数据
		messageBytes, ok := pushMsg["message_bytes"].(string)
		if !ok {
			log.Printf("推送消息格式错误，缺少message_bytes字段: %v", pushMsg)
			continue
		}

		// 将base64编码的字节数据解码
		msgData, err := base64.StdEncoding.DecodeString(messageBytes)
		if err != nil {
			log.Printf("解码消息字节数据失败: %v", err)
			continue
		}

		// 反序列化Protobuf消息
		var wsMsg rest.WSMessage
		if err := proto.Unmarshal(msgData, &wsMsg); err != nil {
			log.Printf("反序列化Protobuf消息失败: %v", err)
			continue
		}
		// 推送到本地WebSocket连接
		if conn, exists := s.connMgr.GetConnection(userID); exists {
			msgBytes, err := proto.Marshal(&wsMsg)
			if err != nil {
				log.Printf("WebSocket推送protobuf序列化失败: %v", err)
				continue
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, msgBytes); err != nil {
				log.Printf("WebSocket推送失败: %v", err)
			} else {
				log.Printf("WebSocket推送成功: UserID=%d, MessageID=%d", userID, wsMsg.MessageId)
			}
		} else {
			log.Printf("用户 %d 不在本地连接，无法推送", userID)
		}
	}
}
