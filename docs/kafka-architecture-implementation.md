# Kafka异步架构实现文档

## 架构概述

我们已经成功将同步的消息处理架构改进为基于Kafka的异步解耦架构：

### 原始架构（同步）
```
A → Connect-A → gRPC → Message Service
                        ├─ 存储 (同步)
                        └─ gRPC流推送 → Connect-B → WebSocket → B
```

### 新架构（异步解耦）
```
A → Connect-A → gRPC → Message Service
                        └─ 写入Kafka
                              ├─ 存储消费者 → MongoDB
                              └─ 推送消费者 → gRPC流 → Connect-B → WebSocket → B
```

## 核心改进

### 1. Message Service 简化
**修改文件**: `apps/message/service/service.go`

**核心变化**:
- `SendWSMessage` 方法只负责发布消息到Kafka
- 移除了同步存储和推送逻辑
- 大幅提升响应速度

```go
func (g *GRPCService) SendWSMessage(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
    // 发布消息到Kafka（异步处理）
    messageEvent := map[string]interface{}{
        "type":      "new_message",
        "message":   req.Msg,
        "timestamp": time.Now().Unix(),
    }
    
    if err := g.svc.kafka.PublishMessage("message-events", messageEvent); err != nil {
        return &rest.SendWSMessageResponse{Success: false, Message: "消息发送失败"}, err
    }
    
    return &rest.SendWSMessageResponse{Success: true, Message: "消息发送成功"}, nil
}
```

### 2. Kafka Producer 增强
**修改文件**: `pkg/kafka/kafka.go`

**新增功能**:
- `PublishMessage` 方法支持JSON消息发布
- 自动序列化复杂数据结构

```go
func (p *Producer) PublishMessage(topic string, data interface{}) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("JSON序列化失败: %v", err)
    }
    return p.SendMessage(topic, nil, jsonData)
}
```

### 3. 存储消费者
**新增文件**: `apps/message/consumer/storage_consumer.go`

**职责**:
- 监听 `message-events` topic
- 异步存储消息到MongoDB
- 独立扩展和故障隔离

**关键特性**:
- 消费者组: `storage-consumer-group`
- 自动重试和错误处理
- 详细的日志记录

### 4. 推送消费者
**新增文件**: `apps/message/consumer/push_consumer.go`

**职责**:
- 监听 `message-events` topic
- 通过gRPC流推送消息到Connect服务
- 管理Connect服务的流连接

**关键特性**:
- 消费者组: `push-consumer-group`
- 全局StreamManager管理
- 支持单聊和群聊消息推送

### 5. 启动流程优化
**修改文件**: `apps/message/cmd/main.go`

**新增启动逻辑**:
```go
// 启动存储消费者
storageConsumer := consumer.NewStorageConsumer(mongoDB)
go func() {
    if err := storageConsumer.Start(ctx, cfg.Kafka.Brokers); err != nil {
        log.Fatalf("Failed to start storage consumer: %v", err)
    }
}()

// 启动推送消费者
pushConsumer := consumer.NewPushConsumer()
go func() {
    if err := pushConsumer.Start(ctx, cfg.Kafka.Brokers); err != nil {
        log.Fatalf("Failed to start push consumer: %v", err)
    }
}()
```

## Kafka Topic 设计

### message-events
**用途**: 新消息事件
**消息格式**:
```json
{
  "type": "new_message",
  "message": {
    "from": 1001,
    "to": 1002,
    "content": "hello",
    "message_type": 1,
    "timestamp": 1642678800
  },
  "timestamp": 1642678800
}
```

**消费者组**:
- `storage-consumer-group`: 负责消息存储
- `push-consumer-group`: 负责消息推送

## 架构优势

### 1. 性能提升
- **异步处理**: 消息发送立即返回，不等待存储和推送
- **并行处理**: 存储和推送可以并行执行
- **高吞吐量**: Kafka支持大量并发消息

### 2. 可靠性增强
- **消息持久化**: Kafka保证消息不丢失
- **故障隔离**: 存储失败不影响推送，推送失败不影响存储
- **自动重试**: 消费者可以重试失败的操作

### 3. 扩展性改善
- **水平扩展**: 可以启动多个消费者实例
- **负载均衡**: Kafka自动分配消息给不同消费者
- **服务解耦**: 各组件可以独立扩展和部署

### 4. 运维友好
- **监控完善**: Kafka提供丰富的监控指标
- **故障恢复**: 消费者重启后从上次位置继续
- **流量控制**: 可以控制消费速度

## 部署和测试

### 1. 启动顺序
```bash
# 1. 启动基础服务
docker-compose up -d kafka mongodb redis

# 2. 启动Message服务（包含消费者）
go run apps/message/cmd/main.go

# 3. 启动Connect服务
go run apps/connect/cmd/main.go

# 4. 测试客户端
cd testClient && go run main.go -user=1001 -target=1002
```

### 2. 监控要点
- Kafka消费者延迟
- 消息处理成功率
- 存储和推送的独立性能指标

### 3. 故障测试
- 存储服务故障时推送是否正常
- 推送服务故障时存储是否正常
- Kafka重启后消费者恢复情况

## 后续扩展

### 1. 新增消费者
可以轻松添加新的消费者来处理其他业务逻辑：
- 消息审核消费者
- 数据分析消费者
- 通知推送消费者

### 2. 多Topic支持
可以根据业务需要添加更多Topic：
- `user-events`: 用户状态变更
- `group-events`: 群组管理事件
- `system-events`: 系统通知事件

### 3. 消息路由
可以实现更复杂的消息路由逻辑：
- 基于用户类型的路由
- 基于消息内容的路由
- 基于地理位置的路由

## 总结

通过引入Kafka异步架构，我们实现了：
- **更高的性能**: 异步处理提升响应速度
- **更好的可靠性**: 故障隔离和消息持久化
- **更强的扩展性**: 独立扩展各个组件
- **更简单的维护**: 清晰的职责分离

这种架构特别适合高并发的即时通讯场景，为后续的功能扩展和性能优化奠定了坚实的基础。
