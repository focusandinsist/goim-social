package server

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"

	"goim-social/pkg/client"
	"goim-social/pkg/config"
	"goim-social/pkg/database"
	"goim-social/pkg/kafka"
	"goim-social/pkg/lifecycle"
	"goim-social/pkg/logger"
	"goim-social/pkg/middleware"
	"goim-social/pkg/redis"
)

// Application 应用程序框架
type Application struct {
	serviceName    string
	config         *config.Config
	logger         kratoslog.Logger
	originalLogger logger.Logger
	serverManager  *ServerManager
	clientManager  *client.ClientManager
	lifecycle      *lifecycle.LifecycleManager

	// 基础设施组件
	mongoDB       *database.MongoDB
	postgreSQL    *database.PostgreSQL
	redisClient   *redis.RedisClient
	kafkaProducer *kafka.Producer

	// 中间件
	authMiddleware    *middleware.AuthMiddleware
	loggingMiddleware *middleware.LoggingMiddleware

	// 注册函数
	httpRouteRegister   func(*gin.Engine)
	grpcServiceRegister func(*grpc.Server)
}

// NewApplication 创建应用程序
func NewApplication(serviceName string) *Application {
	// 加载配置
	cfg := config.LoadConfig(serviceName)

	// 初始化原有日志系统
	if err := logger.Init("info"); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	originalLogger := logger.GetLogger()

	// 创建Kratos日志器
	kratosLogger := logger.NewKratosStdLogger(cfg.App.Name, cfg.App.Version)

	// 创建生命周期管理器
	lifecycleManager := lifecycle.NewLifecycleManager(kratosLogger)

	// 创建服务器管理器
	serverManager := NewServerManager(cfg, kratosLogger)

	// 创建客户端管理器
	clientManager := client.NewClientManager(cfg, kratosLogger)

	// 创建中间件
	authMiddleware := middleware.NewAuthMiddleware(kratosLogger, cfg.App.JWTSecret)
	loggingMiddleware := middleware.NewLoggingMiddleware(kratosLogger)

	app := &Application{
		serviceName:       serviceName,
		config:            cfg,
		logger:            kratosLogger,
		originalLogger:    originalLogger,
		serverManager:     serverManager,
		clientManager:     clientManager,
		lifecycle:         lifecycleManager,
		authMiddleware:    authMiddleware,
		loggingMiddleware: loggingMiddleware,
	}

	// 初始化基础设施
	app.initInfrastructure()

	return app
}

// initInfrastructure 初始化基础设施组件
func (app *Application) initInfrastructure() {
	// 初始化MongoDB
	mongoDB, err := database.NewMongoDB(app.config.Database.MongoDB.URI, app.config.Database.MongoDB.DBName)
	if err != nil {
		app.logger.Log(kratoslog.LevelFatal, "msg", "Failed to connect to MongoDB", "error", err)
		panic(err)
	}
	app.mongoDB = mongoDB

	// 初始化PostgreSQL
	postgreSQL, err := database.NewPostgreSQL(app.config.Database.PostgreSQL.DSN, app.config.Database.PostgreSQL.DBName)
	if err != nil {
		app.logger.Log(kratoslog.LevelFatal, "msg", "Failed to connect to PostgreSQL", "error", err)
		panic(err)
	}
	app.postgreSQL = postgreSQL

	// 初始化Redis
	app.redisClient = redis.NewRedisClient(app.config.Redis.Addr)

	// 初始化Kafka
	kafkaProducer, err := kafka.InitProducer(app.config.Kafka.Brokers)
	if err != nil {
		app.logger.Log(kratoslog.LevelFatal, "msg", "Failed to connect to Kafka", "error", err)
		panic(err)
	}
	app.kafkaProducer = kafkaProducer
}

// EnableHTTP 启用HTTP服务器
func (app *Application) EnableHTTP() HTTPServer {
	httpServer := app.serverManager.EnableHTTP()

	// 添加中间件
	httpServer.RegisterRoutes(func(engine *gin.Engine) {
		engine.Use(app.loggingMiddleware.GinLogging())
		engine.Use(app.loggingMiddleware.GinRecovery())
		engine.Use(app.authMiddleware.GinAuth())
	})

	return httpServer
}

// EnableGRPC 启用gRPC服务器
func (app *Application) EnableGRPC() GRPCServer {
	grpcServer := app.serverManager.EnableGRPC()
	return grpcServer
}

// RegisterHTTPRoutes 注册HTTP路由
func (app *Application) RegisterHTTPRoutes(registerFunc func(*gin.Engine)) {
	app.httpRouteRegister = registerFunc
}

// RegisterGRPCService 注册gRPC服务
func (app *Application) RegisterGRPCService(registerFunc func(*grpc.Server)) {
	app.grpcServiceRegister = registerFunc
}

// GetMongoDB 获取MongoDB连接
func (app *Application) GetMongoDB() *database.MongoDB {
	return app.mongoDB
}

// GetRedisClient 获取Redis客户端
func (app *Application) GetRedisClient() *redis.RedisClient {
	return app.redisClient
}

// GetKafkaProducer 获取Kafka生产者
func (app *Application) GetKafkaProducer() *kafka.Producer {
	return app.kafkaProducer
}

// GetPostgreSQL 获取PostgreSQL连接
func (app *Application) GetPostgreSQL() *database.PostgreSQL {
	return app.postgreSQL
}

// GetLogger 获取原有日志器
func (app *Application) GetLogger() logger.Logger {
	return app.originalLogger
}

// GetKratosLogger 获取Kratos日志器
func (app *Application) GetKratosLogger() kratoslog.Logger {
	return app.logger
}

// GetConfig 获取配置
func (app *Application) GetConfig() *config.Config {
	return app.config
}

// GetClientManager 获取客户端管理器
func (app *Application) GetClientManager() *client.ClientManager {
	return app.clientManager
}

// Run 运行应用程序
func (app *Application) Run() error {
	// 注册生命周期钩子
	app.registerLifecycleHooks()

	// 启动生命周期管理器
	if err := app.lifecycle.Start(); err != nil {
		return fmt.Errorf("failed to start lifecycle: %w", err)
	}

	// 等待停止信号
	app.lifecycle.Wait()

	return nil
}

// registerLifecycleHooks 注册生命周期钩子
func (app *Application) registerLifecycleHooks() {
	// 注册HTTP路由
	if app.httpRouteRegister != nil {
		app.serverManager.RegisterHTTPRoutes(app.httpRouteRegister)
	}

	// 注册gRPC服务
	if app.grpcServiceRegister != nil {
		app.serverManager.RegisterGRPCService(app.grpcServiceRegister)
	}

	// 服务器启动钩子
	app.lifecycle.AddHook(lifecycle.Hook{
		Name:     "servers",
		Priority: 100,
		OnStart: func(ctx context.Context) error {
			return app.serverManager.StartAll(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return app.serverManager.StopAll(ctx)
		},
	})

	// 客户端清理钩子
	app.lifecycle.AddHook(lifecycle.Hook{
		Name:     "clients",
		Priority: 200,
		OnStop: func(ctx context.Context) error {
			return app.clientManager.CloseAll()
		},
	})

	// 数据库清理钩子
	app.lifecycle.AddHook(lifecycle.Hook{
		Name:     "databases",
		Priority: 300,
		OnStop: func(ctx context.Context) error {
			if app.mongoDB != nil {
				if err := app.mongoDB.Close(); err != nil {
					app.logger.Log(kratoslog.LevelError, "msg", "Failed to close MongoDB", "error", err)
				}
			}
			if app.postgreSQL != nil {
				if err := app.postgreSQL.Close(); err != nil {
					app.logger.Log(kratoslog.LevelError, "msg", "Failed to close PostgreSQL", "error", err)
				}
			}
			return nil
		},
	})
}
