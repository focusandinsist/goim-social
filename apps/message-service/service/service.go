package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"websocket-server/api/rest"
	"websocket-server/apps/message-service/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"
)

// Service Message服务 - 专注于消息持久化
type Service struct {
	db    *database.MongoDB
	redis *redis.RedisClient
	kafka *kafka.Producer
}

// GRPCService gRPC服务包装器
type GRPCService struct {
	svc *Service
}

// NewService 创建Message服务实例
func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:    db,
		redis: redis,
		kafka: kafka,
	}
}

// NewGRPCService 创建gRPC服务实例
func NewGRPCService(svc *Service) *GRPCService {
	return &GRPCService{svc: svc}
}

// SendWSMessage 发送并持久化消息
func (g *GRPCService) SendWSMessage(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
	msg := req.Msg
	log.Printf("Message服务接收消息: From=%d, To=%d, GroupID=%d, Content=%s",
		msg.From, msg.To, msg.GroupId, msg.Content)

	// 1. 数据验证
	if msg.From <= 0 {
		return &rest.SendWSMessageResponse{
			Success: false,
			Message: "发送者ID无效",
		}, nil
	}

	if msg.To <= 0 && msg.GroupId <= 0 {
		return &rest.SendWSMessageResponse{
			Success: false,
			Message: "接收者或群组ID必须指定一个",
		}, nil
	}

	if msg.Content == "" {
		return &rest.SendWSMessageResponse{
			Success: false,
			Message: "消息内容不能为空",
		}, nil
	}

	// 2. 构造消息对象
	message := &model.Message{
		ID:          primitive.NewObjectID(),
		MessageID:   msg.MessageId,
		From:        msg.From,
		To:          msg.To,
		GroupID:     msg.GroupId,
		Content:     msg.Content,
		MessageType: int(msg.MessageType),
		Timestamp:   time.Now().Unix(),
		AckID:       msg.AckId,
		Status:      model.MessageStatusSent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 3. 持久化消息
	err := g.svc.SaveMessage(ctx, message)
	if err != nil {
		log.Printf("消息持久化失败: %v", err)
		return &rest.SendWSMessageResponse{
			Success: false,
			Message: "消息保存失败",
		}, nil
	}

	log.Printf("消息持久化成功: MessageID=%d", msg.MessageId)
	return &rest.SendWSMessageResponse{
		Success: true,
		Message: "消息保存成功",
	}, nil
}

// SaveMessage 保存消息到数据库
func (s *Service) SaveMessage(ctx context.Context, msg *model.Message) error {
	collection := s.db.GetCollection("messages")

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	msg.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, msg)
	if err != nil {
		return fmt.Errorf("保存消息失败: %v", err)
	}

	return nil
}

// GetHistoryMessages 获取历史消息
func (g *GRPCService) GetHistoryMessages(ctx context.Context, req *rest.GetHistoryRequest) (*rest.GetHistoryResponse, error) {
	log.Printf("获取历史消息: UserID=%d, GroupID=%d, Page=%d, Size=%d",
		req.UserId, req.GroupId, req.Page, req.Size)

	messages, total, err := g.svc.GetMessageHistory(ctx, req.UserId, req.GroupId, int(req.Page), int(req.Size))
	if err != nil {
		log.Printf("获取历史消息失败: %v", err)
		return nil, err
	}

	// 转换为gRPC响应格式
	var wsMessages []*rest.WSMessage
	for _, msg := range messages {
		wsMessages = append(wsMessages, &rest.WSMessage{
			MessageId:   msg.MessageID,
			From:        msg.From,
			To:          msg.To,
			GroupId:     msg.GroupID,
			Content:     msg.Content,
			Timestamp:   msg.Timestamp,
			MessageType: int32(msg.MessageType),
			AckId:       msg.AckID,
		})
	}

	log.Printf("获取历史消息成功: 总数=%d, 返回=%d", total, len(wsMessages))
	return &rest.GetHistoryResponse{
		Messages: wsMessages,
		Total:    int32(total),
		Page:     req.Page,
		Size:     req.Size,
	}, nil
}

// GetMessageHistory 获取消息历史
func (s *Service) GetMessageHistory(ctx context.Context, userID, groupID int64, page, size int) ([]*model.Message, int64, error) {
	collection := s.db.GetCollection("messages")

	// 构建查询条件
	var filter bson.M
	if groupID > 0 {
		// 群聊消息
		filter = bson.M{"group_id": groupID}
	} else {
		// 私聊消息
		filter = bson.M{
			"$or": []bson.M{
				{"from": userID, "to": userID},
				{"from": userID, "to": userID},
			},
		}
	}

	// 获取总数
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("统计消息数量失败: %v", err)
	}

	// 分页查询
	skip := int64((page - 1) * size)
	limit := int64(size)

	opts := options.Find().
		SetSort(bson.D{{"timestamp", -1}}). // 按时间倒序
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("查询消息失败: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []*model.Message
	for cursor.Next(ctx) {
		var msg model.Message
		if err := cursor.Decode(&msg); err != nil {
			log.Printf("解码消息失败: %v", err)
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, total, nil
}

// UpdateMessageStatus 更新消息状态
func (s *Service) UpdateMessageStatus(ctx context.Context, messageID int64, status string) error {
	collection := s.db.GetCollection("messages")

	filter := bson.M{"message_id": messageID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("更新消息状态失败: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("消息不存在: MessageID=%d", messageID)
	}

	return nil
}

// MarkMessageAsRead 标记消息为已读
func (s *Service) MarkMessageAsRead(ctx context.Context, userID, messageID int64) error {
	// 验证用户是否有权限标记该消息为已读
	collection := s.db.GetCollection("messages")

	filter := bson.M{
		"message_id": messageID,
		"$or": []bson.M{
			{"to": userID},                 // 私聊中的接收者
			{"group_id": bson.M{"$gt": 0}}, // 群聊消息（需要进一步验证群成员身份）
		},
	}

	var message model.Message
	err := collection.FindOne(ctx, filter).Decode(&message)
	if err != nil {
		return fmt.Errorf("消息不存在或无权限: %v", err)
	}

	// 更新消息状态为已读
	return s.UpdateMessageStatus(ctx, messageID, model.MessageStatusRead)
}

// DeleteMessage 删除消息
func (s *Service) DeleteMessage(ctx context.Context, messageID int64, userID int64) error {
	collection := s.db.GetCollection("messages")

	// 只允许发送者删除自己的消息
	filter := bson.M{
		"message_id": messageID,
		"from":       userID,
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("删除消息失败: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("消息不存在或无权限删除")
	}

	return nil
}
