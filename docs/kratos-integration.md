# Kratos框架集成说明

本项目已集成Kratos框架，用于统一管理HTTP和gRPC服务的初始化、配置和生命周期。

## 主要特性

### 1. 统一配置管理
- 支持环境变量配置
- 自动端口分配（HTTP端口对应gRPC端口+12000）
- 统一的数据库、Redis、Kafka配置

### 2. 服务器封装
- HTTP服务器：基于Gin，支持CORS、健康检查
- gRPC服务器：原生gRPC服务器封装
- 优雅启动和关闭

### 3. 端口规范（按依赖关系排序）
| 启动顺序 | 服务 | HTTP端口 | gRPC端口 | 依赖关系 | 说明 |
|----------|------|----------|----------|----------|------|
| 1 | User Service | 21001 | 22001 | 无 | 用户服务（基础服务） |
| 2 | Group Service | 21002 | 22002 | 无 | 群组服务（基础服务） |
| 3 | Friend Service | 21003 | 22003 | User | 好友服务（依赖用户服务） |
| 4 | Message Service | 21004 | 22004 | User, Group | 消息服务（依赖用户、群组服务） |
| 5 | Connect Service | 21005 | 22005 | Message | WebSocket连接服务（依赖消息服务） |

## 使用方法

### 1. 环境变量配置

每个服务支持以下环境变量：

```bash
# 服务端口（User Service示例）
HTTP_PORT=21001
GRPC_PORT=22001

# 数据库配置
MONGODB_URI=mongodb://localhost:27017
MONGODB_DB=userDB

# Redis配置
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Kafka配置
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=user-service-group
```

### 2. VSCode调试配置

已在`launch.json`中配置好各个服务的调试环境，可以：

1. **单独调试某个服务**：选择对应的配置启动
2. **同时启动多个服务**：使用"All Services"或"Core Services"组合配置

### 3. 服务结构

每个服务的main.go结构：

```go
func main() {
    // 1. 加载配置
    cfg := config.LoadConfig("service-name")
    
    // 2. 初始化日志
    kratosLogger := kratoslog.With(kratoslog.NewStdLogger(os.Stdout),
        "service.name", "service-name",
        "service.version", "v1.0.0",
    )
    
    // 3. 初始化依赖（数据库、Redis、Kafka）
    
    // 4. 创建服务器
    httpServer := server.NewHTTPServerWrapper(cfg, kratosLogger)
    
    // 5. 注册路由
    handler.RegisterRoutes(httpServer.GetEngine())
    
    // 6. 启动服务器
    
    // 7. 优雅关闭
}
```

## 迁移指南

### 已迁移的服务
- ✅ Message Service (HTTP + gRPC)
- ✅ User Service (HTTP only)

### 待迁移的服务
- ⏳ Connect Service
- ⏳ Group Service  
- ⏳ Friend Service

### 迁移步骤

1. **修改main.go**：
   - 导入config和server包
   - 使用LoadConfig加载配置
   - 使用HTTPServerWrapper创建HTTP服务器
   - 添加优雅关闭逻辑

2. **更新端口配置**：
   - 从环境变量读取端口
   - 确保gRPC端口符合20000+规范

3. **测试验证**：
   - 确保HTTP接口正常工作
   - 确保gRPC接口正常工作
   - 验证优雅关闭功能

## 健康检查

每个HTTP服务都自动提供健康检查端点：

```bash
GET /health
```

响应：
```json
{
  "status": "ok",
  "time": 1640995200
}
```

## 下一步计划

1. 完成所有服务的Kratos集成
2. 添加服务发现和注册
3. 集成分布式追踪
4. 添加Prometheus监控指标
5. 实现配置热重载
