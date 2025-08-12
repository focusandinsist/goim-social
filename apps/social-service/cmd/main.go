package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/social-service/converter"
	"goim-social/apps/social-service/dao"
	"goim-social/apps/social-service/handler"
	"goim-social/apps/social-service/model"
	"goim-social/apps/social-service/service"
	"goim-social/pkg/database"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/telemetry"
)

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// simpleLogger 简单的日志记录器实现
type simpleLogger struct{}

func (l *simpleLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
	log.Printf("[DEBUG] %s", msg)
}

func (l *simpleLogger) Info(ctx context.Context, msg string, fields ...logger.Field) {
	log.Printf("[INFO] %s", msg)
}

func (l *simpleLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
	log.Printf("[WARN] %s", msg)
}

func (l *simpleLogger) Error(ctx context.Context, msg string, fields ...logger.Field) {
	log.Printf("[ERROR] %s", msg)
}

func (l *simpleLogger) Fatal(ctx context.Context, msg string, fields ...logger.Field) {
	log.Fatalf("[FATAL] %s", msg)
}

func (l *simpleLogger) WithContext(ctx context.Context) logger.Logger {
	return l
}

func main() {
	serviceName := "social-service"

	// 初始化OpenTelemetry
	var otelConfig *telemetry.Config
	if os.Getenv("OTEL_DEBUG") == "true" {
		otelConfig = telemetry.DevelopmentConfig(serviceName)
		log.Printf("OpenTelemetry debug mode enabled - traces will be printed to console")
	} else {
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

	// 初始化数据库（使用环境变量或默认配置）
	dbHost := getEnv("DB_HOST", "localhost")
	dbName := getEnv("DB_NAME", "goim_social")

	db, err := database.NewPostgreSQL(dbHost, dbName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 自动迁移数据库表
	if err := db.GetDB().AutoMigrate(
		&model.Friend{},
		&model.FriendApply{},
		&model.Group{},
		&model.GroupMember{},
		&model.GroupInvitation{},
		&model.GroupJoinRequest{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 初始化Redis
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisClient := redis.NewRedisClient(redisAddr)
	defer redisClient.Close()

	// 初始化Kafka生产者（暂时使用nil，后续可以添加）
	var kafkaProducer *kafka.Producer = nil

	// 创建简单的日志记录器
	loggerInstance := &simpleLogger{}

	// 初始化DAO层
	socialDAO := dao.NewSocialDAO(db)

	// 初始化Service层
	socialService := service.NewService(socialDAO, redisClient, kafkaProducer, loggerInstance)

	// 初始化Converter层
	socialConverter := converter.NewConverter()

	// 初始化Handler层
	httpHandler := handler.NewHTTPHandler(socialService, socialConverter, loggerInstance)
	grpcHandler := handler.NewGRPCHandler(socialService, loggerInstance)

	// 设置Gin模式
	serverMode := getEnv("SERVER_MODE", "debug")
	if serverMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
	engine := gin.New()

	// 添加中间件
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	// 注册路由
	httpHandler.RegisterRoutes(engine)

	// 健康检查端点
	engine.POST("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "social-service",
			"time":    time.Now().Unix(),
		})
	})

	// 获取端口配置
	serverPort := getEnv("SERVER_PORT", "21002")

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: engine,
	}

	// 启动gRPC服务器
	grpcPort := getEnv("GRPC_PORT", "22002")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer()
	rest.RegisterSocialServiceServer(grpcServer, grpcHandler)

	go func() {
		log.Printf("Starting gRPC server on port %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// 启动HTTP服务器
	go func() {
		log.Printf("Starting HTTP server on port %s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down social service...")

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭gRPC服务器
	grpcServer.GracefulStop()

	// 关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Social service stopped")
}
