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

	"github.com/IBM/sarama"
)

// StorageConsumer 存储消费者
type StorageConsumer struct {
	db       *database.MongoDB
	consumer *kafka.Consumer
}

// MessageEvent Kafka消息事件结构
type MessageEvent struct {
	Type      string          `json:"type"`
	Message   *rest.WSMessage `json:"message"`
	Timestamp int64           `json:"timestamp"`
}

// NewStorageConsumer 创建存储消费者
func NewStorageConsumer(db *database.MongoDB) *StorageConsumer {
	return &StorageConsumer{
		db: db,
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
		return nil
	default:
		log.Printf("⚠️  未知的消息事件类型: %s", event.Type)
		return nil
	}
}

// handleNewMessage 处理新消息存储
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
		AckID:     msg.AckId,
		Status:    0, // 0:未读
		CreatedAt: time.Unix(msg.Timestamp, 0),
		UpdatedAt: time.Now(),
	}

	// 存储到MongoDB
	_, err := s.db.GetCollection("message").InsertOne(context.Background(), message)
	if err != nil {
		log.Printf("❌ 存储消息失败: %v", err)
		return err
	}

	log.Printf("✅ 消息存储成功: From=%d, To=%d, Status=未读, MessageID=%d", msg.From, msg.To, msg.MessageId)

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
