package main

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"websocket-server/api/rest"
	"websocket-server/apps/api-gateway-service/handler"
	"websocket-server/apps/api-gateway-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("api-gateway-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 初始化Service层
	svc := service.NewService(app.GetMongoDB(), app.GetRedisClient(), app.GetKafkaProducer(), app.GetConfig())

	// 创建各handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		engine.Use(gin.Logger())   // 请求日志
		engine.Use(gin.Recovery()) // 恢复中间件

		// TODO:认证中间件移到pkg里
		engine.Use(func(c *gin.Context) {
			// 跳过认证
			if c.Request.URL.Path == "/api/v1/gateway/health" ||
				c.Request.URL.Path == "/api/v1/gateway/services" ||
				c.Request.URL.Path == "/api/v1/gateway/online_status" ||
				c.Request.URL.Path == "/api/v1/gateway/online_count" ||
				c.Request.URL.Path == "/api/v1/gateway/online_users" ||
				c.Request.URL.Path == "/api/v1/gateway/user_connections" {
				c.Next()
				return
			}
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(401, map[string]interface{}{"error": "Missing authorization header"})
				c.Abort()
				return
			}
			c.Next()
		})

		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterConnectServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
