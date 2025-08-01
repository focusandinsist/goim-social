package client

import (
	"fmt"
	"sync"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"goim-social/pkg/config"
)

// ClientManager 客户端管理器
type ClientManager struct {
	config      *config.Config
	logger      kratoslog.Logger
	grpcClients map[string]*grpc.ClientConn
	mu          sync.RWMutex
}

// NewClientManager 创建客户端管理器
func NewClientManager(cfg *config.Config, logger kratoslog.Logger) *ClientManager {
	return &ClientManager{
		config:      cfg,
		logger:      logger,
		grpcClients: make(map[string]*grpc.ClientConn),
	}
}

// GetGRPCClient 获取gRPC客户端连接
func (cm *ClientManager) GetGRPCClient(serviceName string, addr string) (*grpc.ClientConn, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查是否已存在连接
	if conn, exists := cm.grpcClients[serviceName]; exists {
		if conn.GetState().String() != "SHUTDOWN" {
			return conn, nil
		}
		// 连接已关闭，删除并重新创建
		delete(cm.grpcClients, serviceName)
	}

	// 创建新连接
	conn, err := cm.createGRPCConnection(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to %s: %w", serviceName, err)
	}

	cm.grpcClients[serviceName] = conn
	cm.logger.Log(kratoslog.LevelInfo, "msg", "gRPC client connected", "service", serviceName, "addr", addr)

	return conn, nil
}

// GetGRPCClientWithRetry 获取gRPC客户端连接
func (cm *ClientManager) GetGRPCClientWithRetry(serviceName string, addr string, maxRetries int) (*grpc.ClientConn, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ { // 重试
		conn, err := cm.GetGRPCClient(serviceName, addr)
		if err == nil {
			return conn, nil
		}

		lastErr = err
		cm.logger.Log(kratoslog.LevelWarn, "msg", "gRPC connection failed, retrying",
			"service", serviceName, "attempt", i+1, "error", err)

		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	return nil, fmt.Errorf("failed to connect to %s after %d attempts: %w", serviceName, maxRetries, lastErr)
}

// createGRPCConnection 创建gRPC连接
func (cm *ClientManager) createGRPCConnection(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// CloseAll 关闭所有客户端连接
func (cm *ClientManager) CloseAll() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var errors []error
	for serviceName, conn := range cm.grpcClients {
		if err := conn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close %s: %w", serviceName, err))
		} else {
			cm.logger.Log(kratoslog.LevelInfo, "msg", "gRPC client closed", "service", serviceName)
		}
	}

	// 清空连接映射
	cm.grpcClients = make(map[string]*grpc.ClientConn)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing clients: %v", errors)
	}
	return nil
}

// GetConnectionStats 获取连接统计信息
func (cm *ClientManager) GetConnectionStats() map[string]any {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := make(map[string]any)
	stats["total_connections"] = len(cm.grpcClients)

	connections := make(map[string]string)
	for serviceName, conn := range cm.grpcClients {
		connections[serviceName] = conn.GetState().String()
	}
	stats["connections"] = connections

	return stats
}

// HealthCheck 健康检查所有连接
func (cm *ClientManager) HealthCheck() map[string]bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	health := make(map[string]bool)
	for serviceName, conn := range cm.grpcClients {
		state := conn.GetState()
		health[serviceName] = state.String() == "READY" || state.String() == "IDLE"
	}

	return health
}
