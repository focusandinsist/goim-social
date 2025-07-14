package service

import (
	"context"
	"encoding/json"
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
	// 类型转换 rest.WSMessage -> model.Message
	msg := &model.Message{
		ID:      req.Msg.MessageId,
		From:    req.Msg.From,
		To:      req.Msg.To,
		GroupID: req.Msg.GroupId,
		Content: req.Msg.Content,
		MsgType: req.Msg.MessageType,
		AckID:   req.Msg.AckId,
		// Timestamp:   req.Msg.Timestamp,
	}
	err := g.svc.SendMessage(ctx, msg)
	if err != nil {
		return &rest.SendWSMessageResponse{Success: false, Message: err.Error()}, err
	}
	return &rest.SendWSMessageResponse{Success: true, Message: "ok"}, nil
}
