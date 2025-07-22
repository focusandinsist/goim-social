package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	friendpb "websocket-server/api/rest"
	"websocket-server/apps/friend/handler"
	"websocket-server/apps/friend/service"
	"websocket-server/pkg/config"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/redis"
	"websocket-server/pkg/server"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig("friend-service")

	// 初始化Kratos日志
	kratosLogger := kratoslog.With(kratoslog.NewStdLogger(os.Stdout),
		"service.name", "friend-service",
		"service.version", "v1.0.0",
	)

	// 初始化原有日志系统
	if err := logger.Init("info"); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	loggerInstance := logger.GetLogger()

	// 初始化 MongoDB
	mongoDB, err := database.NewMongoDB(cfg.Database.MongoDB.URI, cfg.Database.MongoDB.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// 初始化 Redis
	redisClient := redis.NewRedisClient(cfg.Redis.Addr)

	// 初始化 Kafka
	kafkaProducer, err := kafka.InitProducer(cfg.Kafka.Brokers)
	if err != nil {
		log.Fatalf("Failed to connect to Kafka: %v", err)
	}

	// 初始化 gRPC 客户端 - 连接到Message服务
	messageGrpcAddr := "localhost:22004" // Message服务的gRPC端口
	conn, err := grpc.NewClient(messageGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to message gRPC: %v", err)
	}
	defer conn.Close()
	messageClient := friendpb.NewFriendEventServiceClient(conn)

	// 初始化Service层
	svc := service.NewService(mongoDB, redisClient, kafkaProducer, messageClient)

	// 创建HTTP服务器
	httpServer := server.NewHTTPServerWrapper(cfg, kratosLogger)
	httpHandler := handler.NewHandler(svc, loggerInstance)
	httpHandler.RegisterRoutes(httpServer.GetEngine())

	// 创建gRPC服务器
	grpcService := svc.NewGRPCService(svc)
	nativeGrpcServer := grpc.NewServer()
	friendpb.RegisterFriendEventServiceServer(nativeGrpcServer, grpcService)

	// 启动gRPC服务器
	go func() {
		lis, err := net.Listen("tcp", cfg.Server.GRPC.Addr)
		if err != nil {
			log.Fatalf("Failed to listen gRPC: %v", err)
		}
		log.Printf("gRPC server starting on %s", cfg.Server.GRPC.Addr)
		if err := nativeGrpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// 启动HTTP服务器
	go func() {
		log.Printf("HTTP server starting on %s", cfg.Server.HTTP.Addr)
		if err := httpServer.Start(context.Background()); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	log.Println("Shutting down servers...")

	// 停止gRPC服务器
	nativeGrpcServer.GracefulStop()

	// 停止HTTP服务器
	if err := httpServer.Stop(context.Background()); err != nil {
		log.Printf("Failed to stop HTTP server: %v", err)
	}
}
