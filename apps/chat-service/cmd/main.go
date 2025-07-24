package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"websocket-server/api/rest"
	"websocket-server/apps/chat-service/handler"
	"websocket-server/apps/chat-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("chat-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 获取配置
	config := app.GetConfig()

	// Group服务地址
	groupAddr := "localhost:22002" // group服务的gRPC端口
	if config.Chat.GroupService.Host != "" {
		groupAddr = fmt.Sprintf("%s:%d", config.Chat.GroupService.Host, config.Chat.GroupService.Port)
	}

	// Message服务地址
	messageAddr := "localhost:22004" // message服务的gRPC端口
	if config.Chat.MessageService.Host != "" {
		messageAddr = fmt.Sprintf("%s:%d", config.Chat.MessageService.Host, config.Chat.MessageService.Port)
	}

	// Friend服务地址
	friendAddr := "localhost:22003" // friend服务的gRPC端口

	// User服务地址
	userAddr := "localhost:22001" // user服务的gRPC端口

	// 初始化Service层
	svc, err := service.NewService(
		app.GetRedisClient(),
		app.GetKafkaProducer(),
		app.GetLogger(),
		groupAddr,
		messageAddr,
		friendAddr,
		userAddr,
	)
	if err != nil {
		panic("Failed to create chat service: " + err.Error())
	}

	// 初始化Handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterChatServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
