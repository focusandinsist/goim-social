# 离线消息功能实现总结

## 概述

本文档总结了IM系统中离线消息功能的完整实现过程，包括架构设计、问题分析、解决方案和最终的系统优化。

## 问题背景

### 初始问题
1. **历史消息推送方式不合理**：Connect服务通过gRPC主动推送历史消息
2. **消息状态缺失**：没有消息已读/未读状态管理
3. **架构混乱**：历史消息查询和实时推送职责不清
4. **双向通信问题**：用户A能给用户B发消息，但用户B无法给用户A发消息
5. **接口设计不统一**：混用GET和POST请求

### 核心需求
- 用户登录时自动获取未读消息（而非历史消息）
- 消息状态管理（未读/已读）
- 双向实时通信
- 清晰的架构分层
- 统一使用POST接口

## 架构优化

### 原始架构问题
```
客户端 → WebSocket → Connect服务 → gRPC推送历史消息 ❌
```

### 优化后的架构
```
# 未读消息获取
客户端 → HTTP POST → Message服务 → MongoDB查询未读消息 ✅

# 消息已读标记
客户端 → HTTP POST → Message服务 → MongoDB更新状态 ✅

# 实时消息推送
客户端 → WebSocket → Connect服务 → 双向流 → Message服务 → Kafka → 推送消费者 → 双向流 → Connect服务 → WebSocket → 客户端 ✅
```

## 核心实现

### 1. 消息状态管理

#### 数据模型优化
```go
type Message struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    From      int64              `bson:"from" json:"from"`
    To        int64              `bson:"to" json:"to"`
    GroupID   int64              `bson:"group_id" json:"group_id"`
    Content   string             `bson:"content" json:"content"`
    MsgType   int32              `bson:"msg_type" json:"msg_type"`
    AckID     string             `bson:"ack_id" json:"ack_id"`
    Status    int32              `bson:"status" json:"status"` // 0:未读 1:已读 2:撤回
    CreatedAt time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
```

#### 消息存储逻辑
```go
// 新消息默认状态为未读
message := &model.Message{
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
```

### 2. HTTP接口实现

#### Message服务HTTP接口
```go
// POST /api/v1/messages/unread - 获取未读消息
func (h *Handler) GetUnreadMessages(c *gin.Context) {
    var req struct {
        UserID int64 `json:"user_id" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
        return
    }

    // 调用service层获取未读消息
    messages, err := h.service.GetUnreadMessages(c.Request.Context(), req.UserID)

    c.JSON(http.StatusOK, gin.H{
        "messages": messages,
        "total":    len(messages),
    })
}

// POST /api/v1/messages/mark-read - 标记消息已读
func (h *Handler) MarkMessagesRead(c *gin.Context) {
    var req struct {
        UserID     int64    `json:"user_id" binding:"required"`
        MessageIDs []string `json:"message_ids" binding:"required"`
    }

    // 调用service层标记消息已读
    err := h.service.MarkMessagesAsRead(c.Request.Context(), req.UserID, req.MessageIDs)

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "标记已读成功",
    })
}
```

#### 客户端HTTP调用
```go
func fetchUnreadMessages(userID int64) {
    reqBody := UnreadRequest{UserID: userID}
    jsonData, _ := json.Marshal(reqBody)

    url := "http://localhost:21004/api/v1/messages/unread"
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))

    // 处理响应和显示未读消息
    // 自动标记为已读
    go markMessagesAsRead(userID, unreadMessages)
}
```

### 3. 双向流通信优化

#### Connect服务消息转发
```go
func (s *Service) ForwardMessageToMessageService(ctx context.Context, wsMsg *rest.WSMessage) error {
    // 优先使用双向流发送消息
    if s.msgStream != nil {
        return s.SendMessageViaStream(ctx, wsMsg)
    }
    
    // 备用：直接gRPC调用
    return s.sendViaDirectGRPC(ctx, wsMsg)
}
```

#### Proto定义扩展
```protobuf
message MessageStreamRequest {
  oneof request_type {
    SubscribeRequest subscribe = 1;
    MessageAckRequest ack = 2;
    PushResultRequest push_result = 3;
    SendWSMessageRequest send_message = 4; // 新增：发送消息
  }
}
```

#### Message服务双向流处理
```go
case *rest.MessageStreamRequest_SendMessage:
    // 处理通过双向流发送的消息
    sendReq := reqType.SendMessage
    log.Printf("📥 通过双向流接收消息: From=%d, To=%d", sendReq.Msg.From, sendReq.Msg.To)
    
    // 调用现有的SendWSMessage方法处理消息
    _, err := g.SendWSMessage(stream.Context(), sendReq)
```

## 关键问题解决

### 问题1：历史消息时间顺序错误

**问题**：历史消息按倒序显示
```go
// 错误的排序
Sort: map[string]interface{}{"created_at": -1} // 倒序
```

**解决**：改为正序排列
```go
// 正确的排序
Sort: map[string]interface{}{"created_at": 1} // 正序
```

### 问题2：双向通信失败

**问题**：用户A→B正常，B→A失败

**根因分析**：
1. Connect服务使用直接gRPC调用而非双向流
2. 推送消费者无法通过双向流推送消息
3. Redis在线用户状态管理问题

**解决方案**：
1. 优先使用双向流发送消息
2. 扩展proto定义支持双向流发送
3. 添加Redis调试日志
4. 完善在线用户状态管理

## 系统流程

### 消息发送流程
```
1. 客户端发送消息 → WebSocket → Connect服务
2. Connect服务 → 双向流 → Message服务
3. Message服务 → Kafka发布事件
4. 存储消费者 → MongoDB存储
5. 推送消费者 → 双向流 → Connect服务
6. Connect服务 → WebSocket → 目标客户端
```

### 未读消息获取流程
```
1. 客户端连接成功
2. 客户端 → HTTP POST → Message服务（获取未读消息）
3. Message服务 → MongoDB查询（status=0的消息）
4. 返回JSON格式未读消息
5. 客户端显示未读消息
6. 客户端 → HTTP POST → Message服务（标记已读）
7. Message服务 → MongoDB更新（status=1）
```

## 技术要点

### 1. 职责分离
- **WebSocket**：专注实时通信
- **HTTP**：专注数据查询
- **gRPC双向流**：服务间实时通信

### 2. 状态管理
- **MongoDB**：消息持久化和状态存储
- **Redis**：在线用户状态管理
- **内存**：WebSocket连接管理

### 3. 错误处理
- **重试机制**：Connect服务自动重连Message服务
- **备用方案**：双向流失败时使用直接gRPC调用
- **状态检查**：推送前检查用户在线状态

## 性能优化

### 1. 查询优化
```go
// 按时间正序查询，支持分页
cursor, err := collection.Find(ctx, filter, &options.FindOptions{
    Sort:  map[string]interface{}{"created_at": 1},
    Skip:  &skip,
    Limit: &limit,
})
```

### 2. 连接管理
```go
// 本地WebSocket连接缓存
type WSConnectionManager struct {
    localConnections map[int64]*websocket.Conn
    mutex            sync.RWMutex
}
```

### 3. 异步处理
```go
// Kafka异步消息处理
go storageConsumer.Start(ctx, cfg.Kafka.Brokers)
go pushConsumer.Start(ctx, cfg.Kafka.Brokers)
```

## 测试验证

### 功能测试
1. **未读消息**：用户登录自动获取未读消息，按时间正序显示
2. **消息状态**：获取后自动标记为已读，避免重复推送
3. **实时消息**：双向通信正常
4. **多用户**：支持多个用户同时在线聊天
5. **接口统一**：所有短连接请求均使用POST方法

### 性能测试
- **并发连接**：支持多个WebSocket连接
- **消息吞吐**：Kafka异步处理保证性能
- **查询效率**：MongoDB分页查询优化

## 总结

通过本次优化，IM系统实现了：

✅ **清晰的架构分层**：HTTP POST接口 + WebSocket实时通信
✅ **完整的消息状态管理**：未读/已读状态跟踪和自动标记
✅ **可靠的双向通信**：基于gRPC双向流的消息推送
✅ **高性能的消息处理**：Kafka异步处理 + MongoDB持久化
✅ **良好的用户体验**：自动未读消息获取 + 实时消息推送
✅ **统一的接口设计**：所有短连接请求使用POST方法
✅ **精准的离线消息**：只推送真正的未读消息，避免重复

系统现在具备了生产环境所需的稳定性、性能和可扩展性。
