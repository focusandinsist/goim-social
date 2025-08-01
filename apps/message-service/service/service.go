package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goim-social/api/rest"
	"goim-social/apps/message-service/model"
	"goim-social/pkg/database"
	"goim-social/pkg/kafka"
	"goim-social/pkg/redis"
)

// Service Message服务
type Service struct {
	db    *database.MongoDB
	redis *redis.RedisClient
	kafka *kafka.Producer
}

// NewService 创建Message服务实例
func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:    db,
		redis: redis,
		kafka: kafka,
	}
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

// GetMessageHistory 获取消息历史
func (s *Service) GetMessageHistory(ctx context.Context, userID, groupID int64, page, size int) ([]*model.Message, int64, error) {
	collection := s.db.GetCollection("messages")

	// 构建查询条件
	var filter bson.M
	if groupID > 0 {
		// 群聊消息：查询该群组的所有消息
		filter = bson.M{"group_id": groupID}
	} else {
		// 私聊消息：查询用户参与的对话
		// 这里需要另一个参数来指定对话的另一方
		// 暂时查询用户相关的所有私聊消息
		filter = bson.M{
			"$and": []bson.M{
				{"group_id": bson.M{"$eq": 0}}, // 不是群聊
				{
					"$or": []bson.M{
						{"from": userID},
						{"to": userID},
					},
				},
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
		SetSort(bson.D{{Key: "timestamp", Value: -1}}). // 按时间倒序
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

// SaveWSMessage 保存WebSocket消息
func (s *Service) SaveWSMessage(ctx context.Context, msg *rest.WSMessage) error {
	// 构造消息对象
	message := &model.Message{
		ID:          primitive.NewObjectID(),
		MessageID:   msg.MessageId,
		From:        msg.From,
		To:          msg.To,
		GroupID:     msg.GroupId,
		Content:     msg.Content,
		MessageType: int(msg.MessageType),
		Timestamp:   msg.Timestamp,
		AckID:       msg.AckId,
		Status:      model.MessageStatusSent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.SaveMessage(ctx, message)
}

// SendMessage 发送消息（HTTP接口用）
func (s *Service) SendMessage(ctx context.Context, req *rest.SendMessageRequest) (int64, string, error) {
	// 生成消息ID和AckID
	messageID := time.Now().UnixNano()
	ackID := fmt.Sprintf("ack_%d", messageID)

	// 构造消息对象
	message := &model.Message{
		ID:          primitive.NewObjectID(),
		MessageID:   messageID,
		From:        0, // TODO: 从上下文获取用户ID
		To:          req.To,
		GroupID:     req.GroupId,
		Content:     req.Content,
		MessageType: int(req.MessageType),
		Timestamp:   time.Now().Unix(),
		AckID:       ackID,
		Status:      model.MessageStatusSent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := s.SaveMessage(ctx, message)
	if err != nil {
		return 0, "", err
	}

	return messageID, ackID, nil
}

// GetUnreadMessages 获取未读消息
func (s *Service) GetUnreadMessages(ctx context.Context, userID int64) ([]*model.Message, error) {
	collection := s.db.GetCollection("messages")

	// 查询未读消息
	filter := bson.M{
		"$or": []bson.M{
			{"to": userID, "status": bson.M{"$ne": model.MessageStatusRead}},
			{"group_id": bson.M{"$gt": 0}, "status": bson.M{"$ne": model.MessageStatusRead}},
		},
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("查询未读消息失败: %v", err)
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

	return messages, nil
}

// MarkMessagesAsRead 批量标记消息已读
func (s *Service) MarkMessagesAsRead(ctx context.Context, userID int64, messageIDs []int64) ([]int64, error) {
	var failedIDs []int64

	for _, messageID := range messageIDs {
		if err := s.MarkMessageAsRead(ctx, userID, messageID); err != nil {
			failedIDs = append(failedIDs, messageID)
		}
	}

	return failedIDs, nil
}
