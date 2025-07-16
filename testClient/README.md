# WebSocket 测试客户端

这个测试客户端用于测试WebSocket消息服务的功能。

## 功能特性

- 支持多用户同时连接
- 支持手动发送消息
- 支持自动模式发送消息
- 实时接收其他用户发送的消息
- 使用protobuf协议通信

## 使用方法

### 1. 启动多个客户端（推荐）

```bash
# Windows
startClients.bat

# 手动启动
go run main.go -user=1001 -target=1002
go run main.go -user=1002 -target=1001
```

### 2. 启动单个客户端

```bash
# 使用默认参数
test-single.bat

# 指定用户ID
test-single.bat 1001 1002

# 直接使用go run
go run main.go -user=1001 -target=1002
```

### 3. 命令行参数

- `-user`: 当前用户ID（默认：1001）
- `-target`: 目标用户ID（默认：1002）
- `-url`: WebSocket服务地址（默认：ws://localhost:21005/api/v1/connect/ws）
- `-token`: 认证token（默认：auth-debug）
- `-auto`: 自动模式，每5秒发送一条消息

### 4. 使用示例

```bash
# 基本使用
go run main.go -user=1001 -target=1002

# 自动模式
go run main.go -user=1004 -target=1001 -auto

# 自定义服务地址
go run main.go -user=1001 -target=1002 -url=ws://localhost:8080/ws
```

## 测试场景

### 场景1：双向聊天
1. 启动用户1001：`go run main.go -user=1001 -target=1002`
2. 启动用户1002：`go run main.go -user=1002 -target=1001`
3. 在任一客户端输入消息，另一客户端应该能收到

### 场景2：群聊模拟
1. 启动多个客户端，都发送给同一个目标用户
2. 目标用户应该能收到所有消息

### 场景3：自动压测
1. 启动多个自动模式客户端：`go run main.go -user=100X -target=1001 -auto`
2. 观察消息发送和接收情况

## 注意事项

1. **服务启动顺序**：确保按以下顺序启动服务
   - User Service (21001/22001)
   - Group Service (21002/22002)
   - Message Service (21004/22004)
   - Connect Service (21005/22005)

2. **User-ID Header**：客户端会自动在WebSocket连接时发送User-ID header

3. **消息格式**：使用protobuf格式，消息类型为1表示文本消息

4. **连接认证**：需要提供Authorization header（当前使用debug token）

## 故障排除

### 连接失败
- 检查Connect服务是否启动（端口21005）
- 检查防火墙设置
- 确认WebSocket URL正确

### 消息收不到
- 检查Message服务是否启动（端口22004）
- 查看服务日志确认gRPC双向流是否建立
- 确认目标用户ID正确

### 认证失败
- 检查Authorization header是否正确
- 确认User-ID header格式正确（数字）
