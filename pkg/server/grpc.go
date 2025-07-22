package server

import (
	"context"
	"log"
	"net"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"

	"websocket-server/pkg/config"
)

// GRPCServerWrapper gRPC服务器包装器
type GRPCServerWrapper struct {
	server *grpc.Server
	addr   string
}

// NewGRPCServerWrapper 创建gRPC服务器包装器
func NewGRPCServerWrapper(c *config.Config, logger kratoslog.Logger) *GRPCServerWrapper {
	server := grpc.NewServer()

	return &GRPCServerWrapper{
		server: server,
		addr:   c.Server.GRPC.Addr,
	}
}

// GetServer 获取gRPC服务器
func (w *GRPCServerWrapper) GetServer() *grpc.Server {
	return w.server
}

// Start 启动服务器
func (w *GRPCServerWrapper) Start(ctx context.Context) error {
	log.Printf("gRPC server starting on %s", w.addr)
	lis, err := net.Listen("tcp", w.addr)
	if err != nil {
		return err
	}
	return w.server.Serve(lis)
}

// Stop 停止服务器
func (w *GRPCServerWrapper) Stop(ctx context.Context) error {
	log.Println("gRPC server stopping")
	w.server.GracefulStop()
	return nil
}
