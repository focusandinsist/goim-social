package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	App      AppConfig      `yaml:"app"`
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Connect  ConnectConfig  `yaml:"connect"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name      string `yaml:"name"`
	Version   string `yaml:"version"`
	JWTSecret string `yaml:"jwt_secret"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTP HTTPConfig `yaml:"http"`
	GRPC GRPCConfig `yaml:"grpc"`
}

// HTTPConfig HTTP服务配置
type HTTPConfig struct {
	Network string `yaml:"network"`
	Addr    string `yaml:"addr"`
	Timeout string `yaml:"timeout"`
}

// GRPCConfig gRPC服务配置
type GRPCConfig struct {
	Network string `yaml:"network"`
	Addr    string `yaml:"addr"`
	Timeout string `yaml:"timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MongoDB    MongoDBConfig    `yaml:"mongodb"`
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
}

// MongoDBConfig MongoDB配置
type MongoDBConfig struct {
	URI    string `yaml:"uri"`
	DBName string `yaml:"db_name"`
}

// PostgreSQLConfig PostgreSQL配置
type PostgreSQLConfig struct {
	DSN    string `yaml:"dsn"`
	DBName string `yaml:"db_name"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	GroupID string   `yaml:"group_id"`
}

// ConnectConfig Connect服务配置
type ConnectConfig struct {
	MessageService MessageServiceConfig `yaml:"message_service"`
	Instance       InstanceConfig       `yaml:"instance"`
	Heartbeat      HeartbeatConfig      `yaml:"heartbeat"`
	Connection     ConnectionConfig     `yaml:"connection"`
}

// MessageServiceConfig Message服务连接配置
type MessageServiceConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// InstanceConfig 实例配置
type InstanceConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	Interval int `yaml:"interval"` // 心跳间隔（秒）
	Timeout  int `yaml:"timeout"`  // 超时时间（秒）
}

// ConnectionConfig 连接配置
type ConnectionConfig struct {
	ExpireTime int    `yaml:"expire_time"` // 连接过期时间（小时）
	ClientType string `yaml:"client_type"` // 默认客户端类型
}

// LoadConfig 从环境变量加载配置
func LoadConfig(serviceName string) *Config {

	var defaultHTTPPort, defaultGRPCPort string

	// 根据服务名称设置默认端口
	switch serviceName {
	case "user-service":
		defaultHTTPPort = "21001"
		defaultGRPCPort = "22001"
	case "group-service":
		defaultHTTPPort = "21002"
		defaultGRPCPort = "22002"
	case "friend-service":
		defaultHTTPPort = "21003"
		defaultGRPCPort = "22003"
	case "message-service":
		defaultHTTPPort = "21004"
		defaultGRPCPort = "22004"
	case "connect-service":
		defaultHTTPPort = "21005"
		defaultGRPCPort = "22005"
	default:
		panic(fmt.Sprintf("未知的服务名称: %s，支持的服务名称: user-service, group-service, friend-service, message-service, connect-service", serviceName))
	}

	httpPort := getEnvOrDefault("HTTP_PORT", defaultHTTPPort)
	grpcPort := getEnvOrDefault("GRPC_PORT", defaultGRPCPort)

	return &Config{
		App: AppConfig{
			Name:      serviceName,
			Version:   getEnvOrDefault("APP_VERSION", "1.0.0"),
			JWTSecret: getEnvOrDefault("JWT_SECRET", "focusandinsist"),
		},
		Server: ServerConfig{
			HTTP: HTTPConfig{
				Network: "tcp",
				Addr:    ":" + httpPort,
				Timeout: "30s",
			},
			GRPC: GRPCConfig{
				Network: "tcp",
				Addr:    ":" + grpcPort,
				Timeout: "30s",
			},
		},
		Database: DatabaseConfig{
			MongoDB: MongoDBConfig{
				URI:    getEnvOrDefault("MONGODB_URI", "mongodb://localhost:27017"),
				DBName: getEnvOrDefault("MONGODB_DB", serviceName+"DB"),
			},
			PostgreSQL: PostgreSQLConfig{
				DSN:    getEnvOrDefault("POSTGRESQL_DSN", "host=localhost user=postgres password=postgres dbname="+serviceName+"DB port=5432 sslmode=disable TimeZone=Asia/Shanghai"),
				DBName: getEnvOrDefault("POSTGRESQL_DB", serviceName+"DB"),
			},
		},
		Redis: RedisConfig{
			Addr:     getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       getEnvIntOrDefault("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnvOrDefault("KAFKA_BROKERS", "localhost:9092")},
			GroupID: getEnvOrDefault("KAFKA_GROUP_ID", serviceName+"-group"),
		},
		Connect: ConnectConfig{
			MessageService: MessageServiceConfig{
				Host: getEnvOrDefault("MESSAGE_SERVICE_HOST", "localhost"),
				Port: getEnvIntOrDefault("MESSAGE_SERVICE_PORT", 22004),
			},
			Instance: InstanceConfig{
				Host: getEnvOrDefault("INSTANCE_HOST", "localhost"),
				Port: getEnvIntOrDefault("INSTANCE_PORT", 21003),
			},
			Heartbeat: HeartbeatConfig{
				Interval: getEnvIntOrDefault("HEARTBEAT_INTERVAL", 10),
				Timeout:  getEnvIntOrDefault("HEARTBEAT_TIMEOUT", 30),
			},
			Connection: ConnectionConfig{
				ExpireTime: getEnvIntOrDefault("CONNECTION_EXPIRE_TIME", 2),
				ClientType: getEnvOrDefault("DEFAULT_CLIENT_TYPE", "web"),
			},
		},
	}
}

// getEnvOrDefault 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault 获取环境变量整数值或默认值
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
