package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/message-service/consumer"
	"goim-social/apps/message-service/handler"
	"goim-social/apps/message-service/service"
	"goim-social/pkg/middleware"
	"goim-social/pkg/server"
	"goim-social/pkg/telemetry"
)

func main() {
	serviceName := "message-service"

	// 初始化OpenTelemetry
	// 根据环境变量选择配置
	var otelConfig *telemetry.Config
	if os.Getenv("OTEL_DEBUG") == "true" {
		// 调试模式：输出到控制台
		otelConfig = telemetry.DevelopmentConfig(serviceName)
		log.Printf("OpenTelemetry debug mode enabled - traces will be printed to console")
	} else {
		// 默认模式：不输出，只记录到日志
		otelConfig = telemetry.DefaultConfig(serviceName)
		log.Printf("OpenTelemetry quiet mode - traces recorded but not printed")
	}

	if err := telemetry.InitGlobal(otelConfig); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}

	// 确保在程序退出时关闭OpenTelemetry
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.ShutdownGlobal(ctx); err != nil {
			log.Printf("Failed to shutdown OpenTelemetry: %v", err)
		}
	}()

	log.Printf("OpenTelemetry initialized for %s", serviceName)

	// 创建应用程序
	app := server.NewApplication(serviceName)

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 初始化Service层
	svc := service.NewService(app.GetMongoDB(), app.GetRedisClient(), app.GetKafkaProducer(), app.GetLogger())

	// 启动Kafka消费者
	ctx := context.Background()
	cfg := app.GetConfig()

	// 启动存储消费者（处理uplink_messages中的原始消息）
	storageConsumer := consumer.NewStorageConsumer(app.GetMongoDB())
	go func() {
		log.Println("启动存储消费者...")
		if err := storageConsumer.Start(ctx, cfg.Kafka.Brokers); err != nil {
			log.Fatalf("Failed to start storage consumer: %v", err)
		}
	}()

	// 启动持久化消费者（处理message_persistence_log中的归档命令）
	persistenceConsumer := consumer.NewPersistenceConsumer(app.GetMongoDB())
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

	// 创建OpenTelemetry中间件
	otelMW := middleware.NewOTelMiddleware(serviceName, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		// 添加OpenTelemetry中间件
		engine.Use(otelMW.GinMiddleware())

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
