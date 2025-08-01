package main

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/content-service/dao"
	"goim-social/apps/content-service/handler"
	"goim-social/apps/content-service/model"
	"goim-social/apps/content-service/service"
	"goim-social/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("content-service")

	// 启用HTTP和gRPC服务器
	app.EnableHTTP()
	app.EnableGRPC()

	// 初始化PostgreSQL连接
	postgreSQL := app.GetPostgreSQL()

	// 自动迁移数据库表结构
	if err := postgreSQL.AutoMigrate(
		&model.Content{},
		&model.ContentMediaFile{},
		&model.ContentTag{},
		&model.ContentTopic{},
		&model.ContentTagRelation{},
		&model.ContentTopicRelation{},
		&model.ContentStatusLog{},
	); err != nil {
		panic("Failed to migrate database: " + err.Error())
	}

	// 初始化DAO层
	contentDAO := dao.NewContentDAO(postgreSQL)

	// 初始化Service层
	svc := service.NewService(contentDAO, app.GetRedisClient(), app.GetKafkaProducer(), app.GetLogger())

	// 初始化Handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())
	grpcHandler := handler.NewGRPCHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler.RegisterRoutes(engine)
	})

	// 注册gRPC服务
	app.RegisterGRPCService(func(grpcSrv *grpc.Server) {
		rest.RegisterContentServiceServer(grpcSrv, grpcHandler)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
