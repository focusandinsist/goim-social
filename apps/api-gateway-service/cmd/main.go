package main

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/api-gateway-service/handler"
	"goim-social/apps/api-gateway-service/service"
	"goim-social/pkg/middleware"
	"goim-social/pkg/server"
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
		// 创建API网关中间件
		apiGatewayMW := middleware.NewAPIGatewayMiddleware(app.GetLogger(), "focusandinsist")

		// 注册中间件
		engine.Use(apiGatewayMW.GinLogging())  // 请求日志
		engine.Use(apiGatewayMW.GinRecovery()) // 恢复中间件
		engine.Use(apiGatewayMW.GinAuth())     // 认证中间件

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
