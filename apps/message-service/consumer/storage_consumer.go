package consumer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	"go.mongodb.org/mongo-driver/mongo"
	"goim-social/api/rest"
	"goim-social/apps/message-service/model"
	"goim-social/pkg/database"
	"goim-social/pkg/kafka"
)

// StorageConsumer 存储消费者
// 幂等性保护：依赖MongoDB的MessageID唯一索引
type StorageConsumer struct {
	db       *database.MongoDB
	consumer *kafka.Consumer
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
		Topics:  []string{"uplink_messages"},
	}

	consumer, err := kafka.InitConsumer(cfg, s)
	if err != nil {
		return err
	}

	s.consumer = consumer
	log.Printf("存储消费者启动成功，监听topic: uplink_messages")

	return s.consumer.StartConsuming(ctx)
}

// HandleMessage 实现 kafka.ConsumerHandler 接口
func (s *StorageConsumer) HandleMessage(msg *sarama.ConsumerMessage) error {
	log.Printf("存储消费者收到消息: topic=%s, partition=%d, offset=%d",
		msg.Topic, msg.Partition, msg.Offset)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("存储消费者处理消息时发生panic: %v", r)
		}
	}()

	var event rest.MessageEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("解析protobuf消息事件失败: %v, 原始消息: %s", err, string(msg.Value))
		return nil // 返回nil避免重试
	}

	// 根据事件类型处理
	switch event.Type {
	case "new_message":
		if err := s.handleNewMessage(event.Message); err != nil {
			log.Printf("处理新消息失败: %v", err)
			return nil // 返回nil避免重试
		}

		log.Printf("消息存储成功或已存在: MessageID=%d", event.Message.MessageId)
		return nil

	default:
		log.Printf("未知的消息事件类型: %s", event.Type)
		return nil
	}
}

// handleNewMessage 处理新消息存储（使用乐观插入策略）
func (s *StorageConsumer) handleNewMessage(msg *rest.WSMessage) error {
	log.Printf("存储消息: From=%d, To=%d, Content=%s, MessageID=%d", msg.From, msg.To, msg.Content, msg.MessageId)

	// 检查MessageID是否存在
	if msg.MessageId == 0 {
		log.Printf("MessageID为0，跳过存储: From=%d, To=%d", msg.From, msg.To)
		return fmt.Errorf("MessageID不能为0")
	}

	// 转换为Message模型并设置状态
	message := &model.Message{
		// 不设置ID，让MongoDB自动生成_id
		MessageID:   msg.MessageId, // 直接使用Kafka消息中的MessageID
		From:        msg.From,
		To:          msg.To,
		GroupID:     msg.GroupId,
		Content:     msg.Content,
		MessageType: int(msg.MessageType),
		Timestamp:   msg.Timestamp,
		Status:      model.MessageStatusSent,
		CreatedAt:   time.Unix(msg.Timestamp, 0),
		UpdatedAt:   time.Now(),
	}

	// 乐观插入策略：直接尝试插入，让MongoDB唯一索引处理重复
	collection := s.db.GetCollection("messages")
	_, err := collection.InsertOne(context.Background(), message)

	// 检查错误类型
	if err != nil {
		// 如果是重复键错误，说明是幂等触发，这不是一个真正的错误
		if mongo.IsDuplicateKeyError(err) {
			log.Printf("消息已存在(唯一索引幂等性保护): MessageID=%d", msg.MessageId)
			return nil // 幂等处理，返回成功
		}
		// 其他类型的数据库错误
		log.Printf("存储消息失败: %v", err)
		return err
	}

	log.Printf("消息存储成功: From=%d, To=%d, Status=未读, MessageID=%d",
		msg.From, msg.To, msg.MessageId)

	return nil
}

// Stop 停止消费者
func (s *StorageConsumer) Stop() error {
	if s.consumer != nil {
		// TODO: 实现优雅停止
		log.Printf("存储消费者停止")
	}
	return nil
}
