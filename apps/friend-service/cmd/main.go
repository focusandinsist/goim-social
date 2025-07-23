package main

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"websocket-server/api/rest"
	"websocket-server/apps/friend-service/dao"
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
	messageClient := rest.NewFriendEventServiceClient(conn)

	// 初始化DAO层
	friendDAO := dao.NewFriendDAO(app.GetMongoDB())

	// 初始化Service层
	svc := service.NewService(friendDAO, app.GetRedisClient(), app.GetKafkaProducer(), messageClient)

	// 初始化Handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterFriendEventServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
