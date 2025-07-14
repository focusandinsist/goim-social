package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
)

// KafkaConfig 配置
type KafkaConfig struct {
	Brokers []string
	GroupID string
	Topics  []string
}

// Producer 生产者
type Producer struct {
	asyncProducer sarama.AsyncProducer
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
	config.Producer.Partitioner = sarama.NewHashPartitioner
	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Producer{asyncProducer: producer}, nil
}

// SendMessage 发送消息
func (p *Producer) SendMessage(topic string, key, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
	p.asyncProducer.Input() <- msg
	return nil
}

// Close 关闭生产者
func (p *Producer) Close() error {
	return p.asyncProducer.Close()
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
			if err := c.group.Consume(ctx, c.topics, c); err != nil {
				fmt.Printf("Error from consumer: %v\n", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
	<-c.ready
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
	for msg := range claim.Messages() {
		if err := c.Handler.HandleMessage(msg); err == nil {
			sess.MarkMessage(msg, "")
		}
	}
	return nil
}
