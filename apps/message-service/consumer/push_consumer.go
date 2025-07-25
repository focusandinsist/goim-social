package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"

	"websocket-server/api/rest"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"
)

// PushConsumer 推送消费者
type PushConsumer struct {
	consumer      *kafka.Consumer
	streamManager *StreamManager
	redis         *redis.RedisClient
}

// StreamManager 管理所有Connect服务的连接（已废弃双向流）
type StreamManager struct {
	// 保留结构体以避免编译错误，但不再使用双向流
	mutex sync.RWMutex
}

var globalStreamManager = &StreamManager{}

// NewPushConsumer 创建推送消费者
func NewPushConsumer(redis *redis.RedisClient) *PushConsumer {
	return &PushConsumer{
		streamManager: globalStreamManager,
		redis:         redis,
	}
}

// Start 启动推送消费者
func (p *PushConsumer) Start(ctx context.Context, brokers []string) error {
	cfg := kafka.KafkaConfig{
		Brokers: brokers,
		GroupID: "push-consumer-group",
		Topics:  []string{"downlink_messages"},
	}

	consumer, err := kafka.InitConsumer(cfg, p)
	if err != nil {
		return err
	}

	p.consumer = consumer
	log.Printf("推送消费者启动成功，监听topic: downlink_messages")

	return p.consumer.StartConsuming(ctx)
}

// HandleMessage 实现 kafka.ConsumerHandler 接口
func (p *PushConsumer) HandleMessage(msg *sarama.ConsumerMessage) error {
	log.Printf("推送消费者收到消息: topic=%s, partition=%d, offset=%d",
		msg.Topic, msg.Partition, msg.Offset)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("推送消费者处理消息时发生panic: %v", r)
		}
	}()

	// 幂等性检查：检查推送是否已处理
	ctx := context.Background()
	if p.isPushProcessed(ctx, msg.Partition, msg.Offset) {
		log.Printf("推送已处理，跳过: partition=%d, offset=%d", msg.Partition, msg.Offset)
		return nil
	}

	// 解析消息事件
	var event MessageEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("解析消息事件失败: %v, 原始消息: %s", err, string(msg.Value))
		return nil // 返回nil避免重试
	}

	// 根据事件类型处理
	switch event.Type {
	case "new_message":
		if err := p.handleNewMessage(event.Message); err != nil {
			log.Printf("处理新消息失败: %v", err)
			return nil // 返回nil避免重试
		}

		// 标记推送已处理
		if err := p.markPushProcessed(ctx, msg.Partition, msg.Offset); err != nil {
			log.Printf("标记推送已处理失败: %v", err)
		}

		return nil
	default:
		log.Printf("未知的消息事件类型: %s", event.Type)
		return nil
	}
}

// isPushProcessed 检查推送是否已处理（幂等性检查）
func (p *PushConsumer) isPushProcessed(ctx context.Context, partition int32, offset int64) bool {
	key := fmt.Sprintf("kafka:push:%d:%d", partition, offset)
	exists, err := p.redis.Exists(ctx, key)
	if err != nil {
		log.Printf("检查推送处理状态失败: %v", err)
		return false // 出错时假设未处理，允许重试
	}
	return exists > 0 // Redis Exists返回存在的key数量
}

// markPushProcessed 标记推送已处理
func (p *PushConsumer) markPushProcessed(ctx context.Context, partition int32, offset int64) error {
	key := fmt.Sprintf("kafka:push:%d:%d", partition, offset)
	return p.redis.Set(ctx, key, "processed", time.Hour) // 1小时过期
}

// handleNewMessage 处理新消息推送
func (p *PushConsumer) handleNewMessage(msg *rest.WSMessage) error {
	// 检查MessageID是否存在
	if msg.MessageId == 0 {
		log.Printf("MessageID为0，跳过推送: From=%d, To=%d, Content=%s", msg.From, msg.To, msg.Content)
		return fmt.Errorf("MessageID不能为0")
	}

	if msg.To > 0 {
		// 单聊消息：推送给目标用户
		log.Printf("推送单聊消息: From=%d, To=%d, Content=%s, MessageID=%d", msg.From, msg.To, msg.Content, msg.MessageId)
		if err := p.pushToConnectService(msg.To, msg); err != nil {
			log.Printf("推送消息到Connect服务失败: %v", err)
		}
	} else if msg.GroupId > 0 {
		// 群聊消息：推送给所有群成员（已在Logic服务中处理扇出）
		log.Printf("推送群聊消息: From=%d, GroupID=%d, Content=%s, MessageID=%d", msg.From, msg.GroupId, msg.Content, msg.MessageId)
		// 注意：群聊消息的推送已经在Logic服务中通过消息队列扇出到各个成员
		// 这里接收到的应该是针对特定用户的消息，直接推送即可
		// 如果To字段有值，说明是扇出后的单个用户消息
		if msg.To > 0 {
			if err := p.pushToConnectService(msg.To, msg); err != nil {
				log.Printf("推送群聊消息到Connect服务失败: %v", err)
			}
		} else {
			log.Printf("群聊消息缺少目标用户ID，跳过推送: GroupID=%d, MessageID=%d", msg.GroupId, msg.MessageId)
		}
	}

	return nil
}

// // AddStream 添加Connect服务流连接
// func (sm *StreamManager) AddStream(serviceID string, stream rest.MessageService_MessageStreamServer) {
// 	sm.mutex.Lock()
// 	defer sm.mutex.Unlock()
// 	sm.streams[serviceID] = &ConnectStream{
// 		ServiceID: serviceID,
// 		Stream:    stream,
// 	}
// 	log.Printf("添加Connect服务流连接: %s", serviceID)
// }

// RemoveStream 移除Connect服务连接（已废弃）
func (sm *StreamManager) RemoveStream(serviceID string) {
	log.Printf("RemoveStream已废弃，不再使用双向流: %s", serviceID)
}

// PushToAllStreams 推送消息到所有Connect服务（已废弃，改用消息队列）
func (sm *StreamManager) PushToAllStreams(targetUserID int64, message *rest.WSMessage) {
	log.Printf("PushToAllStreams已废弃，请使用消息队列进行消息推送: UserID=%d", targetUserID)
}

// pushToConnectService 通过Redis发布消息到Connect服务
func (p *PushConsumer) pushToConnectService(targetUserID int64, message *rest.WSMessage) error {
	ctx := context.Background()

	// 查找用户所在的Connect实例
	pattern := fmt.Sprintf("conn:%d:*", targetUserID)
	keys, err := p.redis.Keys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("查找用户连接失败: %v", err)
	}

	if len(keys) == 0 {
		log.Printf("用户 %d 不在线，跳过推送", targetUserID)
		return nil
	}

	// 获取用户连接信息
	connInfo, err := p.redis.HGetAll(ctx, keys[0])
	if err != nil {
		return fmt.Errorf("获取连接信息失败: %v", err)
	}

	serverID, exists := connInfo["serverID"]
	if !exists {
		return fmt.Errorf("连接信息中缺少serverID")
	}

	// 构造推送消息
	pushMsg := map[string]interface{}{
		"type":        "push_message",
		"target_user": targetUserID,
		"message":     message,
		"timestamp":   time.Now().Unix(),
	}

	// 序列化消息
	msgBytes, err := json.Marshal(pushMsg)
	if err != nil {
		return fmt.Errorf("序列化推送消息失败: %v", err)
	}

	// 发布到Connect服务的频道
	channel := fmt.Sprintf("connect_forward:%s", serverID)
	if err := p.redis.Publish(ctx, channel, string(msgBytes)); err != nil {
		return fmt.Errorf("发布推送消息失败: %v", err)
	}

	log.Printf("已发布推送消息到Connect服务: ServerID=%s, UserID=%d, MessageID=%d",
		serverID, targetUserID, message.MessageId)
	return nil
}

// GetStreamManager 获取全局流管理器
func GetStreamManager() *StreamManager {
	return globalStreamManager
}

// Stop 停止消费者
func (p *PushConsumer) Stop() error {
	if p.consumer != nil {
		// TODO: 实现优雅停止
		log.Printf("推送消费者停止")
	}
	return nil
}
