package main

import (
	"github.com/gin-gonic/gin"
	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"

	"websocket-server/api/rest"
	"websocket-server/apps/connect-service/handler"
	"websocket-server/apps/connect-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("connect-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 初始化Service层
	svc := service.NewService(app.GetMongoDB(), app.GetRedisClient(), app.GetKafkaProducer(), app.GetConfig())

	// 创建handler
	httpHandler := handler.NewHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		grpcService := httpHandler.NewGRPCService()
		rest.RegisterConnectServiceServer(grpcSrv, grpcService)
	})

	// 启动与Message服务的gRPC双向流连接
	go func() {
		app.GetKratosLogger().Log(kratoslog.LevelInfo, "msg", "Starting message stream connection")
		svc.StartMessageStream()
	}()

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
