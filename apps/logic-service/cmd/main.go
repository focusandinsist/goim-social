package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/logic-service/handler"
	"goim-social/apps/logic-service/service"
	"goim-social/pkg/middleware"
	"goim-social/pkg/server"
	"goim-social/pkg/snowflake"
	"goim-social/pkg/telemetry"
)

func main() {
	serviceName := "logic-service"

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

	// 初始化Snowflake ID生成器 (Logic服务使用机器ID: 1)
	if err := snowflake.InitGlobalSnowflake(1); err != nil {
		panic(fmt.Sprintf("初始化Snowflake失败: %v", err))
	}

	// 创建应用程序
	app := server.NewApplication(serviceName)

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
		config.Kafka.Brokers, // 传递Kafka brokers配置
		groupAddr,
		messageAddr,
		friendAddr,
		userAddr,
	)
	if err != nil {
		panic("Failed to create logic service: " + err.Error())
	}

	// 创建OpenTelemetry中间件
	otelMW := middleware.NewOTelMiddleware(serviceName, app.GetLogger())

	// 初始化Handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		// 添加OpenTelemetry中间件
		engine.Use(otelMW.GinMiddleware())

		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterLogicServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
