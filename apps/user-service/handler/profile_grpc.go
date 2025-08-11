package handler

import (
	"context"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// getUserImpl 获取用户实现
func (g *GRPCHandler) getUserImpl(ctx context.Context, req *rest.GetUserRequest) (*rest.GetUserResponse, error) {
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
