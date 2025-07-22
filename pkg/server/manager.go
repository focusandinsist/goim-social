package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"

	"websocket-server/pkg/config"
)

// ServerManager 统一服务器管理器
type ServerManager struct {
	config     *config.Config
	logger     kratoslog.Logger
	httpServer HTTPServer
	grpcServer GRPCServer
	servers    []Server
	mu         sync.RWMutex
}

// Server 通用服务器接口
type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// NewServerManager 创建服务器管理器
func NewServerManager(cfg *config.Config, logger kratoslog.Logger) *ServerManager {
	return &ServerManager{
		config:  cfg,
		logger:  logger,
		servers: make([]Server, 0),
	}
}

// EnableHTTP 启用HTTP服务器
func (sm *ServerManager) EnableHTTP() HTTPServer {
	if sm.httpServer == nil {
		sm.httpServer = NewHTTPServerWrapper(sm.config, sm.logger)
		sm.addServer(sm.httpServer)
	}
	return sm.httpServer
}

// EnableGRPC 启用gRPC服务器
func (sm *ServerManager) EnableGRPC() GRPCServer {
	if sm.grpcServer == nil {
		sm.grpcServer = NewGRPCServerWrapper(sm.config, sm.logger)
		sm.addServer(sm.grpcServer)
	}
	return sm.grpcServer
}

// GetHTTPServer 获取HTTP服务器
func (sm *ServerManager) GetHTTPServer() HTTPServer {
	return sm.httpServer
}

// GetGRPCServer 获取gRPC服务器
func (sm *ServerManager) GetGRPCServer() GRPCServer {
	return sm.grpcServer
}

// RegisterHTTPRoutes 注册HTTP路由
func (sm *ServerManager) RegisterHTTPRoutes(registerFunc func(*gin.Engine)) error {
	if sm.httpServer == nil {
		return fmt.Errorf("HTTP server not enabled")
	}
	sm.httpServer.RegisterRoutes(registerFunc)
	return nil
}

// RegisterGRPCService 注册gRPC服务
func (sm *ServerManager) RegisterGRPCService(registerFunc func(*grpc.Server)) error {
	if sm.grpcServer == nil {
		return fmt.Errorf("gRPC server not enabled")
	}
	sm.grpcServer.RegisterService(registerFunc)
	return nil
}

// addServer 添加服务器到管理列表
func (sm *ServerManager) addServer(server Server) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.servers = append(sm.servers, server)
}

// StartAll 启动所有服务器
func (sm *ServerManager) StartAll(ctx context.Context) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 启动所有服务器
	for _, server := range sm.servers {
		go func(s Server) {
			if err := s.Start(ctx); err != nil {
				sm.logger.Log(kratoslog.LevelError, "msg", "Server start failed", "error", err)
			}
		}(server)
	}

	sm.logger.Log(kratoslog.LevelInfo, "msg", "All servers started")
	return nil
}

// StopAll 停止所有服务器
func (sm *ServerManager) StopAll(ctx context.Context) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var errors []error
	for _, server := range sm.servers {
		if err := server.Stop(ctx); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping servers: %v", errors)
	}
	return nil
}
