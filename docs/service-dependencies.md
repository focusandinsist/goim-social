# 微服务依赖关系和启动顺序

## 服务依赖关系图

```
基础服务层（无依赖）
├── User Service (21001/22001)     # 用户认证和管理
└── Group Service (21002/22002)    # 群组管理

业务服务层（依赖基础服务）
├── Friend Service (21003/22003)   # 依赖 User Service
└── Message Service (21004/22004)  # 依赖 User Service, Group Service

接入服务层（依赖业务服务）
└── Connect Service (21005/22005)  # 依赖 Message Service
```

## 端口分配规则

- **HTTP端口**: 从21001开始，按启动顺序递增
- **gRPC端口**: 从22001开始，按启动顺序递增
- **端口间隔**: 每个服务占用1个HTTP端口和1个gRPC端口

## 详细服务信息

### 1. User Service (优先级: 最高)
- **端口**: HTTP:21001, gRPC:22001
- **依赖**: 无
- **功能**: 用户注册、登录、认证、用户信息管理
- **被依赖**: Friend Service, Message Service
- **启动顺序**: 第1个启动

### 2. Group Service (优先级: 最高)
- **端口**: HTTP:21002, gRPC:22002
- **依赖**: 无
- **功能**: 群组创建、管理、成员管理
- **被依赖**: Message Service
- **启动顺序**: 第2个启动

### 3. Friend Service (优先级: 中等)
- **端口**: HTTP:21003, gRPC:22003
- **依赖**: User Service
- **功能**: 好友关系管理、好友请求处理
- **被依赖**: 无
- **启动顺序**: 第3个启动

### 4. Message Service (优先级: 中等)
- **端口**: HTTP:21004, gRPC:22004
- **依赖**: User Service, Group Service
- **功能**: 消息存储、历史消息查询、消息转发
- **被依赖**: Connect Service
- **启动顺序**: 第4个启动

### 5. Connect Service (优先级: 低)
- **端口**: HTTP:21005, gRPC:22005
- **依赖**: Message Service
- **功能**: WebSocket连接管理、实时消息推送
- **被依赖**: 无
- **启动顺序**: 第5个启动

## 启动方式

### 方式1: 使用批处理脚本（推荐）
```bash
# Windows
startServers.bat

# 脚本会按依赖顺序自动启动所有服务，并设置正确的端口号
```

### 方式2: 使用VSCode调试配置
1. 打开VSCode
2. 按F5选择"All Services (按依赖顺序启动)"
3. 或者选择单个服务进行调试

### 方式3: 手动启动（开发调试）
```bash
# 1. 启动User Service
set HTTP_PORT=21001&& set GRPC_PORT=22001&& go run apps/user-service/cmd/main.go

# 2. 启动Group Service  
set HTTP_PORT=21002&& set GRPC_PORT=22002&& go run apps/group/cmd/main.go

# 3. 启动Friend Service
set HTTP_PORT=21003&& set GRPC_PORT=22003&& go run apps/friend/cmd/main.go

# 4. 启动Message Service
set HTTP_PORT=21004&& set GRPC_PORT=22004&& go run apps/message/cmd/main.go

# 5. 启动Connect Service
set HTTP_PORT=21005&& set GRPC_PORT=22005&& go run apps/connect/cmd/main.go
```

## 健康检查端点

每个服务都提供健康检查端点：
- User Service: http://localhost:21001/health
- Group Service: http://localhost:21002/health
- Friend Service: http://localhost:21003/health
- Message Service: http://localhost:21004/health
- Connect Service: http://localhost:21005/health

## 服务间通信

服务间通过gRPC进行通信，通信端口：
- User Service gRPC: localhost:22001
- Group Service gRPC: localhost:22002
- Friend Service gRPC: localhost:22003
- Message Service gRPC: localhost:22004
- Connect Service gRPC: localhost:22005

## 注意事项

1. **启动顺序很重要**: 必须按依赖关系启动，否则依赖服务可能连接失败
2. **端口冲突**: 确保端口21001-21005和22001-22005没有被其他程序占用
3. **服务发现**: 目前使用硬编码的端口，后续可以集成服务发现机制
4. **健康检查**: 启动服务后建议检查健康检查端点确认服务正常运行
