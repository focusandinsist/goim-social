package main

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"websocket-server/api/rest"
	"websocket-server/apps/interaction-service/dao"
	"websocket-server/apps/interaction-service/handler"
	"websocket-server/apps/interaction-service/model"
	"websocket-server/apps/interaction-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("interaction-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 初始化PostgreSQL连接
	postgreSQL := app.GetPostgreSQL()

	// 自动迁移数据库表结构
	if err := postgreSQL.AutoMigrate(
		&model.Interaction{},
		&model.InteractionStats{},
	); err != nil {
		panic("Failed to migrate database: " + err.Error())
	}

	// 初始化DAO层
	interactionDAO := dao.NewInteractionDAO(postgreSQL)

	// 初始化Service层
	svc := service.NewService(interactionDAO, app.GetRedisClient(), app.GetKafkaProducer(), app.GetLogger())

	// 初始化Handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterInteractionServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
