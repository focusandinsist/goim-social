package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/user-service/converter"
	"goim-social/apps/user-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedUserServiceServer
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    log,
	}
}

// Login 登陆
func (g *GRPCHandler) Login(ctx context.Context, req *rest.LoginRequest) (*rest.LoginResponse, error) {
	return g.loginImpl(ctx, req)
}

// Register 注册
func (g *GRPCHandler) Register(ctx context.Context, req *rest.RegisterRequest) (*rest.RegisterResponse, error) {
	return g.registerImpl(ctx, req)
}

// GetUser 获取用户
func (g *GRPCHandler) GetUser(ctx context.Context, req *rest.GetUserRequest) (*rest.GetUserResponse, error) {
	return g.getUserImpl(ctx, req)
}
