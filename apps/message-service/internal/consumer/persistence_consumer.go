package consumer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/proto"

	"goim-social/api/rest"
	"goim-social/apps/message-service/internal/model"
	"goim-social/pkg/database"
	"goim-social/pkg/kafka"
)

// PersistenceConsumer 专门的持久化消费者
// 职责：消费message_persistence_log Topic，执行消息归档
// 幂等性保护：依赖MongoDB的MessageID唯一索引
type PersistenceConsumer struct {
	db       *database.MongoDB
	consumer *kafka.Consumer
}

// NewPersistenceConsumer 创建持久化消费者
func NewPersistenceConsumer(db *database.MongoDB) *PersistenceConsumer {
	return &PersistenceConsumer{
		db: db,
	}
}

// Start 启动持久化消费者
func (p *PersistenceConsumer) Start(ctx context.Context, brokers []string) error {
	log.Printf("正在初始化持久化消费者，brokers: %v", brokers)

	cfg := kafka.KafkaConfig{
		Brokers: brokers,
		GroupID: "persistence-consumer-group",        // 独立的Consumer Group
		Topics:  []string{"message_persistence_log"}, // 专门的持久化Topic
	}

	consumer, err := kafka.InitConsumer(cfg, p)
	if err != nil {
		log.Printf("初始化持久化消费者失败: %v", err)
		return err
	}

	p.consumer = consumer
	log.Printf("持久化消费者启动成功，监听topic: message_persistence_log, GroupID: persistence-consumer-group")

	return p.consumer.StartConsuming(ctx)
}

// HandleMessage 实现 kafka.ConsumerHandler 接口
func (p *PersistenceConsumer) HandleMessage(msg *sarama.ConsumerMessage) error {
	log.Printf("持久化消费者收到归档命令: topic=%s, partition=%d, offset=%d, 消息大小=%d bytes",
		msg.Topic, msg.Partition, msg.Offset, len(msg.Value))

	defer func() {
		if r := recover(); r != nil {
			log.Printf("持久化消费者处理消息时发生panic: %v", r)
		}
	}()

	// 幂等性由MongoDB的MessageID唯一索引保证
	var event rest.MessageEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("解析归档命令失败: %v", err)
		return nil // 返回nil避免重试
	}

	log.Printf("解析归档命令成功: Type=%s, MessageID=%d", event.Type, event.Message.MessageId)

	// 根据事件类型处理
	switch event.Type {
	case "archive_message":
		if err := p.handleArchiveMessage(event.Message); err != nil {
			log.Printf("处理消息归档失败: %v", err)
			// 即使失败也返回nil，避免Kafka无休止地重试毒消息
			return nil
		}
		log.Printf("消息归档成功或已存在: MessageID=%d", event.Message.MessageId)
		return nil
	default:
		log.Printf("未知的归档命令类型: %s", event.Type)
		return nil
	}
}

// handleArchiveMessage 处理消息归档（经过Logic Service处理的标准格式）
// 使用乐观插入策略：直接插入，依赖MongoDB唯一索引处理重复
func (p *PersistenceConsumer) handleArchiveMessage(msg *rest.WSMessage) error {
	log.Printf("执行消息归档: From=%d, To=%d, Content=%s, MessageID=%d",
		msg.From, msg.To, msg.Content, msg.MessageId)

	// 检查MessageID是否存在
	if msg.MessageId == 0 {
		log.Printf("归档消息MessageID为0，跳过归档: From=%d, To=%d", msg.From, msg.To)
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
	collection := p.db.GetCollection("messages")
	_, err := collection.InsertOne(context.Background(), message)

	// 检查错误类型
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			log.Printf("消息已归档(唯一索引幂等性保护): MessageID=%d", msg.MessageId)
			return nil // 幂等处理，返回成功
		}
		// 其他类型的数据库错误
		log.Printf("消息归档失败: %v", err)
		return err
	}

	log.Printf("消息归档成功: From=%d, To=%d, Status=已发送, MessageID=%d",
		msg.From, msg.To, msg.MessageId)

	return nil
}

// Stop 停止消费者
func (p *PersistenceConsumer) Stop() error {
	if p.consumer != nil {
		// TODO: 实现优雅停止
		log.Printf("持久化消费者停止")
	}
	return nil
}
