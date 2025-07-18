# goim-social

一个微服务架构的IM系统，更适用于社交场景


my notes:
| 对比点    | Kafka             | EventManager             |
| ------ | ----------------- | ------------------------ |
| 用途     | 跨实例消息队列、持久化、解耦异步流 | 当前进程内分发、调用业务逻辑           |
| 时效性    | 异步、消费延迟（取决于消费者处理） | 同步（或自己改成异步），实时调用 handler |
| 面向谁    | 服务节点之间            | 当前服务实例内 handler          |
| 注册的是什么 | 消费者订阅 topic       | handler 函数注册到事件名         |




架构在： https://chatgpt.com/c/67f62b23-48ec-8004-9ca6-5d80e946d86f
搜索：
    一个用户向指定用户发消息，是先走事件触发，然后在队列中排队，再被对方收到


// Handler: HTTP层面
func (h *Handler) CreateGroup(c *gin.Context) {
    // 1. 解析JSON参数
    // 2. 参数验证
    // 3. 调用service
    // 4. 返回HTTP响应
}

// Service: 业务层面  
func (s *Service) CreateGroup(ctx context.Context, name, desc string, ownerID int64) {
    // 1. 业务规则验证
    // 2. 调用DAO创建群组
    // 3. 发送通知
    // 4. 返回业务结果
}
Handler = HTTP适配器，Service = 业务引擎


【重要！！！】
如何界定：什么时候用 gRPC，什么时候用 MQ？
    非常好的问题！一句话总结：
同步请求 → 用 gRPC
异步通知 / 广播 / 削峰 → 用 MQ