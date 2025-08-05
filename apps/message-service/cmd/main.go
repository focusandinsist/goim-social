package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/message-service/consumer"
	"goim-social/apps/message-service/handler"
	"goim-social/apps/message-service/service"
	"goim-social/pkg/server"
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

	// 启动存储消费者（处理uplink_messages中的原始消息）
	storageConsumer := consumer.NewStorageConsumer(app.GetMongoDB(), app.GetRedisClient())
	go func() {
		log.Println("启动存储消费者...")
		if err := storageConsumer.Start(ctx, cfg.Kafka.Brokers); err != nil {
			log.Fatalf("Failed to start storage consumer: %v", err)
		}
	}()

	// 启动持久化消费者（处理message_persistence_log中的归档命令）
	persistenceConsumer := consumer.NewPersistenceConsumer(app.GetMongoDB(), app.GetRedisClient())
	go func() {
		log.Println("启动持久化消费者...")
		if err := persistenceConsumer.Start(ctx, cfg.Kafka.Brokers); err != nil {
			log.Fatalf("Failed to start persistence consumer: %v", err)
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
		httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		grpcHandler := handler.NewGRPCHandler(svc)
		rest.RegisterMessageServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
