package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"websocket-server/api/rest"
	"websocket-server/apps/logic-service/handler"
	"websocket-server/apps/logic-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("logic-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 获取配置
	config := app.GetConfig()

	// Group服务地址
	groupAddr := "localhost:22002" // group服务的gRPC端口
	if config.Logic.GroupService.Host != "" {
		groupAddr = fmt.Sprintf("%s:%d", config.Logic.GroupService.Host, config.Logic.GroupService.Port)
	}

	// Message服务地址
	messageAddr := "localhost:22004" // message服务的gRPC端口
	if config.Logic.MessageService.Host != "" {
		messageAddr = fmt.Sprintf("%s:%d", config.Logic.MessageService.Host, config.Logic.MessageService.Port)
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
		panic("Failed to create logic service: " + err.Error())
	}

	// 初始化Handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务 (暂时使用ChatService接口，后续会重命名)
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterChatServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
