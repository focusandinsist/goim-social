package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/social-service/internal/converter"
	"goim-social/apps/social-service/internal/dao"
	"goim-social/apps/social-service/internal/handler"
	"goim-social/apps/social-service/internal/model"
	"goim-social/apps/social-service/internal/service"
	"goim-social/pkg/middleware"
	"goim-social/pkg/server"
	"goim-social/pkg/telemetry"
)

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

	// 创建应用程序
	app := server.NewApplication(serviceName)

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 获取PostgreSQL连接
	postgreSQL := app.GetPostgreSQL()

	// 自动迁移数据库表结构
	if err := postgreSQL.AutoMigrate(
		&model.Friend{},
		&model.FriendApply{},
		&model.Group{},
		&model.GroupMember{},
		&model.GroupInvitation{},
		&model.GroupJoinRequest{},
	); err != nil {
		panic("Failed to migrate database: " + err.Error())
	}

	// 初始化DAO层
	socialDAO := dao.NewSocialDAO(postgreSQL)

	// 初始化Service层
	socialService := service.NewService(socialDAO, app.GetRedisClient(), app.GetKafkaProducer(), app.GetLogger())

	// 初始化Converter层
	socialConverter := converter.NewConverter()

	// 创建OpenTelemetry中间件
	otelMW := middleware.NewOTelMiddleware(serviceName, app.GetLogger())

	// 初始化Handler层
	httpHandler := handler.NewHTTPHandler(socialService, socialConverter, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(socialService, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		// 添加OpenTelemetry中间件
		engine.Use(otelMW.GinMiddleware())

		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterSocialServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
