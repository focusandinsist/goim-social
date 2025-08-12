package handler

import (
	"github.com/gin-gonic/gin"

	rest "goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// GetUserByID 根据ID获取用户
func (h *HTTPHandler) GetUserByID(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetUserRequest
		resp *rest.GetUserResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetUserResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将用户ID添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	h.logger.Info(ctx, "Get user by ID", logger.F("user_id", req.UserId))

	user, err := h.service.GetUserByID(ctx, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "Get user failed",
			logger.F("user_id", req.UserId),
			logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetUserResponse(err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	h.logger.Info(ctx, "Get user by ID successful",
		logger.F("user_id", req.UserId),
		logger.F("username", user.Username))

	resp = h.converter.BuildGetUserResponse(true, "获取用户成功", user)
	httpx.WriteObject(c, resp, nil)
}
