package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

// KafkaConfig 配置
type KafkaConfig struct {
	Brokers []string
	GroupID string
	Topics  []string
}

// RetryMessage 重试消息结构
type RetryMessage struct {
	Message     *sarama.ProducerMessage
	RetryCount  int
	LastAttempt time.Time
}

// Producer 生产者
type Producer struct {
	asyncProducer sarama.AsyncProducer
	retryQueue    chan *RetryMessage
	maxRetries    int
	retryDelay    time.Duration
}

// Consumer 消费者
type Consumer struct {
	group   sarama.ConsumerGroup
	topics  []string
	ready   chan bool
	Handler ConsumerHandler
}

type ConsumerHandler interface {
	HandleMessage(msg *sarama.ConsumerMessage) error
}

// InitProducer 初始化生产者
func InitProducer(brokers []string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Retry.Max = 3
	config.Producer.Retry.Backoff = 100 * time.Millisecond

	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	p := &Producer{
		asyncProducer: producer,
		retryQueue:    make(chan *RetryMessage, 1000), // 重试队列
		maxRetries:    5,                              // 最大重试次数
		retryDelay:    2 * time.Second,                // 重试延迟
	}

	// 启动错误处理和重试goroutine
	go p.handleErrors()

	// 启动成功监听goroutine
	go p.handleSuccesses()

	// 启动重试处理goroutine
	go p.handleRetries()

	return p, nil
}

// handleErrors 处理发送错误，将失败消息加入重试队列
func (p *Producer) handleErrors() {
	for err := range p.asyncProducer.Errors() {
		fmt.Printf("Kafka Producer错误: %v, topic=%s, partition=%d\n",
			err.Err, err.Msg.Topic, err.Msg.Partition)

		// 创建重试消息
		retryMsg := &RetryMessage{
			Message:     err.Msg,
			RetryCount:  0,
			LastAttempt: time.Now(),
		}

		// 加入重试队列
		select {
		case p.retryQueue <- retryMsg:
			fmt.Printf("消息已加入重试队列: topic=%s\n", err.Msg.Topic)
		default:
			fmt.Printf("重试队列已满，消息丢弃: topic=%s\n", err.Msg.Topic)
		}
	}
}

// handleSuccesses 处理发送成功
func (p *Producer) handleSuccesses() {
	for success := range p.asyncProducer.Successes() {
		fmt.Printf("Kafka消息发送成功: topic=%s, partition=%d, offset=%d\n",
			success.Topic, success.Partition, success.Offset)
	}
}

// handleRetries 处理重试队列
func (p *Producer) handleRetries() {
	for retryMsg := range p.retryQueue {
		// 检查是否超过最大重试次数
		if retryMsg.RetryCount >= p.maxRetries {
			fmt.Printf("消息重试次数超限，最终丢弃: topic=%s, retries=%d\n",
				retryMsg.Message.Topic, retryMsg.RetryCount)
			continue
		}

		// 等待重试延迟
		time.Sleep(p.retryDelay)

		// 增加重试次数
		retryMsg.RetryCount++
		retryMsg.LastAttempt = time.Now()

		fmt.Printf("重试发送消息: topic=%s, attempt=%d/%d\n",
			retryMsg.Message.Topic, retryMsg.RetryCount, p.maxRetries)

		// 重新发送消息
		p.asyncProducer.Input() <- retryMsg.Message
	}
}

// SendMessage 发送消息
func (p *Producer) SendMessage(topic string, key, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	fmt.Printf("准备发送消息到topic: %s, 消息大小: %d bytes\n", topic, len(value))

	// 发送消息到异步队列
	p.asyncProducer.Input() <- msg
	fmt.Printf("消息已提交到异步队列\n")

	return nil
}

// PublishMessage 发送protobuf消息
func (p *Producer) PublishMessage(topic string, data interface{}) error {
	// 如果是protobuf消息，直接序列化
	if msg, ok := data.(proto.Message); ok {
		protoData, err := proto.Marshal(msg)
		if err != nil {
			return fmt.Errorf("protobuf序列化失败: %v", err)
		}
		return p.SendMessage(topic, nil, protoData)
	}

	// TEMP兼容性：如果不是protobuf消息，仍使用json（主要用于非消息数据）
	// TODO 服务端全部protobuf，尤其是kafka这里
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	return p.SendMessage(topic, nil, jsonData)
}

// Close 关闭生产者
func (p *Producer) Close() error {
	// 关闭重试队列
	close(p.retryQueue)

	// 关闭异步生产者
	return p.asyncProducer.Close()
}

// GetRetryQueueSize 获取重试队列大小（用于监控）
func (p *Producer) GetRetryQueueSize() int {
	return len(p.retryQueue)
}

// InitConsumer 初始化消费者
func InitConsumer(cfg KafkaConfig, handler ConsumerHandler) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, config)
	if err != nil {
		return nil, err
	}
	c := &Consumer{
		group:   group,
		topics:  cfg.Topics,
		ready:   make(chan bool),
		Handler: handler,
	}
	return c, nil
}

// StartConsuming 启动消费
func (c *Consumer) StartConsuming(ctx context.Context) error {
	go func() {
		for {
			fmt.Printf("消费者开始新的消费循环...\n")
			if err := c.group.Consume(ctx, c.topics, c); err != nil {
				fmt.Printf("消费者错误: %v\n", err)
			}
			if ctx.Err() != nil {
				fmt.Printf("消费者上下文取消，退出\n")
				return
			}
			fmt.Printf("消费者循环结束，重新开始...\n")
		}
	}()
	<-c.ready
	fmt.Printf("消费者已准备就绪\n")
	return nil
}

// Setup sarama.ConsumerGroupHandler
func (c *Consumer) Setup(_ sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

// Cleanup sarama.ConsumerGroupHandler
func (c *Consumer) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费消息
func (c *Consumer) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	fmt.Printf("会话建立成功！开始监听分区 %d 的消息, 起始Offset: %d\n", claim.Partition(), claim.InitialOffset())

	for msg := range claim.Messages() {
		fmt.Printf("收到消息: partition=%d, offset=%d\n", msg.Partition, msg.Offset)

		if err := c.Handler.HandleMessage(msg); err == nil {
			sess.MarkMessage(msg, "")
			fmt.Printf("消息已标记为已处理: offset=%d\n", msg.Offset)
		} else {
			fmt.Printf("消息处理失败: %v, offset=%d\n", err, msg.Offset)
		}
	}

	fmt.Printf("分区 %d 的消息消费结束\n", claim.Partition())
	return nil
}
