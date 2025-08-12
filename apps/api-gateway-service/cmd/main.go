package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/api-gateway-service/internal/handler"
	"goim-social/apps/api-gateway-service/internal/service"
	"goim-social/pkg/middleware"
	"goim-social/pkg/server"
	"goim-social/pkg/telemetry"
)

func main() {
	serviceName := "api-gateway-service"

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
	svc := service.NewService(app.GetMongoDB(), app.GetRedisClient(), app.GetKafkaProducer(), app.GetConfig(), app.GetLogger())

	// 创建OpenTelemetry中间件
	otelMW := middleware.NewOTelMiddleware(serviceName, app.GetLogger())

	// 创建各handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		// 创建API网关中间件
		apiGatewayMW := middleware.NewAPIGatewayMiddleware(app.GetLogger(), "focusandinsist")

		// 注册中间件（OpenTelemetry中间件放在最前面）
		engine.Use(otelMW.GinMiddleware())     // OpenTelemetry追踪
		engine.Use(apiGatewayMW.GinLogging())  // 请求日志
		engine.Use(apiGatewayMW.GinRecovery()) // 恢复中间件
		engine.Use(apiGatewayMW.GinAuth())     // 认证中间件

		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterConnectServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
