package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"goim-social/api/rest"
	"goim-social/apps/search-service/dao"
	"goim-social/apps/search-service/handler"
	"goim-social/apps/search-service/service"
	"goim-social/pkg/database"
	"goim-social/pkg/logger"
)

func main() {
	// 初始化配置
	cfg, err := initConfig()
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// 初始化日志
	log := initLogger(cfg)
	log.Info(context.Background(), "Starting search service")

	// 初始化数据库连接
	db, err := initDatabase(cfg, log)
	if err != nil {
		log.Fatal(context.Background(), "Failed to initialize database", logger.F("error", err.Error()))
	}

	// 初始化ElasticSearch客户端
	esClient, err := initElasticSearch(cfg, log)
	if err != nil {
		log.Fatal(context.Background(), "Failed to initialize ElasticSearch", logger.F("error", err.Error()))
	}

	// 初始化DAO层
	searchDAO := dao.NewElasticsearchDAO(esClient, log)
	historyDAO := dao.NewHistoryDAO(db, log)

	// 初始化服务配置
	serviceConfig := initServiceConfig(cfg)

	// 初始化服务层
	// 使用mock服务作为临时实现
	cacheService := service.NewMockCacheService()
	eventService := service.NewMockEventService()
	searchService := service.NewSearchService(searchDAO, historyDAO, cacheService, eventService, serviceConfig, log)
	indexService := service.NewIndexService(searchDAO, historyDAO, eventService, serviceConfig, log)

	// 初始化HTTP处理器
	httpHandler := handler.NewHTTPHandler(searchService, indexService, log)

	// 初始化gRPC处理器
	grpcHandler := handler.NewGRPCHandler(searchService, indexService, log)

	// 初始化Gin引擎
	gin.SetMode(getGinMode(cfg))
	router := gin.New()

	// 添加中间件
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestLoggerMiddleware(log))

	// 注册路由
	httpHandler.RegisterRoutes(router)

	// 启动HTTP服务器
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.GetInt("search.server.port")),
		Handler: router,
	}

	// 启动gRPC服务器
	grpcServer := grpc.NewServer()
	rest.RegisterSearchServiceServer(grpcServer, grpcHandler)
	rest.RegisterIndexServiceServer(grpcServer, grpcHandler)

	grpcPort := cfg.GetInt("search.server.port") + 1000 // HTTP端口+1000作为gRPC端口
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatal(context.Background(), "Failed to listen for gRPC", logger.F("error", err.Error()))
	}

	// 启动HTTP服务器
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(context.Background(), "Failed to start HTTP server", logger.F("error", err.Error()))
		}
	}()

	// 启动gRPC服务器
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatal(context.Background(), "Failed to start gRPC server", logger.F("error", err.Error()))
		}
	}()

	log.Info(context.Background(), "Search service started",
		logger.F("http_port", cfg.GetInt("search.server.port")),
		logger.F("grpc_port", grpcPort))

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info(context.Background(), "Shutting down search service...")

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error(context.Background(), "HTTP server forced to shutdown", logger.F("error", err.Error()))
	}

	// 关闭gRPC服务器
	grpcServer.GracefulStop()

	log.Info(context.Background(), "Search service stopped")
}

// initConfig 初始化配置
func initConfig() (*viper.Viper, error) {
	cfg := viper.New()

	// 设置配置文件路径 - 从根目录读取
	cfg.SetConfigName("config")
	cfg.SetConfigType("yaml")
	cfg.AddConfigPath(".")
	cfg.AddConfigPath("..")
	cfg.AddConfigPath("../..")
	cfg.AddConfigPath("../../..")

	// 设置环境变量
	cfg.AutomaticEnv()

	// 设置默认值
	cfg.SetDefault("search.server.port", 21011)
	cfg.SetDefault("search.server.mode", "debug")
	cfg.SetDefault("search.elasticsearch.addresses", []string{"http://localhost:9200"})
	cfg.SetDefault("database.postgresql.dsn", "host=localhost user=postgres password=123456 dbname=goim_search port=5432 sslmode=disable TimeZone=Asia/Shanghai")
	cfg.SetDefault("logger.level", "info")
	cfg.SetDefault("logger.format", "json")

	// 读取配置文件
	if err := cfg.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到，使用默认值
			log.Println("Config file not found, using default values")
		} else {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
	}

	return cfg, nil
}

// initLogger 初始化日志
func initLogger(cfg *viper.Viper) logger.Logger {
	logLevel := cfg.GetString("logger.level")
	if logLevel == "" {
		logLevel = "info"
	}

	log, err := logger.NewLogger(logLevel)
	if err != nil {
		// 如果创建失败，使用fallback logger
		return logger.GetLogger()
	}

	return log
}

// initDatabase 初始化数据库
func initDatabase(cfg *viper.Viper, log logger.Logger) (*database.PostgreSQL, error) {
	// 从配置中获取DSN
	dsn := cfg.GetString("database.postgresql.dsn")
	if dsn == "" {
		// 如果没有DSN，构建一个
		host := cfg.GetString("database.postgresql.host")
		port := cfg.GetInt("database.postgresql.port")
		user := cfg.GetString("database.postgresql.user")
		password := cfg.GetString("database.postgresql.password")
		dbname := cfg.GetString("database.postgresql.dbname")
		sslmode := cfg.GetString("database.postgresql.sslmode")

		if host == "" {
			host = "localhost"
		}
		if port == 0 {
			port = 5432
		}
		if user == "" {
			user = "postgres"
		}
		if password == "" {
			password = "123456"
		}
		if dbname == "" {
			dbname = "goim_search"
		}
		if sslmode == "" {
			sslmode = "disable"
		}

		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
			host, user, password, dbname, port, sslmode)
	}

	db, err := database.NewPostgreSQL(dsn, "goim_search")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	log.Info(context.Background(), "Database connected successfully")
	return db, nil
}

// initElasticSearch 初始化ElasticSearch客户端
func initElasticSearch(cfg *viper.Viper, log logger.Logger) (*elasticsearch.Client, error) {
	addresses := cfg.GetStringSlice("search.elasticsearch.addresses")
	if len(addresses) == 0 {
		addresses = []string{"http://localhost:9200"}
	}

	username := cfg.GetString("search.elasticsearch.username")
	password := cfg.GetString("search.elasticsearch.password")

	esConfig := elasticsearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
	}

	client, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ElasticSearch client: %v", err)
	}

	// 测试连接
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ElasticSearch: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ElasticSearch connection error: %s", res.String())
	}

	log.Info(context.Background(), "ElasticSearch connected successfully")
	return client, nil
}

// initServiceConfig 初始化服务配置
func initServiceConfig(cfg *viper.Viper) *service.ServiceConfig {
	// 构建CacheTTL映射
	cacheTTL := make(map[string]int)
	cacheTTL["search_results"] = 300
	cacheTTL["hot_queries"] = 3600
	cacheTTL["user_history"] = 86400
	cacheTTL["suggestions"] = 300
	cacheTTL["user_preference"] = 86400

	// 构建FieldWeights映射
	fieldWeights := make(map[string]float64)
	fieldWeights["title"] = 3.0
	fieldWeights["content"] = 1.0
	fieldWeights["summary"] = 2.0
	fieldWeights["tags"] = 2.0
	fieldWeights["username"] = 2.5
	fieldWeights["nickname"] = 2.0
	fieldWeights["bio"] = 1.0
	fieldWeights["group_name"] = 2.0
	fieldWeights["description"] = 1.5

	// 构建EventTopics映射
	eventTopics := make(map[string]string)
	eventTopics["search"] = "search-events"
	eventTopics["index"] = "index-events"

	return &service.ServiceConfig{
		DefaultPageSize:  cfg.GetInt("search.search_config.default_page_size"),
		MaxPageSize:      cfg.GetInt("search.search_config.max_page_size"),
		SearchTimeout:    30000, // 30秒，单位毫秒
		HighlightPreTag:  cfg.GetString("search.search_config.highlight_pre_tag"),
		HighlightPostTag: cfg.GetString("search.search_config.highlight_post_tag"),
		CacheEnabled:     cfg.GetBool("search.search_config.cache.enabled"),
		CacheTTL:         cacheTTL,
		IndexSettings:    cfg.GetStringMap("search.search_config.indices"),
		FieldWeights:     fieldWeights,
		EventEnabled:     cfg.GetBool("search.monitoring.enabled"),
		EventTopics:      eventTopics,
	}
}

// getGinMode 获取Gin模式
func getGinMode(cfg *viper.Viper) string {
	mode := cfg.GetString("search.server.mode")
	switch mode {
	case "release", "prod", "production":
		return gin.ReleaseMode
	case "test", "testing":
		return gin.TestMode
	default:
		return gin.DebugMode
	}
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-ID, X-Admin")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// requestLoggerMiddleware 请求日志中间件
func requestLoggerMiddleware(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		log.Info(c.Request.Context(), "HTTP request completed",
			logger.F("method", c.Request.Method),
			logger.F("path", c.Request.URL.Path),
			logger.F("status_code", statusCode),
			logger.F("duration_ms", duration.Milliseconds()),
			logger.F("client_ip", c.ClientIP()))
	}
}
