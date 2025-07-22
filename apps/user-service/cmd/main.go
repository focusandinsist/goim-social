package main

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"websocket-server/api/rest"
	"websocket-server/apps/user-service/handler"
	"websocket-server/apps/user-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("user-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 初始化Service层
	svc := service.NewService(app.GetMongoDB(), app.GetRedisClient(), app.GetKafkaProducer())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler := handler.NewHandler(svc, app.GetLogger())
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		grpcService := svc.NewGRPCService(svc)
		rest.RegisterUserServiceServer(grpcSrv, grpcService)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
