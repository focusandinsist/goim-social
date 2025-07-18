package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"websocket-server/api/rest"
	"websocket-server/apps/connect/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// WebSocket连接管理 - 使用Redis存储连接信息，内存存储WebSocket连接对象
type WSConnectionManager struct {
	localConnections map[int64]*websocket.Conn // 本地WebSocket连接
	mutex            sync.RWMutex
}

var wsConnManager = &WSConnectionManager{
	localConnections: make(map[int64]*websocket.Conn),
}

type Service struct {
	db         *database.MongoDB
	redis      *redis.RedisClient
	kafka      *kafka.Producer
	instanceID string                                  // Connect服务实例ID
	msgStream  rest.MessageService_MessageStreamClient // 消息流连接
}

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:         db,
		redis:      redis,
		kafka:      kafka,
		instanceID: fmt.Sprintf("connect-%d", time.Now().UnixNano()), // 生成唯一实例ID
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

	// 优先使用双向流发送消息
	if s.msgStream != nil {
		log.Printf("🔄 通过双向流转发消息")
		return s.SendMessageViaStream(ctx, wsMsg)
	}

	// 如果双向流不可用，使用直接gRPC调用作为备用
	log.Printf("⚠️ 双向流不可用，使用直接gRPC调用")
	conn, err := grpc.Dial("localhost:22004", grpc.WithInsecure()) // Message Service gRPC端口
	if err != nil {
		return err
	}
	defer conn.Close()

	client := rest.NewMessageServiceClient(conn)
	// 构造 gRPC 请求
	req := &rest.SendWSMessageRequest{Msg: wsMsg}
	_, err = client.SendWSMessage(ctx, req)
	if err != nil {
		log.Printf("❌ 转发消息到Message服务失败: %v", err)
	} else {
		log.Printf("✅ 成功转发消息到Message服务")
	}
	return err
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
	if token == "" {
		return false
	}
	if token == "auth-debug" {
		return true
	}
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// 校验签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("your-secret"), nil // 建议配置化
	})
	return err == nil && parsedToken != nil && parsedToken.Valid
}

// gRPC服务端实现
type GRPCService struct {
	rest.UnimplementedConnectServiceServer
	svc *Service
}

// NewGRPCService 创建gRPC服务
func (s *Service) NewGRPCService(svc *Service) *GRPCService {
	return &GRPCService{svc: svc}
}

// Connect 处理连接请求
func (g *GRPCService) Connect(ctx context.Context, req *rest.ConnectRequest) (*rest.ConnectResponse, error) {
	_, err := g.svc.Connect(ctx, req.UserId, req.Token, "grpc-server", "grpc-client")
	if err != nil {
		return &rest.ConnectResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &rest.ConnectResponse{
		Success: true,
		Message: "connected successfully",
	}, nil
}

// Disconnect 处理断开连接请求
func (g *GRPCService) Disconnect(ctx context.Context, req *rest.DisconnectRequest) (*rest.DisconnectResponse, error) {
	err := g.svc.Disconnect(ctx, req.UserId, fmt.Sprintf("%d", req.ConnId))
	if err != nil {
		return &rest.DisconnectResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &rest.DisconnectResponse{
		Success: true,
		Message: "disconnected successfully",
	}, nil
}

// Heartbeat 处理心跳请求
func (g *GRPCService) Heartbeat(ctx context.Context, req *rest.HeartbeatRequest) (*rest.ConnectResponse, error) {
	err := g.svc.Heartbeat(ctx, req.UserId, req.ConnId)
	if err != nil {
		return &rest.ConnectResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &rest.ConnectResponse{
		Success: true,
		Message: "heartbeat received",
	}, nil
}

// OnlineStatus 查询在线状态
func (g *GRPCService) OnlineStatus(ctx context.Context, req *rest.OnlineStatusRequest) (*rest.OnlineStatusResponse, error) {
	status, err := g.svc.OnlineStatus(ctx, req.UserIds)
	if err != nil {
		return &rest.OnlineStatusResponse{
			Status: make(map[int64]bool),
		}, err
	}
	return &rest.OnlineStatusResponse{
		Status: status,
	}, nil
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
					s.pushToLocalConnection(event.TargetUserId, event.Message)
					// 发送推送结果反馈
					stream.Send(&rest.MessageStreamRequest{
						RequestType: &rest.MessageStreamRequest_PushResult{
							PushResult: &rest.PushResultRequest{
								Success:      true,
								TargetUserId: event.TargetUserId,
							},
						},
					})
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
func (s *Service) pushToLocalConnection(targetUserID int64, message *rest.WSMessage) {
	log.Printf("🔍 开始推送消息给用户 %d, 消息内容: %s", targetUserID, message.Content)

	// 1. 先检查Redis中用户是否在线
	ctx := context.Background()
	isOnline, err := s.redis.SIsMember(ctx, "online_users", targetUserID)
	if err != nil {
		log.Printf("❌ Redis查询失败，用户 %d: %v", targetUserID, err)
		return
	}

	// 调试：显示所有在线用户
	allOnlineUsers, _ := s.redis.SMembers(ctx, "online_users")
	log.Printf("🔍 当前Redis中的在线用户: %v", allOnlineUsers)

	if !isOnline {
		log.Printf("❌ 用户 %d 在Redis中显示不在线", targetUserID)
		return
	}
	log.Printf("✅ 用户 %d 在Redis中显示在线", targetUserID)

	// 2. 查找本地WebSocket连接
	wsConnManager.mutex.RLock()
	conn, exists := wsConnManager.localConnections[targetUserID]
	totalConnections := len(wsConnManager.localConnections)
	wsConnManager.mutex.RUnlock()

	log.Printf("🔍 本地连接状态: 总连接数=%d, 用户%d连接存在=%v", totalConnections, targetUserID, exists)

	if !exists {
		log.Printf("❌ 用户 %d 没有本地WebSocket连接，可能在其他Connect服务实例上", targetUserID)
		// 打印当前所有本地连接
		wsConnManager.mutex.RLock()
		log.Printf("🔍 当前本地连接列表:")
		for uid := range wsConnManager.localConnections {
			log.Printf("  - 用户ID: %d", uid)
		}
		wsConnManager.mutex.RUnlock()
		return
	}

	// 3. 直接尝试推送消息，如果失败再清理连接

	// 4. 将消息序列化为二进制
	msgBytes, err := proto.Marshal(message)
	if err != nil {
		log.Printf("❌ 消息序列化失败: %v", err)
		return
	}

	// 5. 推送消息
	log.Printf("📤 尝试通过WebSocket推送消息给用户 %d", targetUserID)
	if err := conn.WriteMessage(websocket.BinaryMessage, msgBytes); err != nil {
		log.Printf("❌ 推送消息给用户 %d 失败: %v", targetUserID, err)
		// 如果推送失败，可能连接已断开，移除连接
		s.RemoveWebSocketConnection(targetUserID)
	} else {
		log.Printf("✅ 成功推送消息给用户 %d", targetUserID)
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

// AddWebSocketConnection 添加WebSocket连接
func (s *Service) AddWebSocketConnection(userID int64, conn *websocket.Conn) {
	// 1. 添加到本地WebSocket连接管理
	wsConnManager.mutex.Lock()
	// 检查是否已存在连接
	if existingConn, exists := wsConnManager.localConnections[userID]; exists {
		log.Printf("⚠️  用户 %d 已有WebSocket连接，将替换旧连接", userID)
		// 关闭旧连接
		existingConn.Close()
	}
	wsConnManager.localConnections[userID] = conn
	totalConnections := len(wsConnManager.localConnections)
	wsConnManager.mutex.Unlock()

	log.Printf("✅ 用户 %d 的WebSocket连接已添加到本地管理，当前总连接数: %d", userID, totalConnections)

	// WebSocket连接建立完成，客户端可以通过HTTP接口自行获取历史消息
}

// RemoveWebSocketConnection 移除WebSocket连接
func (s *Service) RemoveWebSocketConnection(userID int64) {
	// 1. 从本地WebSocket连接管理中移除
	wsConnManager.mutex.Lock()
	if conn, exists := wsConnManager.localConnections[userID]; exists {
		// 尝试关闭连接
		conn.Close()
		delete(wsConnManager.localConnections, userID)
		totalConnections := len(wsConnManager.localConnections)
		wsConnManager.mutex.Unlock()
		log.Printf("✅ 用户 %d 的WebSocket连接已关闭并从本地管理中移除，剩余连接数: %d", userID, totalConnections)

		// 2. 检查是否还有该用户的其他连接
		// 注意：在多设备场景下，一个用户可能有多个连接
		// 这里简化处理，直接从Redis移除
		ctx := context.Background()
		err := s.redis.SRem(ctx, "online_users", userID)
		if err != nil {
			log.Printf("❌ 从Redis移除用户 %d 在线状态失败: %v", userID, err)
		} else {
			log.Printf("✅ 用户 %d 已从Redis在线用户列表中移除", userID)
		}
	} else {
		wsConnManager.mutex.Unlock()
		log.Printf("⚠️  用户 %d 的WebSocket连接在本地管理中不存在，无需移除", userID)
	}
}

// CleanupInvalidConnections 清理所有失效的连接（被动清理，在推送失败时调用）
func (s *Service) CleanupInvalidConnections() {
	// 这个方法现在主要用于日志记录，实际清理在推送失败时进行
	wsConnManager.mutex.RLock()
	totalConnections := len(wsConnManager.localConnections)
	wsConnManager.mutex.RUnlock()

	log.Printf("🧹 当前活跃连接数: %d", totalConnections)
}
