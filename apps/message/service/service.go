package service

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
	"websocket-server/api/rest"
	"websocket-server/apps/message/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"

	"github.com/gorilla/websocket"
)

type Service struct {
	db    *database.MongoDB
	redis *redis.RedisClient
	kafka *kafka.Producer
}

// ConnectStream 存储Connect服务的流连接
type ConnectStream struct {
	ServiceID string
	Stream    rest.MessageService_MessageStreamServer
}

// StreamManager 管理所有Connect服务的流连接
type StreamManager struct {
	streams map[string]*ConnectStream
	mutex   sync.RWMutex
}

var streamManager = &StreamManager{
	streams: make(map[string]*ConnectStream),
}

// AddStream 添加Connect服务流连接
func (sm *StreamManager) AddStream(serviceID string, stream rest.MessageService_MessageStreamServer) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.streams[serviceID] = &ConnectStream{
		ServiceID: serviceID,
		Stream:    stream,
	}
	log.Printf("添加Connect服务流连接: %s", serviceID)
}

// RemoveStream 移除Connect服务流连接
func (sm *StreamManager) RemoveStream(serviceID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.streams, serviceID)
	log.Printf("移除Connect服务流连接: %s", serviceID)
}

// PushToAllStreams 推送消息到所有Connect服务
func (sm *StreamManager) PushToAllStreams(targetUserID int64, message *rest.WSMessage) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	for serviceID, connectStream := range sm.streams {
		go func(sid string, stream rest.MessageService_MessageStreamServer) {
			err := stream.Send(&rest.MessageStreamResponse{
				ResponseType: &rest.MessageStreamResponse_PushEvent{
					PushEvent: &rest.MessagePushEvent{
						TargetUserId: targetUserID,
						Message:      message,
						EventType:    "new_message",
					},
				},
			})
			if err != nil {
				log.Printf("推送消息到Connect服务 %s 失败: %v", sid, err)
				// 如果推送失败，移除这个连接
				sm.RemoveStream(sid)
			} else {
				log.Printf("成功推送消息到Connect服务 %s, 目标用户: %d", sid, targetUserID)
			}
		}(serviceID, connectStream.Stream)
	}
}

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:    db,
		redis: redis,
		kafka: kafka,
	}
}

// SendMessage 发送消息
func (s *Service) SendMessage(ctx context.Context, msg *model.Message) error {
	// TODO: 持久化消息、推送到目标用户/群组
	return nil
}

// GetHistory 获取历史消息
func (s *Service) GetHistory(ctx context.Context, userID, groupID int64, page, size int) ([]*model.Message, int, error) {
	// TODO: 查询历史消息
	return []*model.Message{}, 0, nil
}

// HandleWSMessage 处理 WebSocket 消息收发并存储到 MongoDB
func (s *Service) HandleWSMessage(ctx context.Context, wsMsg *model.WSMessage, conn *websocket.Conn) error {
	// 构造消息
	msg := &model.HistoryMessage{
		ID:        wsMsg.MessageID,
		From:      wsMsg.From,
		To:        wsMsg.To,
		GroupID:   wsMsg.GroupID,
		Content:   wsMsg.Content,
		MsgType:   wsMsg.MessageType,
		AckID:     wsMsg.AckID,
		CreatedAt: time.Now(),
		Status:    0, // 默认未读
	}
	// 存储到 MongoDB 消息表（collection: message）
	_, err := s.db.GetCollection("message").InsertOne(ctx, msg)
	if err != nil {
		return err
	}
	// 回显消息（可扩展为推送给目标用户）
	resp, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, resp)
}

// gRPC接口实现
func (s *Service) SendWSMessage(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
	msg := req.Msg
	// 持久化到MongoDB
	_, err := s.db.GetCollection("message").InsertOne(ctx, msg)
	if err != nil {
		return &rest.SendWSMessageResponse{Success: false, Message: err.Error()}, nil
	}
	// 可选: 推送到Kafka等
	return &rest.SendWSMessageResponse{Success: true, Message: "OK"}, nil
}

type GRPCService struct {
	rest.UnimplementedMessageServiceServer
	svc *Service
}

// NewGRPCService 构造函数
func (s *Service) NewGRPCService(svc *Service) *GRPCService {
	return &GRPCService{svc: svc}
}

func (g *GRPCService) SendWSMessage(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
	log.Printf("📥 Message服务接收消息: From=%d, To=%d, Content=%s", req.Msg.From, req.Msg.To, req.Msg.Content)

	// 1. 存储消息到数据库
	_, err := g.svc.db.GetCollection("message").InsertOne(ctx, req.Msg)
	if err != nil {
		return &rest.SendWSMessageResponse{Success: false, Message: err.Error()}, err
	}

	// 2. 推送消息给目标用户
	if req.Msg.To > 0 {
		// 单聊消息：推送给目标用户
		log.Printf("推送单聊消息: From=%d, To=%d, Content=%s", req.Msg.From, req.Msg.To, req.Msg.Content)
		streamManager.PushToAllStreams(req.Msg.To, req.Msg)
	} else if req.Msg.GroupId > 0 {
		// 群聊消息：需要查询群成员并推送给所有成员
		log.Printf("推送群聊消息: From=%d, GroupID=%d, Content=%s", req.Msg.From, req.Msg.GroupId, req.Msg.Content)
		// TODO: 查询群成员列表，推送给所有成员
		// 这里简化处理，假设群成员ID为1,2,3
		groupMembers := []int64{1, 2, 3}
		for _, memberID := range groupMembers {
			if memberID != req.Msg.From { // 不推送给发送者自己
				streamManager.PushToAllStreams(memberID, req.Msg)
			}
		}
	}

	return &rest.SendWSMessageResponse{Success: true, Message: "消息发送成功"}, nil
}

// MessageStream 实现双向流通信
func (g *GRPCService) MessageStream(stream rest.MessageService_MessageStreamServer) error {
	// 存储连接的Connect服务实例
	var connectServiceID string

	// 在函数返回时移除连接
	defer func() {
		if connectServiceID != "" {
			streamManager.RemoveStream(connectServiceID)
		}
	}()

	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		switch reqType := req.RequestType.(type) {
		case *rest.MessageStreamRequest_Subscribe:
			// Connect服务订阅消息推送
			connectServiceID = reqType.Subscribe.ConnectServiceId
			log.Printf("Connect服务 %s 已订阅消息推送", connectServiceID)

			// 添加到连接管理器
			streamManager.AddStream(connectServiceID, stream)

		case *rest.MessageStreamRequest_Ack:
			// 处理消息确认
			ack := reqType.Ack
			log.Printf("收到消息确认: MessageID=%d, UserID=%d", ack.MessageId, ack.UserId)

			// 发送确认回复
			stream.Send(&rest.MessageStreamResponse{
				ResponseType: &rest.MessageStreamResponse_AckConfirm{
					AckConfirm: &rest.AckConfirmEvent{
						AckId:     ack.AckId,
						MessageId: ack.MessageId,
						Confirmed: true,
					},
				},
			})

		case *rest.MessageStreamRequest_PushResult:
			// 处理推送结果反馈
			result := reqType.PushResult
			if result.Success {
				log.Printf("消息推送成功: UserID=%d", result.TargetUserId)
			} else {
				log.Printf("消息推送失败: UserID=%d, Error=%s", result.TargetUserId, result.ErrorMessage)
			}
		}
	}
}
