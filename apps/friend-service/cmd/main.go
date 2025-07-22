package main

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	friendpb "websocket-server/api/rest"
	"websocket-server/apps/friend-service/handler"
	"websocket-server/apps/friend-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("friend-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 初始化gRPC客户端，并连接到Message服务
	messageGrpcAddr := "localhost:22004"
	conn, err := grpc.NewClient(messageGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	messageClient := friendpb.NewFriendEventServiceClient(conn)

	// 初始化Service层
	svc := service.NewService(app.GetMongoDB(), app.GetRedisClient(), app.GetKafkaProducer(), messageClient)

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler := handler.NewHandler(svc, app.GetLogger())
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		grpcService := svc.NewGRPCService(svc)
		friendpb.RegisterFriendEventServiceServer(grpcSrv, grpcService)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
