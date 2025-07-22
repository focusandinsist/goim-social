package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"websocket-server/api/rest"
	"websocket-server/apps/message-service/consumer"
	"websocket-server/apps/message-service/handler"
	"websocket-server/apps/message-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("message-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 初始化Service层
	svc := service.NewService(app.GetMongoDB(), app.GetRedisClient(), app.GetKafkaProducer())

	// 启动Kafka消费者
	ctx := context.Background()
	cfg := app.GetConfig()

	// 启动存储消费者
	storageConsumer := consumer.NewStorageConsumer(app.GetMongoDB(), app.GetRedisClient())
	go func() {
		log.Println("启动存储消费者...")
		if err := storageConsumer.Start(ctx, cfg.Kafka.Brokers); err != nil {
			log.Fatalf("Failed to start storage consumer: %v", err)
		}
	}()

	// 启动推送消费者
	pushConsumer := consumer.NewPushConsumer(app.GetRedisClient())
	go func() {
		log.Println("启动推送消费者...")
		if err := pushConsumer.Start(ctx, cfg.Kafka.Brokers); err != nil {
			log.Fatalf("Failed to start push consumer: %v", err)
		}
	}()

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler := handler.NewHandler(svc, app.GetLogger())
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		grpcService := svc.NewGRPCService(svc)
		rest.RegisterMessageServiceServer(grpcSrv, grpcService)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
