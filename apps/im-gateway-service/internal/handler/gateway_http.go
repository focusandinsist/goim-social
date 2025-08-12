package handler

import (
	"github.com/gin-gonic/gin"

	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// OnlineStatus 查询在线状态
func (h *HTTPHandler) OnlineStatus(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	var req struct {
		UserIDs []int64 `json:"user_ids" binding:"required"`
	}

	if err = c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid online status request", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPInvalidRequestResponse(err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context（如果有用户ID的话）
	if len(req.UserIDs) > 0 {
		ctx = tracecontext.WithUserID(ctx, req.UserIDs[0]) // 使用第一个用户ID作为主要用户
	}

	status, err := h.svc.OnlineStatus(ctx, req.UserIDs)
	if err != nil {
		h.log.Error(ctx, "Online status failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorOnlineStatusResponse(err.Error())
	} else {
		resp = h.converter.BuildHTTPOnlineStatusResponse(status)
	}

	httpx.WriteObject(c, resp, err)
}
