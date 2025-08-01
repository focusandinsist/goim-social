package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/user-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedUserServiceServer
	svc    *service.Service
	logger logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:    svc,
		logger: log,
	}
}

// Login 登陆
func (g *GRPCHandler) Login(ctx context.Context, req *rest.LoginRequest) (*rest.LoginResponse, error) {
	return g.svc.Login(ctx, req)
}

// Register 注册
func (g *GRPCHandler) Register(ctx context.Context, req *rest.RegisterRequest) (*rest.RegisterResponse, error) {
	return g.svc.Register(ctx, req)
}

// GetUser 获取用户
func (g *GRPCHandler) GetUser(ctx context.Context, req *rest.GetUserRequest) (*rest.GetUserResponse, error) {
	// 这里需要将string类型的user_id转换为int64
	// 简化处理，实际应该做更严格的转换
	userID := int64(1) // 临时处理

	user, err := g.svc.GetUserByID(ctx, userID)
	if err != nil {
		return &rest.GetUserResponse{
			Success: false,
			Message: err.Error(),
			User:    nil,
		}, nil
	}

	return &rest.GetUserResponse{
		Success: true,
		Message: "获取用户信息成功",
		User:    user,
	}, nil
}
