package consumer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/proto"

	"goim-social/api/rest"
	"goim-social/apps/message-service/model"
	"goim-social/pkg/database"
	"goim-social/pkg/kafka"
	"goim-social/pkg/redis"
)

// PersistenceConsumer 专门的持久化消费者
// 职责：消费message_persistence_log Topic，执行消息归档
type PersistenceConsumer struct {
	db       *database.MongoDB
	redis    *redis.RedisClient
	consumer *kafka.Consumer
}

// NewPersistenceConsumer 创建持久化消费者
func NewPersistenceConsumer(db *database.MongoDB, redis *redis.RedisClient) *PersistenceConsumer {
	return &PersistenceConsumer{
		db:    db,
		redis: redis,
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

	// 幂等性检查：检查消息是否已处理
	ctx := context.Background()
	if p.isMessageProcessed(ctx, msg.Partition, msg.Offset) {
		log.Printf("归档命令已处理，跳过: partition=%d, offset=%d", msg.Partition, msg.Offset)
		return nil
	}

	// 解析protobuf消息事件
	var event rest.MessageEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("解析归档命令失败: %v, 原始消息: %s", err, string(msg.Value))
		return nil // 返回nil避免重试
	}

	log.Printf("解析归档命令成功: Type=%s, MessageID=%d", event.Type, event.Message.MessageId)

	// 根据事件类型处理
	switch event.Type {
	case "archive_message":
		// 处理消息归档命令
		if err := p.handleArchiveMessage(event.Message); err != nil {
			log.Printf("处理消息归档失败: %v", err)
			return nil // 返回nil避免重试
		}

		// 标记消息已处理
		if err := p.markMessageProcessed(ctx, msg.Partition, msg.Offset); err != nil {
			log.Printf("标记归档命令已处理失败: %v", err)
		}

		log.Printf("消息归档成功: MessageID=%d", event.Message.MessageId)
		return nil
	default:
		log.Printf("未知的归档命令类型: %s", event.Type)
		return nil
	}
}

// handleArchiveMessage 处理消息归档（经过Logic Service处理的标准格式）
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

	// 先尝试简单的插入操作，如果重复则忽略（幂等性保护）
	collection := p.db.GetCollection("messages")

	// 检查消息是否已存在
	var existingMsg model.Message
	err := collection.FindOne(context.Background(), bson.M{"message_id": msg.MessageId}).Decode(&existingMsg)
	if err == nil {
		// 消息已存在，跳过插入
		log.Printf("消息已归档(幂等性保护): From=%d, To=%d, MessageID=%d",
			msg.From, msg.To, msg.MessageId)
		return nil
	}

	// 消息不存在，执行插入
	result, err := collection.InsertOne(context.Background(), message)
	if err != nil {
		log.Printf("消息归档失败: %v", err)
		return err
	}

	log.Printf("消息归档成功: From=%d, To=%d, Status=已发送, MessageID=%d, InsertedID=%v",
		msg.From, msg.To, msg.MessageId, result.InsertedID)

	return nil
}

// isMessageProcessed 检查消息是否已处理（幂等性保护）
func (p *PersistenceConsumer) isMessageProcessed(ctx context.Context, partition int32, offset int64) bool {
	// 使用Redis检查消息是否已处理
	// Key格式: persistence:processed:{partition}:{offset}
	key := fmt.Sprintf("persistence:processed:%d:%d", partition, offset)

	count, err := p.redis.Exists(ctx, key)
	if err != nil {
		log.Printf("检查消息处理状态失败: %v, partition=%d, offset=%d", err, partition, offset)
		// 出错时返回false，允许重新处理（安全策略）
		return false
	}

	if count > 0 {
		log.Printf("消息已处理(幂等性保护): partition=%d, offset=%d", partition, offset)
		return true
	}

	return false
}

// markMessageProcessed 标记消息已处理
func (p *PersistenceConsumer) markMessageProcessed(ctx context.Context, partition int32, offset int64) error {
	// 使用Redis标记消息已处理
	// Key格式: persistence:processed:{partition}:{offset}
	key := fmt.Sprintf("persistence:processed:%d:%d", partition, offset)

	// 设置标记，过期时间为7天（与Kafka消息保留期一致）
	expiration := 7 * 24 * time.Hour

	err := p.redis.Set(ctx, key, "1", expiration)
	if err != nil {
		log.Printf("标记消息已处理失败: %v, partition=%d, offset=%d", err, partition, offset)
		return fmt.Errorf("标记消息已处理失败: %v", err)
	}

	log.Printf("消息已标记为已处理: partition=%d, offset=%d, 过期时间=%v", partition, offset, expiration)
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
