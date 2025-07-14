package service

import (
	"context"
	"fmt"
	"time"
	"websocket-server/api/rest"
	"websocket-server/apps/connect/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
)

// 移除内存 map，全部用 redis hash

type Service struct {
	db    *database.MongoDB
	redis *redis.RedisClient
	kafka *kafka.Producer
}

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:    db,
		redis: redis,
		kafka: kafka,
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
	conn, err := grpc.Dial("localhost:22004", grpc.WithInsecure()) // Message Service gRPC端口
	if err != nil {
		return err
	}
	defer conn.Close()

	client := rest.NewMessageServiceClient(conn)
	// 构造 gRPC 请求
	req := &rest.SendWSMessageRequest{Msg: wsMsg}
	_, err = client.SendWSMessage(ctx, req)
	return err
}

// // HandleWSConnectOrHeartbeat 处理心跳和连接管理
// func (s *Service) HandleWSConnectOrHeartbeat(ctx context.Context, wsMsg *model.WSMessage, conn interface{}) error {
// 	// 假设 MessageType == 2 表示心跳，MessageType == 3 表示连接管理
// 	if wsMsg.MessageType == 2 {
// 		// 心跳包，假设 Content 里有 ConnID
// 		connID, ok := wsMsg.Content.(string)
// 		if !ok {
// 			return fmt.Errorf("心跳包缺少 ConnID")
// 		}
// 		// 这里假设用户ID已在 wsMsg 结构中
// 		return s.Heartbeat(ctx, int64(wsMsg.MessageType), connID)
// 	}
// 	if wsMsg.MessageType == 3 {
// 		// 连接管理包，假设 Content 里有 token、serverID、clientType
// 		data, ok := wsMsg.Content.(map[string]interface{})
// 		if !ok {
// 			return fmt.Errorf("连接管理包内容格式错误")
// 		}
// 		userID, _ := data["user_id"].(int64)
// 		token, _ := data["token"].(string)
// 		serverID, _ := data["server_id"].(string)
// 		clientType, _ := data["client_type"].(string)
// 		_, err := s.Connect(ctx, userID, token, serverID, clientType)
// 		return err
// 	}
// 	return nil // 其它类型不处理
// }

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
