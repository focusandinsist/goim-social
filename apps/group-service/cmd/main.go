package main

import (
	"github.com/gin-gonic/gin"

	"websocket-server/apps/group-service/dao"
	"websocket-server/apps/group-service/handler"
	"websocket-server/apps/group-service/model"
	"websocket-server/apps/group-service/service"
	"websocket-server/pkg/server"
)

func main() {
	// 创建应用程序
	app := server.NewApplication("group-service")

	// 启用HTTP服务器
	app.EnableHTTP()

	// 初始化PostgreSQL连接
	postgreSQL := app.GetPostgreSQL()

	// 自动迁移数据库表结构
	if err := postgreSQL.AutoMigrate(
		&model.Group{},
		&model.GroupMember{},
		&model.GroupInvitation{},
		&model.GroupJoinRequest{},
		&model.GroupAnnouncement{},
	); err != nil {
		panic("Failed to migrate database: " + err.Error())
	}

	// 初始化DAO层
	groupDAO := dao.NewGroupDAO(postgreSQL)

	// 初始化Service层
	svc := service.NewService(groupDAO, app.GetRedisClient(), app.GetKafkaProducer(), app.GetLogger())

	// 初始化Handler
	httpHandler := handler.NewHTTPHandler(svc, app.GetLogger())

	// 注册HTTP路由
	app.RegisterHTTPRoutes(func(engine *gin.Engine) {
		httpHandler.RegisterRoutes(engine)
	})

	// 运行应用程序
	if err := app.Run(); err != nil {
		panic(err)
	}
}
