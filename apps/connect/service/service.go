package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
	"websocket-server/api/rest"
	"websocket-server/apps/connect/model"
	"websocket-server/pkg/auth"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// 统一连接管理器 - 封装本地WebSocket连接和Redis状态
type ConnectionManager struct {
	localConnections map[int64]*websocket.Conn // 本地WebSocket连接
	redis            *redis.RedisClient        // Redis客户端
	mutex            sync.RWMutex              // 读写锁
}

// 创建连接管理器
func NewConnectionManager(redis *redis.RedisClient) *ConnectionManager {
	return &ConnectionManager{
		localConnections: make(map[int64]*websocket.Conn),
		redis:            redis,
	}
}

// 原子式添加连接 - 同时更新本地连接和Redis状态
func (cm *ConnectionManager) AddConnection(ctx context.Context, userID int64, conn *websocket.Conn, connID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 1. 检查是否已存在连接，如果有则关闭旧连接
	if existingConn, exists := cm.localConnections[userID]; exists {
		log.Printf("⚠️ 用户 %d 已有WebSocket连接，将替换旧连接", userID)
		existingConn.Close()
	}

	// 2. 添加到本地连接管理
	cm.localConnections[userID] = conn

	// 3. 更新Redis状态
	// 添加到在线用户集合
	if err := cm.redis.SAdd(ctx, "online_users", userID); err != nil {
		// Redis操作失败，回滚本地操作
		delete(cm.localConnections, userID)
		return fmt.Errorf("添加Redis在线状态失败: %v", err)
	}

	// 4. 添加连接信息到Redis Hash
	key := fmt.Sprintf("conn:%d:%s", userID, connID)
	connInfo := map[string]interface{}{
		"userID":        userID,
		"connID":        connID,
		"serverID":      "connect-server-1", // 可以从配置获取
		"clientType":    "web",
		"timestamp":     time.Now().Unix(),
		"lastHeartbeat": time.Now().Unix(),
	}

	if err := cm.redis.HMSet(ctx, key, connInfo); err != nil {
		// Redis操作失败，回滚之前的操作
		delete(cm.localConnections, userID)
		cm.redis.SRem(ctx, "online_users", userID)
		return fmt.Errorf("添加Redis连接信息失败: %v", err)
	}

	// 5. 设置连接过期时间
	if err := cm.redis.Expire(ctx, key, 2*time.Hour); err != nil {
		log.Printf("⚠️ 设置连接过期时间失败: %v", err)
	}

	totalConnections := len(cm.localConnections)
	log.Printf("✅ 用户 %d 连接已添加 (本地+Redis)，当前总连接数: %d", userID, totalConnections)
	return nil
}

// 原子式移除连接 - 同时清理本地连接和Redis状态
func (cm *ConnectionManager) RemoveConnection(ctx context.Context, userID int64, connID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 1. 从本地连接管理中移除并关闭连接
	if conn, exists := cm.localConnections[userID]; exists {
		conn.Close()
		delete(cm.localConnections, userID)
		log.Printf("✅ 用户 %d 的本地WebSocket连接已关闭并移除", userID)
	}

	// 2. 从Redis在线用户集合中移除
	if err := cm.redis.SRem(ctx, "online_users", userID); err != nil {
		log.Printf("❌ 从Redis移除用户 %d 在线状态失败: %v", userID, err)
	} else {
		log.Printf("✅ 用户 %d 已从Redis在线用户列表中移除", userID)
	}

	// 3. 删除Redis中的连接信息
	if connID != "" {
		key := fmt.Sprintf("conn:%d:%s", userID, connID)
		if err := cm.redis.Del(ctx, key); err != nil {
			log.Printf("❌ 删除Redis连接信息失败: %v", err)
		} else {
			log.Printf("✅ 用户 %d 的Redis连接信息已删除", userID)
		}
	}

	totalConnections := len(cm.localConnections)
	log.Printf("✅ 用户 %d 连接已完全清理，剩余连接数: %d", userID, totalConnections)
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
			log.Printf("✅ 已关闭用户 %d 的WebSocket连接", userID)
		}
	}

	// 清空连接map
	cm.localConnections = make(map[int64]*websocket.Conn)

	log.Printf("✅ 所有本地连接已清理完成")
}

type Service struct {
	db         *database.MongoDB
	redis      *redis.RedisClient
	kafka      *kafka.Producer
	instanceID string                                  // Connect服务实例ID
	msgStream  rest.MessageService_MessageStreamClient // 消息流连接
	connMgr    *ConnectionManager                      // 统一连接管理器
}

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:         db,
		redis:      redis,
		kafka:      kafka,
		instanceID: fmt.Sprintf("connect-%d", time.Now().UnixNano()), // 生成唯一实例ID
		connMgr:    NewConnectionManager(redis),                      // 初始化连接管理器
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
		"userId":        userID,
		"connectionId":  connID,
		"serverId":      serverID,
		"timestamp":     timestamp,
		"lastHeartbeat": timestamp,
		"clientType":    clientType,
	}
	if err := s.redis.HMSet(ctx, key, fields); err != nil {
		return nil, err
	}
	_ = s.redis.Expire(ctx, key, 2*time.Hour)
	// 新增：将用户ID加入在线用户集合
	_ = s.redis.SAdd(ctx, "online_users", userID)
	return conn, nil
}

// Disconnect 处理断开，删除 redis hash，并维护在线用户 set
func (s *Service) Disconnect(ctx context.Context, userID int64, connID string) error {
	key := fmt.Sprintf("conn:%d:%s", userID, connID)
	err := s.redis.Del(ctx, key)
	// 新增：将用户ID移出在线用户集合
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
	return s.redis.Expire(ctx, key, 2*time.Hour)
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

// ForwardMessageToMessageService 通过 gRPC 转发消息到 Message 微服务
func (s *Service) ForwardMessageToMessageService(ctx context.Context, wsMsg *rest.WSMessage) error {
	log.Printf("📨 Connect服务转发消息: From=%d, To=%d, Content=%s", wsMsg.From, wsMsg.To, wsMsg.Content)

	// 双向流是IM系统的核心，必须可用
	if s.msgStream == nil {
		log.Printf("❌ 双向流连接不可用，IM系统无法正常工作")
		return fmt.Errorf("双向流连接不可用，无法转发消息")
	}

	log.Printf("🔄 通过双向流转发消息")
	return s.SendMessageViaStream(ctx, wsMsg)
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

func (s *Service) StartMessageStream() {
	log.Printf("🚀 开始连接Message服务...")

	// 重试连接Message服务
	for i := 0; i < 10; i++ {
		if i == 0 {
			log.Printf("🔄 尝试连接Message服务... (第%d次)", i+1)
		} else {
			log.Printf("🔄 重试连接Message服务... (第%d次) - 等待Message服务启动完成", i+1)
		}

		conn, err := grpc.Dial("localhost:22004", grpc.WithInsecure())
		if err != nil {
			log.Printf("❌ 连接Message服务失败: %v", err)
			if i < 9 {
				log.Printf("⏳ 等待2秒后重试...")
			}
			time.Sleep(2 * time.Second)
			continue
		}

		client := rest.NewMessageServiceClient(conn)
		stream, err := client.MessageStream(context.Background())
		if err != nil {
			log.Printf("❌ 创建消息流失败: %v", err)
			conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		s.msgStream = stream // 保存stream连接
		log.Printf("✅ 成功连接到Message服务")

		// 发送订阅请求
		err = stream.Send(&rest.MessageStreamRequest{
			RequestType: &rest.MessageStreamRequest_Subscribe{
				Subscribe: &rest.SubscribeRequest{ConnectServiceId: s.instanceID},
			},
		})
		if err != nil {
			log.Printf("❌ 发送订阅请求失败: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// 连接成功，启动消息接收goroutine
		go func(stream rest.MessageService_MessageStreamClient) {
			for {
				resp, err := stream.Recv()
				if err != nil {
					log.Printf("❌ 消息流接收失败: %v", err)
					return
				}
				switch respType := resp.ResponseType.(type) {
				case *rest.MessageStreamResponse_PushEvent:
					event := respType.PushEvent
					// 推送给本地用户
					err := s.pushToLocalConnection(event.TargetUserId, event.Message)
					if err != nil {
						log.Printf("[X]pushToLocalConnection fail: %v", err)
						continue
					}
					// 发送推送结果反馈
					err = stream.Send(&rest.MessageStreamRequest{
						RequestType: &rest.MessageStreamRequest_PushResult{
							PushResult: &rest.PushResultRequest{
								Success:      true,
								TargetUserId: event.TargetUserId,
							},
						},
					})
					if err != nil {
						log.Printf("[X]stream.Send fail: %v", err)
						continue
					}
				case *rest.MessageStreamResponse_Failure:
					failure := respType.Failure
					// 通知原发送者消息失败
					s.notifyMessageFailure(failure.OriginalSender, failure.FailureReason)
				}
			}
		}(stream)

		// 连接成功，跳出重试循环
		break
	}
}

// pushToLocalConnection 推送消息给本地连接的用户
func (s *Service) pushToLocalConnection(targetUserID int64, message *rest.WSMessage) error {
	log.Printf("🔍 开始推送消息给用户 %d, 消息内容: %s", targetUserID, message.Content)

	// 1. 先检查Redis中用户是否在线
	ctx := context.Background()
	isOnline, err := s.connMgr.IsUserOnline(ctx, targetUserID)
	if err != nil {
		log.Printf("❌ Redis查询失败，用户 %d: %v", targetUserID, err)
		return err
	}

	// 调试：显示所有在线用户
	allOnlineUsers, _ := s.connMgr.GetOnlineUsers(ctx)
	log.Printf("🔍 当前Redis中的在线用户: %v", allOnlineUsers)

	if !isOnline {
		log.Printf("❌ 用户 %d 在Redis中显示不在线", targetUserID)
		return fmt.Errorf("用户 %d 不在线", targetUserID)
	}
	log.Printf("✅ 用户 %d 在Redis中显示在线", targetUserID)

	// 2. 将消息序列化为二进制（在获取连接前先序列化）
	msgBytes, err := proto.Marshal(message)
	if err != nil {
		log.Printf("❌ 消息序列化失败: %v", err)
		return err
	}

	// 3. 获取连接
	conn, exists := s.connMgr.GetConnection(targetUserID)
	stats := s.connMgr.GetStats()

	log.Printf("🔍 本地连接状态: 总连接数=%d, 用户%d连接存在=%v", stats["local_connections"], targetUserID, exists)

	if !exists {
		log.Printf("❌ 用户 %d 没有本地WebSocket连接，可能在其他Connect服务实例上", targetUserID)
		log.Printf("🔍 当前本地连接列表: %v", stats["connection_list"])
		return fmt.Errorf("用户 %d 没有本地WebSocket连接", targetUserID)
	}

	// 4. 推送消息
	log.Printf("📤 尝试通过WebSocket推送消息给用户 %d，消息长度: %d bytes", targetUserID, len(msgBytes))

	// 添加连接状态检查
	if conn == nil {
		log.Printf("❌ 用户 %d 的WebSocket连接为nil", targetUserID)
		s.connMgr.RemoveConnection(context.Background(), targetUserID, "")
		return fmt.Errorf("用户 %d 的WebSocket连接为nil", targetUserID)
	}

	err = conn.WriteMessage(websocket.BinaryMessage, msgBytes)

	// 5. 处理推送结果
	if err != nil {
		log.Printf("❌ 推送消息给用户 %d 失败: %v", targetUserID, err)
		log.Printf("🔍 错误类型: %T", err)
		// 如果推送失败，可能连接已断开，移除连接
		s.connMgr.RemoveConnection(context.Background(), targetUserID, "")
	} else {
		log.Printf("✅ 成功推送消息给用户 %d，消息内容: %s", targetUserID, message.Content)

		// 6. 推送成功后，标记消息为已读
		go s.markMessageAsRead(targetUserID, message)
	}
	return nil
}

// markMessageAsRead 标记消息为已读
func (s *Service) markMessageAsRead(userID int64, message *rest.WSMessage) {
	// 检查消息ID是否存在
	if message.MessageId == 0 {
		log.Printf("⚠️ 消息ID为空，无法标记为已读: UserID=%d, Content=%s", userID, message.Content)
		return
	}

	log.Printf("📖 标记消息为已读: UserID=%d, MessageID=%d, From=%d", userID, message.MessageId, message.From)

	// 通过双向流发送ACK请求
	if s.msgStream != nil {
		ackReq := &rest.MessageStreamRequest{
			RequestType: &rest.MessageStreamRequest_Ack{
				Ack: &rest.MessageAckRequest{
					AckId:     message.AckId,
					MessageId: message.MessageId,
					UserId:    userID,
					Timestamp: time.Now().Unix(),
				},
			},
		}

		err := s.msgStream.Send(ackReq)
		if err != nil {
			log.Printf("❌ 发送消息ACK失败: %v", err)
		} else {
			log.Printf("✅ 已发送消息ACK: MessageID=%d, UserID=%d", message.MessageId, userID)
		}
	} else {
		log.Printf("❌ 双向流连接不可用，无法发送ACK")
	}
}

// notifyMessageFailure 通知消息发送失败
func (s *Service) notifyMessageFailure(originalSender int64, failureReason string) {
	// TODO: 实现失败通知逻辑
	// 这里应该通知原发送者消息发送失败
	log.Printf("通知用户 %d 消息发送失败: %s", originalSender, failureReason)
}

// SendMessageViaStream 通过双向流发送消息
func (s *Service) SendMessageViaStream(ctx context.Context, wsMsg *rest.WSMessage) error {
	if s.msgStream == nil {
		return fmt.Errorf("消息流连接未建立")
	}

	// 通过双向流发送消息
	log.Printf("📡 通过双向流发送消息: From=%d, To=%d, Content=%s", wsMsg.From, wsMsg.To, wsMsg.Content)

	err := s.msgStream.Send(&rest.MessageStreamRequest{
		RequestType: &rest.MessageStreamRequest_SendMessage{
			SendMessage: &rest.SendWSMessageRequest{
				Msg: wsMsg,
			},
		},
	})

	if err != nil {
		log.Printf("❌ 双向流发送消息失败: %v", err)
		return err
	}

	log.Printf("✅ 双向流发送消息成功")
	return nil
}

// AddWebSocketConnection 添加WebSocket连接（兼容旧接口）
func (s *Service) AddWebSocketConnection(userID int64, conn *websocket.Conn) {
	// 生成连接ID
	timestamp := time.Now().Unix()
	connID := fmt.Sprintf("conn-%d-%d", userID, timestamp)

	// 使用新的连接管理器
	ctx := context.Background()
	if err := s.connMgr.AddConnection(ctx, userID, conn, connID); err != nil {
		log.Printf("❌ 添加WebSocket连接失败: %v", err)
	}
}

// RemoveWebSocketConnection 移除WebSocket连接（兼容旧接口）
func (s *Service) RemoveWebSocketConnection(userID int64) {
	ctx := context.Background()
	if err := s.connMgr.RemoveConnection(ctx, userID, ""); err != nil {
		log.Printf("❌ 移除WebSocket连接失败: %v", err)
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
		log.Printf("❌ 更新用户 %d 心跳时间失败: %v", userID, err)
		return err
	}

	return nil
}

// CleanupAllConnections 清理所有Redis连接记录（服务关闭时调用）
func (s *Service) CleanupAllConnections() {
	ctx := context.Background()

	log.Printf("🧹 开始清理Redis中的所有连接记录...")

	// 1. 清理所有连接记录 (conn:*:* 模式)
	connKeys, err := s.redis.Keys(ctx, "conn:*")
	if err != nil {
		log.Printf("❌ 获取连接记录失败: %v", err)
	} else if len(connKeys) > 0 {
		// 批量删除连接记录
		if err := s.redis.Del(ctx, connKeys...); err != nil {
			log.Printf("❌ 删除连接记录失败: %v", err)
		} else {
			log.Printf("✅ 已删除 %d 个连接记录", len(connKeys))
		}
	}

	// 2. 清空在线用户集合
	if err := s.redis.Del(ctx, "online_users"); err != nil {
		log.Printf("❌ 清空在线用户集合失败: %v", err)
	} else {
		log.Printf("✅ 已清空在线用户集合")
	}

	// 3. 清理本地连接管理器
	s.connMgr.CleanupAll()

	log.Printf("✅ Redis连接记录清理完成")
}
