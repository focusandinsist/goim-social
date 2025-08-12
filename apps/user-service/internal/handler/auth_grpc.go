package handler

import (
	"context"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// loginImpl 登陆实现
func (g *GRPCHandler) loginImpl(ctx context.Context, req *rest.LoginRequest) (*rest.LoginResponse, error) {
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

// registerImpl 注册实现
func (g *GRPCHandler) registerImpl(ctx context.Context, req *rest.RegisterRequest) (*rest.RegisterResponse, error) {
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
