package server

import (
	"context"
	"net"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"

	"goim-social/pkg/config"
)

// GRPCServer gRPC服务器接口
type GRPCServer interface {
	GetServer() *grpc.Server
	RegisterService(registerFunc func(*grpc.Server))
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// GRPCServerWrapper gRPC服务器包装器
type GRPCServerWrapper struct {
	server   *grpc.Server
	addr     string
	logger   kratoslog.Logger
	listener net.Listener
}

// NewGRPCServerWrapper 创建gRPC服务器包装器
func NewGRPCServerWrapper(c *config.Config, logger kratoslog.Logger) *GRPCServerWrapper {
	// 添加拦截器
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
		// TODO: 拦截器
		),
		grpc.ChainStreamInterceptor(
		// TODO: 流拦截器
		),
	)

	return &GRPCServerWrapper{
		server: server,
		addr:   c.Server.GRPC.Addr,
		logger: logger,
	}
}

// GetServer 获取gRPC服务器
func (w *GRPCServerWrapper) GetServer() *grpc.Server {
	return w.server
}

// RegisterService 注册gRPC服务
func (w *GRPCServerWrapper) RegisterService(registerFunc func(*grpc.Server)) {
	registerFunc(w.server)
}

// Start 启动服务器
func (w *GRPCServerWrapper) Start(ctx context.Context) error {
	w.logger.Log(kratoslog.LevelInfo, "msg", "gRPC server starting", "addr", w.addr)

	lis, err := net.Listen("tcp", w.addr)
	if err != nil {
		return err
	}
	w.listener = lis

	return w.server.Serve(lis)
}

// Stop 停止服务器
func (w *GRPCServerWrapper) Stop(ctx context.Context) error {
	w.logger.Log(kratoslog.LevelInfo, "msg", "gRPC server stopping")
	w.server.GracefulStop()
	if w.listener != nil {
		w.listener.Close()
	}
	return nil
}
