package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/user-service/converter"
	"goim-social/apps/user-service/service"
	tracecontext "goim-social/pkg/context"
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
	g.logger.Info(ctx, "gRPC Login attempt", logger.F("username", req.Username))

	response, err := g.svc.Login(ctx, req)
	if err != nil {
		g.logger.Error(ctx, "gRPC Login failed",
			logger.F("username", req.Username),
			logger.F("error", err.Error()))
		return nil, err
	}

	// 将用户ID添加到context
	ctx = tracecontext.WithUserID(ctx, response.User.Id)

	g.logger.Info(ctx, "gRPC Login successful",
		logger.F("userID", response.User.Id),
		logger.F("username", req.Username))

	return response, nil
}

// Register 注册
func (g *GRPCHandler) Register(ctx context.Context, req *rest.RegisterRequest) (*rest.RegisterResponse, error) {
	g.logger.Info(ctx, "gRPC Register attempt", logger.F("username", req.Username))

	response, err := g.svc.Register(ctx, req)
	if err != nil {
		g.logger.Error(ctx, "gRPC Register failed",
			logger.F("username", req.Username),
			logger.F("error", err.Error()))
		return nil, err
	}

	// 将用户ID添加到context
	ctx = tracecontext.WithUserID(ctx, response.User.Id)

	g.logger.Info(ctx, "gRPC Register successful",
		logger.F("userID", response.User.Id),
		logger.F("username", req.Username))

	return response, nil
}

// GetUser 获取用户
func (g *GRPCHandler) GetUser(ctx context.Context, req *rest.GetUserRequest) (*rest.GetUserResponse, error) {
	// 将用户ID添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	g.logger.Info(ctx, "gRPC GetUser request", logger.F("userID", req.UserId))

	user, err := g.svc.GetUserByID(ctx, req.UserId)
	if err != nil {
		g.logger.Error(ctx, "gRPC GetUser failed",
			logger.F("userID", req.UserId),
			logger.F("error", err.Error()))
		return g.converter.BuildErrorGetUserResponse(err.Error()), nil
	}

	g.logger.Info(ctx, "gRPC GetUser successful",
		logger.F("userID", req.UserId),
		logger.F("username", user.Username))

	return g.converter.BuildGetUserResponse(true, "获取用户信息成功", user), nil
}
