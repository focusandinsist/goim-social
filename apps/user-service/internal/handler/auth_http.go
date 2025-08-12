package handler

import (
	"github.com/gin-gonic/gin"

	rest "goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// Register 用户注册
func (h *HTTPHandler) Register(c *gin.Context) {

	var (
		ctx  = c.Request.Context()
		req  rest.RegisterRequest
		resp *rest.RegisterResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid register request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorRegisterResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	h.logger.Info(ctx, "User registration attempt", logger.F("username", req.Username))

	resp, err = h.service.Register(ctx, &req)
	if err != nil {
		h.logger.Error(ctx, "User registration failed",
			logger.F("username", req.Username),
			logger.F("error", err.Error()))
		resp = h.converter.BuildErrorRegisterResponse(err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将用户ID添加到context
	ctx = tracecontext.WithUserID(ctx, resp.User.Id)

	h.logger.Info(ctx, "User registration successful",
		logger.F("user_id", resp.User.Id),
		logger.F("username", resp.User.Username))
	httpx.WriteObject(c, resp, nil)
}

// Login 用户登录
func (h *HTTPHandler) Login(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.LoginRequest
		resp *rest.LoginResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid login request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorLoginResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	h.logger.Info(ctx, "User login attempt", logger.F("username", req.Username))

	resp, err = h.service.Login(ctx, &req)
	if err != nil {
		h.logger.Error(ctx, "User login failed", logger.F("username", req.Username), logger.F("error", err.Error()))
		resp = h.converter.BuildErrorLoginResponse(err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将用户ID添加到context
	ctx = tracecontext.WithUserID(ctx, resp.User.Id)

	h.logger.Info(ctx, "User login successful",
		logger.F("user_id", resp.User.Id),
		logger.F("username", resp.User.Username))
	httpx.WriteObject(c, resp, nil)
}
