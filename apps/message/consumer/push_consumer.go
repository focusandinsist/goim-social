package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"websocket-server/api/rest"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"

	"github.com/IBM/sarama"
)

// PushConsumer 推送消费者
type PushConsumer struct {
	consumer      *kafka.Consumer
	streamManager *StreamManager
	redis         *redis.RedisClient
}

// StreamManager 管理所有Connect服务的流连接
type StreamManager struct {
	streams map[string]*ConnectStream
	mutex   sync.RWMutex
}

// ConnectStream 存储Connect服务的流连接
type ConnectStream struct {
	ServiceID string
	Stream    rest.MessageService_MessageStreamServer
}

var globalStreamManager = &StreamManager{
	streams: make(map[string]*ConnectStream),
}

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
		Topics:  []string{"message-events"},
	}

	consumer, err := kafka.InitConsumer(cfg, p)
	if err != nil {
		return err
	}

	p.consumer = consumer
	log.Printf("✅ 推送消费者启动成功，监听topic: message-events")

	return p.consumer.StartConsuming(ctx)
}

// HandleMessage 实现 kafka.ConsumerHandler 接口
func (p *PushConsumer) HandleMessage(msg *sarama.ConsumerMessage) error {
	log.Printf("📥 推送消费者收到消息: topic=%s, partition=%d, offset=%d",
		msg.Topic, msg.Partition, msg.Offset)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ 推送消费者处理消息时发生panic: %v", r)
		}
	}()

	// 幂等性检查：检查推送是否已处理
	ctx := context.Background()
	if p.isPushProcessed(ctx, msg.Partition, msg.Offset) {
		log.Printf("✅ 推送已处理，跳过: partition=%d, offset=%d", msg.Partition, msg.Offset)
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
		if err := p.handleNewMessage(event.Message); err != nil {
			log.Printf("❌ 处理新消息失败: %v", err)
			return nil // 返回nil避免重试
		}

		// 标记推送已处理
		if err := p.markPushProcessed(ctx, msg.Partition, msg.Offset); err != nil {
			log.Printf("❌ 标记推送已处理失败: %v", err)
		}

		return nil
	default:
		log.Printf("⚠️  未知的消息事件类型: %s", event.Type)
		return nil
	}
}

// isPushProcessed 检查推送是否已处理（幂等性检查）
func (p *PushConsumer) isPushProcessed(ctx context.Context, partition int32, offset int64) bool {
	key := fmt.Sprintf("kafka:push:%d:%d", partition, offset)
	exists, err := p.redis.Exists(ctx, key)
	if err != nil {
		log.Printf("❌ 检查推送处理状态失败: %v", err)
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
		log.Printf("❌ MessageID为0，跳过推送: From=%d, To=%d, Content=%s", msg.From, msg.To, msg.Content)
		return fmt.Errorf("MessageID不能为0")
	}

	if msg.To > 0 {
		// 单聊消息：推送给目标用户
		log.Printf("📤 推送单聊消息: From=%d, To=%d, Content=%s, MessageID=%d", msg.From, msg.To, msg.Content, msg.MessageId)
		p.streamManager.PushToAllStreams(msg.To, msg)
	} else if msg.GroupId > 0 {
		// 群聊消息：需要查询群成员并推送给所有成员
		log.Printf("📤 推送群聊消息: From=%d, GroupID=%d, Content=%s, MessageID=%d", msg.From, msg.GroupId, msg.Content, msg.MessageId)
		// TODO: 查询群成员列表，推送给所有成员
		// 这里简化处理，假设群成员ID为1,2,3
		groupMembers := []int64{1, 2, 3}
		for _, memberID := range groupMembers {
			if memberID != msg.From { // 不推送给发送者自己
				p.streamManager.PushToAllStreams(memberID, msg)
			}
		}
	}

	return nil
}

// AddStream 添加Connect服务流连接
func (sm *StreamManager) AddStream(serviceID string, stream rest.MessageService_MessageStreamServer) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.streams[serviceID] = &ConnectStream{
		ServiceID: serviceID,
		Stream:    stream,
	}
	log.Printf("✅ 添加Connect服务流连接: %s", serviceID)
}

// RemoveStream 移除Connect服务流连接
func (sm *StreamManager) RemoveStream(serviceID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.streams, serviceID)
	log.Printf("🗑️  移除Connect服务流连接: %s", serviceID)
}

// PushToAllStreams 推送消息到所有Connect服务
func (sm *StreamManager) PushToAllStreams(targetUserID int64, message *rest.WSMessage) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	for serviceID, connectStream := range sm.streams {
		go func(sid string, stream rest.MessageService_MessageStreamServer) {
			err := stream.Send(&rest.MessageStreamResponse{
				ResponseType: &rest.MessageStreamResponse_PushEvent{
					PushEvent: &rest.MessagePushEvent{
						TargetUserId: targetUserID,
						Message:      message,
						EventType:    "new_message",
					},
				},
			})
			if err != nil {
				log.Printf("❌ 推送消息到Connect服务 %s 失败: %v", sid, err)
				// 如果推送失败，移除这个连接
				sm.RemoveStream(sid)
			} else {
				log.Printf("✅ 成功推送消息到Connect服务 %s, 目标用户: %d", sid, targetUserID)
			}
		}(serviceID, connectStream.Stream)
	}
}

// GetStreamManager 获取全局流管理器
func GetStreamManager() *StreamManager {
	return globalStreamManager
}

// Stop 停止消费者
func (p *PushConsumer) Stop() error {
	if p.consumer != nil {
		// TODO: 实现优雅停止
		log.Printf("🛑 推送消费者停止")
	}
	return nil
}
