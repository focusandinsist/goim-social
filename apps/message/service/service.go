package service

import (
	"context"
	"encoding/json"
	"log"
	"time"
	"websocket-server/api/rest"
	"websocket-server/apps/message/consumer"
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

// 移除本地StreamManager，使用consumer包中的全局StreamManager

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

	// 1. 发布消息到Kafka（异步处理）
	messageEvent := map[string]interface{}{
		"type":      "new_message",
		"message":   req.Msg,
		"timestamp": time.Now().Unix(),
	}

	if err := g.svc.kafka.PublishMessage("message-events", messageEvent); err != nil {
		log.Printf("❌ 发布消息到Kafka失败: %v", err)
		return &rest.SendWSMessageResponse{Success: false, Message: "消息发送失败"}, err
	}

	log.Printf("✅ 消息已发布到Kafka: From=%d, To=%d", req.Msg.From, req.Msg.To)
	return &rest.SendWSMessageResponse{Success: true, Message: "消息发送成功"}, nil
}

// MessageStream 实现双向流通信
func (g *GRPCService) MessageStream(stream rest.MessageService_MessageStreamServer) error {
	// 存储连接的Connect服务实例
	var connectServiceID string

	// 获取全局流管理器
	streamManager := consumer.GetStreamManager()

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
