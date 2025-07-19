package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"websocket-server/api/rest"
	"websocket-server/apps/message/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"

	"github.com/IBM/sarama"
	"go.mongodb.org/mongo-driver/bson"
)

// StorageConsumer 存储消费者
type StorageConsumer struct {
	db       *database.MongoDB
	consumer *kafka.Consumer
	redis    *redis.RedisClient
}

// MessageEvent Kafka消息事件结构
type MessageEvent struct {
	Type      string          `json:"type"`
	Message   *rest.WSMessage `json:"message"`
	Timestamp int64           `json:"timestamp"`
}

// NewStorageConsumer 创建存储消费者
func NewStorageConsumer(db *database.MongoDB, redis *redis.RedisClient) *StorageConsumer {
	return &StorageConsumer{
		db:    db,
		redis: redis,
	}
}

// Start 启动存储消费者
func (s *StorageConsumer) Start(ctx context.Context, brokers []string) error {
	cfg := kafka.KafkaConfig{
		Brokers: brokers,
		GroupID: "storage-consumer-group",
		Topics:  []string{"message-events"},
	}

	consumer, err := kafka.InitConsumer(cfg, s)
	if err != nil {
		return err
	}

	s.consumer = consumer
	log.Printf("✅ 存储消费者启动成功，监听topic: message-events")

	return s.consumer.StartConsuming(ctx)
}

// HandleMessage 实现 kafka.ConsumerHandler 接口
func (s *StorageConsumer) HandleMessage(msg *sarama.ConsumerMessage) error {
	log.Printf("📥 存储消费者收到消息: topic=%s, partition=%d, offset=%d",
		msg.Topic, msg.Partition, msg.Offset)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ 存储消费者处理消息时发生panic: %v", r)
		}
	}()

	// 幂等性检查：检查消息是否已处理
	ctx := context.Background()
	if s.isMessageProcessed(ctx, msg.Partition, msg.Offset) {
		log.Printf("✅ 消息已处理，跳过: partition=%d, offset=%d", msg.Partition, msg.Offset)
		return nil
	}

	// 解析消息事件
	var event MessageEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("❌ 解析消息事件失败: %v, 原始消息: %s", err, string(msg.Value))
		return nil // 返回nil避免重试
	}

	// 根据事件类型处理
	switch event.Type {
	case "new_message":
		if err := s.handleNewMessage(event.Message); err != nil {
			log.Printf("❌ 处理新消息失败: %v", err)
			return nil // 返回nil避免重试
		}

		// 标记消息已处理
		if err := s.markMessageProcessed(ctx, msg.Partition, msg.Offset); err != nil {
			log.Printf("❌ 标记消息已处理失败: %v", err)
		}

		return nil
	default:
		log.Printf("⚠️  未知的消息事件类型: %s", event.Type)
		return nil
	}
}

// isMessageProcessed 检查消息是否已处理（幂等性检查）
func (s *StorageConsumer) isMessageProcessed(ctx context.Context, partition int32, offset int64) bool {
	key := fmt.Sprintf("kafka:storage:%d:%d", partition, offset)
	exists, err := s.redis.Exists(ctx, key)
	if err != nil {
		log.Printf("❌ 检查消息处理状态失败: %v", err)
		return false // 出错时假设未处理，允许重试
	}
	return exists > 0 // Redis Exists返回存在的key数量
}

// markMessageProcessed 标记消息已处理
func (s *StorageConsumer) markMessageProcessed(ctx context.Context, partition int32, offset int64) error {
	key := fmt.Sprintf("kafka:storage:%d:%d", partition, offset)
	return s.redis.Set(ctx, key, "processed", time.Hour) // 1小时过期
}

// handleNewMessage 处理新消息存储（带幂等性保护）
func (s *StorageConsumer) handleNewMessage(msg *rest.WSMessage) error {
	log.Printf("💾 存储消息: From=%d, To=%d, Content=%s, MessageID=%d", msg.From, msg.To, msg.Content, msg.MessageId)

	// 检查MessageID是否存在
	if msg.MessageId == 0 {
		log.Printf("❌ MessageID为0，跳过存储: From=%d, To=%d", msg.From, msg.To)
		return fmt.Errorf("MessageID不能为0")
	}

	// 转换为Message模型并设置状态
	message := &model.Message{
		// 不设置ID，让MongoDB自动生成_id
		MessageID: msg.MessageId, // 直接使用Kafka消息中的MessageID
		From:      msg.From,
		To:        msg.To,
		GroupID:   msg.GroupId,
		Content:   msg.Content,
		MsgType:   msg.MessageType,
		Status:    0, // 0:未读
		CreatedAt: time.Unix(msg.Timestamp, 0),
		UpdatedAt: time.Now(),
	}

	// 先尝试简单的插入操作，如果重复则忽略
	collection := s.db.GetCollection("message")

	// 检查消息是否已存在
	var existingMsg model.Message
	err := collection.FindOne(context.Background(), bson.M{"message_id": msg.MessageId}).Decode(&existingMsg)
	if err == nil {
		// 消息已存在，跳过插入
		log.Printf("✅ 消息已存在(幂等性保护): From=%d, To=%d, MessageID=%d", msg.From, msg.To, msg.MessageId)
		return nil
	}

	// 消息不存在，执行插入
	result, err := collection.InsertOne(context.Background(), message)
	if err != nil {
		log.Printf("❌ 存储消息失败: %v", err)
		return err
	}

	log.Printf("✅ 消息存储成功: From=%d, To=%d, Status=未读, MessageID=%d, InsertedID=%v",
		msg.From, msg.To, msg.MessageId, result.InsertedID)

	return nil
}

// Stop 停止消费者
func (s *StorageConsumer) Stop() error {
	if s.consumer != nil {
		// TODO: 实现优雅停止
		log.Printf("🛑 存储消费者停止")
	}
	return nil
}
