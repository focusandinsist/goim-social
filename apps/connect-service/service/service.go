package service

import (
	"context"
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
	"websocket-server/apps/connect-service/model"
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
	db         *database.MongoDB
	redis      *redis.RedisClient
	kafka      *kafka.Producer
	config     *config.Config         // 配置
	instanceID string                 // Connect服务实例ID
	chatClient rest.ChatServiceClient // Chat服务客户端
	connMgr    *ConnectionManager     // 统一连接管理器
}

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer, cfg *config.Config) *Service {
	service := &Service{
		db:         db,
		redis:      redis,
		kafka:      kafka,
		config:     cfg,
		instanceID: fmt.Sprintf("connect-%d", time.Now().UnixNano()), // 生成唯一实例ID
		connMgr:    NewConnectionManager(redis, cfg),                 // 初始化连接管理器
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
	s.chatClient = rest.NewChatServiceClient(conn)

	log.Printf("Logic服务客户端初始化成功，地址: %s", logicAddr)
	return nil
}

// StartLogicConnection 初始化与Logic服务的连接
func (s *Service) StartLogicConnection() {
	log.Printf("初始化Logic服务连接...")

	if s.chatClient == nil {
		log.Printf("Logic服务客户端未初始化")
		return
	}

	log.Printf("Logic服务连接已就绪")
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

	// 启动跨节点消息订阅
	go s.startCrossNodeSubscription()

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

// findUserInstance 查找用户所在的Connect实例
func (s *Service) findUserInstance(ctx context.Context, userID int64) (string, error) {
	// 查询用户连接信息
	pattern := fmt.Sprintf("conn:%d:*", userID)
	keys, err := s.redis.Keys(ctx, pattern)
	if err != nil {
		return "", fmt.Errorf("查询用户连接失败: %v", err)
	}

	if len(keys) == 0 {
		return "", fmt.Errorf("用户不在线")
	}

	// 获取连接信息
	connInfo, err := s.redis.HGetAll(ctx, keys[0])
	if err != nil {
		return "", fmt.Errorf("获取连接信息失败: %v", err)
	}

	serverID, exists := connInfo["serverID"]
	if !exists {
		return "", fmt.Errorf("连接信息中缺少serverID")
	}

	return serverID, nil
}

// forwardToRemoteInstance 转发消息到远程Connect实例
func (s *Service) forwardToRemoteInstance(ctx context.Context, targetInstance string, userID int64, message *rest.WSMessage) error {
	// 构造跨节点消息
	crossNodeMsg := map[string]interface{}{
		"type":          "forward_message",
		"from_instance": s.instanceID,
		"to_instance":   targetInstance,
		"user_id":       userID,
		"message":       message,
		"timestamp":     time.Now().Unix(),
	}

	// 序列化消息
	msgBytes, err := json.Marshal(crossNodeMsg)
	if err != nil {
		return fmt.Errorf("序列化跨节点消息失败: %v", err)
	}

	// 通过Redis发布到目标实例的频道
	channel := fmt.Sprintf("connect_forward:%s", targetInstance)
	if err := s.redis.Publish(ctx, channel, string(msgBytes)); err != nil {
		return fmt.Errorf("发布跨节点消息失败: %v", err)
	}

	log.Printf("已转发消息到远程实例: %s, UserID=%d, MessageID=%d", targetInstance, userID, message.MessageId)
	return nil
}

// startCrossNodeSubscription 启动跨节点消息订阅
func (s *Service) startCrossNodeSubscription() {
	ctx := context.Background()
	channel := fmt.Sprintf("connect_forward:%s", s.instanceID)

	// 订阅自己的转发频道
	pubsub := s.redis.Subscribe(ctx, channel)
	defer pubsub.Close()

	log.Printf("开始监听跨节点消息频道: %s", channel)

	// 接收消息
	ch := pubsub.Channel()
	for msg := range ch {
		if err := s.handleCrossNodeMessage(ctx, msg.Payload); err != nil {
			log.Printf("处理跨节点消息失败: %v", err)
		}
	}
}

// handleCrossNodeMessage 处理跨节点消息
func (s *Service) handleCrossNodeMessage(ctx context.Context, payload string) error {
	// 解析跨节点消息
	var crossNodeMsg map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &crossNodeMsg); err != nil {
		return fmt.Errorf("解析跨节点消息失败: %v", err)
	}

	msgType, ok := crossNodeMsg["type"].(string)
	if !ok {
		return fmt.Errorf("跨节点消息类型无效")
	}

	switch msgType {
	case "forward_message":
		return s.handleForwardMessage(ctx, crossNodeMsg)
	case "push_message":
		return s.handlePushMessage(ctx, crossNodeMsg)
	default:
		log.Printf("未知的跨节点消息类型: %s", msgType)
	}

	return nil
}

// handleForwardMessage 处理转发的消息
func (s *Service) handleForwardMessage(ctx context.Context, crossNodeMsg map[string]interface{}) error {
	// 提取用户ID
	userIDFloat, ok := crossNodeMsg["user_id"].(float64)
	if !ok {
		return fmt.Errorf("用户ID无效")
	}
	userID := int64(userIDFloat)

	// 提取消息内容
	messageData, ok := crossNodeMsg["message"]
	if !ok {
		return fmt.Errorf("消息内容无效")
	}

	// 重新序列化消息
	msgBytes, err := json.Marshal(messageData)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 反序列化为WSMessage
	var message rest.WSMessage
	if err := json.Unmarshal(msgBytes, &message); err != nil {
		return fmt.Errorf("反序列化WSMessage失败: %v", err)
	}

	// 推送给本地用户
	return s.pushToLocalUser(ctx, userID, &message)
}

// handlePushMessage 处理推送消息
func (s *Service) handlePushMessage(ctx context.Context, crossNodeMsg map[string]interface{}) error {
	// 提取目标用户ID
	targetUserFloat, ok := crossNodeMsg["target_user"].(float64)
	if !ok {
		return fmt.Errorf("目标用户ID无效")
	}
	targetUserID := int64(targetUserFloat)

	// 提取消息内容
	messageData, ok := crossNodeMsg["message"]
	if !ok {
		return fmt.Errorf("消息内容无效")
	}

	// 重新序列化消息
	msgBytes, err := json.Marshal(messageData)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 反序列化为WSMessage
	var message rest.WSMessage
	if err := json.Unmarshal(msgBytes, &message); err != nil {
		return fmt.Errorf("反序列化WSMessage失败: %v", err)
	}

	log.Printf("Connect服务收到推送消息: UserID=%d, MessageID=%d, Content=%s",
		targetUserID, message.MessageId, message.Content)

	// 推送给本地用户
	return s.pushToLocalUser(ctx, targetUserID, &message)
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

// HandleConnectionManage 处理连接管理包
func (s *Service) HandleConnectionManage(ctx context.Context, wsMsg *rest.WSMessage, conn interface{}) error {
	// 这里假设 Content 字段为 JSON 字符串或直接传递参数
	// 需根据实际协议解析 wsMsg 内容
	// 示例：直接用 wsMsg.From、wsMsg.Content、wsMsg.GroupId 等
	_, err := s.Connect(ctx, wsMsg.From, wsMsg.Content, fmt.Sprintf("%d", wsMsg.GroupId), "")
	return err
}

// HandleOnlineStatusEvent 处理在线状态事件推送
func (s *Service) HandleOnlineStatusEvent(ctx context.Context, wsMsg *rest.WSMessage, conn interface{}) error {
	// 这里 wsMsg.Content 应包含 userId、status（online/offline）、timestamp 等
	// 伪代码：将事件推送给所有相关好友
	// 实际场景下应维护好友连接映射
	// 示例：
	// event := map[string]interface{}{
	//     "type": "online_status",
	//     "user_id": wsMsg.Content["user_id"],
	//     "status": wsMsg.Content["status"],
	//     "timestamp": wsMsg.Content["timestamp"],
	// }
	// for _, friendConn := range 好友连接 {
	//     friendConn.WriteJSON(event)
	// }
	return nil // 具体推送逻辑根据实际业务补充
}

// ValidateToken 校验 JWT token
func (s *Service) ValidateToken(token string) bool {
	return auth.ValidateToken(token)
}

// isPushDuplicate 检查消息是否已推送给用户（防重复推送）
func (s *Service) isPushDuplicate(ctx context.Context, userID int64, messageID int64) bool {
	key := fmt.Sprintf("push:%d:%d", userID, messageID)
	exists, err := s.redis.Exists(ctx, key)
	if err != nil {
		log.Printf("检查推送重复状态失败: %v", err)
		return false // 出错时假设未推送，允许推送
	}
	return exists > 0
}

// markPushSent 标记消息已推送给用户
func (s *Service) markPushSent(ctx context.Context, userID int64, messageID int64) error {
	key := fmt.Sprintf("push:%d:%d", userID, messageID)
	return s.redis.Set(ctx, key, "pushed", 10*time.Minute) // 10分钟过期
}

// pushToLocalConnection 推送消息给用户，支持跨节点路由
func (s *Service) pushToLocalConnection(targetUserID int64, message *rest.WSMessage) error {
	log.Printf("开始推送消息给用户 %d, 消息内容: %s", targetUserID, message.Content)

	// 幂等性检查：检查消息是否已推送
	ctx := context.Background()
	if s.isPushDuplicate(ctx, targetUserID, message.MessageId) {
		log.Printf("消息已推送，跳过: UserID=%d, MessageID=%d", targetUserID, message.MessageId)
		return nil
	}

	// 查找用户所在的Connect实例
	targetInstance, err := s.findUserInstance(ctx, targetUserID)
	if err != nil {
		log.Printf("用户 %d 不在线或查找实例失败: %v", targetUserID, err)
		return err
	}

	// 判断是本地连接还是跨节点连接
	if targetInstance == s.instanceID {
		// 本地连接，直接推送
		return s.pushToLocalUser(ctx, targetUserID, message)
	} else {
		// 跨节点连接，通过Redis发布订阅转发
		return s.forwardToRemoteInstance(ctx, targetInstance, targetUserID, message)
	}
}

// pushToLocalUser 推送消息给本地用户
func (s *Service) pushToLocalUser(ctx context.Context, targetUserID int64, message *rest.WSMessage) error {
	// 先检查Redis中用户是否在线
	isOnline, err := s.connMgr.IsUserOnline(ctx, targetUserID)
	if err != nil {
		log.Printf("Redis查询失败，用户 %d: %v", targetUserID, err)
		return err
	}

	// 调试：显示所有在线用户
	allOnlineUsers, _ := s.connMgr.GetOnlineUsers(ctx)
	log.Printf("当前Redis中的在线用户: %v", allOnlineUsers)

	if !isOnline {
		log.Printf("用户 %d 在Redis中显示不在线", targetUserID)
		return fmt.Errorf("用户 %d 不在线", targetUserID)
	}
	log.Printf("用户 %d 在Redis中显示在线", targetUserID)

	// 2. 将消息序列化为二进制（在获取连接前先序列化）
	msgBytes, err := proto.Marshal(message)
	if err != nil {
		log.Printf("消息序列化失败: %v", err)
		return err
	}

	// 3. 获取连接
	conn, exists := s.connMgr.GetConnection(targetUserID)
	stats := s.connMgr.GetStats()

	log.Printf("本地连接状态: 总连接数=%d, 用户%d连接存在=%v", stats["local_connections"], targetUserID, exists)

	if !exists {
		log.Printf("用户 %d 没有本地WebSocket连接，可能在其他Connect服务实例上", targetUserID)
		log.Printf("当前本地连接列表: %v", stats["connection_list"])
		return fmt.Errorf("用户 %d 没有本地WebSocket连接", targetUserID)
	}

	// 4. 推送消息
	log.Printf("尝试通过WebSocket推送消息给用户 %d，消息长度: %d bytes", targetUserID, len(msgBytes))

	// 添加连接状态检查
	if conn == nil {
		log.Printf("用户 %d 的WebSocket连接为nil", targetUserID)
		s.connMgr.RemoveConnection(context.Background(), targetUserID, "")
		return fmt.Errorf("用户 %d 的WebSocket连接为nil", targetUserID)
	}

	err = conn.WriteMessage(websocket.BinaryMessage, msgBytes)

	// 5. 处理推送结果
	if err != nil {
		log.Printf("推送消息给用户 %d 失败: %v", targetUserID, err)
		log.Printf("错误类型: %T", err)
		// 如果推送失败，可能连接已断开，移除连接
		s.connMgr.RemoveConnection(context.Background(), targetUserID, "")
	} else {
		log.Printf("成功推送消息给用户 %d，消息内容: %s", targetUserID, message.Content)
		// 标记消息已推送
		if err := s.markPushSent(ctx, targetUserID, message.MessageId); err != nil {
			log.Printf("标记消息已推送失败: %v", err)
		}
		// 注意：这里不自动ACK，等待客户端主动确认已读
	}
	return nil
}

// HandleMessageACK 处理客户端的消息ACK确认
func (s *Service) HandleMessageACK(ctx context.Context, wsMsg *rest.WSMessage) error {
	// 从WebSocket消息中提取用户ID和消息ID
	userID := wsMsg.From // 客户端发送ACK时，From字段是自己的用户ID
	messageID := wsMsg.MessageId

	log.Printf("收到客户端ACK: UserID=%d, MessageID=%d", userID, messageID)

	// 检查消息ID是否存在
	if messageID == 0 {
		log.Printf("ACK消息ID为空: UserID=%d", userID)
		return fmt.Errorf("MessageID不能为0")
	}

	return nil
}

// notifyMessageFailure 通知消息发送失败
func (s *Service) notifyMessageFailure(originalSender int64, failureReason string) {
	// TODO: 实现失败通知逻辑
	// 这里应该通知原发送者消息发送失败
	log.Printf("通知用户 %d 消息发送失败: %s", originalSender, failureReason)
}

// sendMessageViaUnaryCall 通过Logic服务单向调用发送消息
func (s *Service) sendMessageViaUnaryCall(ctx context.Context, wsMsg *rest.WSMessage) error {
	if s.chatClient == nil {
		return fmt.Errorf("Logic服务客户端未初始化")
	}

	log.Printf("通过Logic服务单向调用发送消息: From=%d, To=%d, GroupID=%d, Content=%s",
		wsMsg.From, wsMsg.To, wsMsg.GroupId, wsMsg.Content)

	// 调用Logic服务的SendMessage方法
	req := &rest.SendChatMessageRequest{
		Msg: wsMsg,
	}

	resp, err := s.chatClient.SendMessage(ctx, req)
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

// CleanupInvalidConnections 清理所有失效的连接（被动清理，在推送失败时调用）
func (s *Service) CleanupInvalidConnections() {
	// 这个方法现在主要用于日志记录，实际清理在推送失败时进行
	stats := s.connMgr.GetStats()
	log.Printf("🧹 当前活跃连接数: %d", stats["local_connections"])
}

// UpdateHeartbeat 更新连接的心跳时间
func (s *Service) UpdateHeartbeat(ctx context.Context, userID int64, connID string, timestamp int64) error {
	connKey := fmt.Sprintf("conn:%d:%s", userID, connID)

	// 更新Redis中的lastHeartbeat字段
	err := s.redis.HSet(ctx, connKey, "lastHeartbeat", timestamp)
	if err != nil {
		log.Printf("更新用户 %d 心跳时间失败: %v", userID, err)
		return err
	}

	return nil
}

// CleanupAllConnections 清理Redis连接记录，服务关闭时调用
func (s *Service) CleanupAllConnections() {
	ctx := context.Background()

	log.Printf("开始清理Redis中的连接记录和实例信息...")

	// 清理实例注册信息
	instanceKey := fmt.Sprintf("connect_instances:%s", s.instanceID)
	if err := s.redis.Del(ctx, instanceKey); err != nil {
		log.Printf("清理实例信息失败: %v", err)
	} else {
		log.Printf("已清理实例信息: %s", s.instanceID)
	}

	// 2. 清理本实例的连接记录
	connKeys, err := s.redis.Keys(ctx, "conn:*")
	if err != nil {
		log.Printf("获取连接记录失败: %v", err)
	} else {
		cleanedConnections := 0
		cleanedUsers := make(map[string]bool)

		for _, key := range connKeys {
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

		// 从在线用户集合中移除用户
		for userID := range cleanedUsers {
			s.redis.SRem(ctx, "online_users", userID)
		}

		log.Printf("已清理 %d 个本实例连接记录, %d 个用户下线", cleanedConnections, len(cleanedUsers))
	}

	// 3. 清理本地连接管理器
	s.connMgr.CleanupAll()

	log.Printf("Redis连接记录和实例信息清理完成")
}
