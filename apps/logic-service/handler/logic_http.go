package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
	"goim-social/pkg/utils"
)

// RouteMessage 消息路由测试接口
func (h *HTTPHandler) RouteMessage(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	var req struct {
		From        int64  `json:"from" binding:"required"`
		To          int64  `json:"to"`
		GroupID     int64  `json:"group_id"`
		Content     string `json:"content" binding:"required"`
		MessageType int32  `json:"message_type"`
		ChatType    int32  `json:"chat_type" binding:"required"`
	}

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid route message request", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("请求参数错误: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.From)
	if req.GroupID > 0 {
		ctx = tracecontext.WithGroupID(ctx, req.GroupID)
	}

	// 转换为WSMessage
	wsMsg := &rest.WSMessage{
		From:        req.From,
		To:          req.To,
		GroupId:     req.GroupID,
		Content:     req.Content,
		MessageType: req.MessageType,
	}

	// 处理消息
	result, err := h.svc.ProcessMessage(ctx, wsMsg)
	if err != nil {
		h.logger.Error(ctx, "Route message failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("消息路由失败: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	resp = h.converter.BuildHTTPRouteMessageResponse(result)
	httpx.WriteObject(c, resp, err)
}

// HealthCheck 健康检查
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	resp := h.converter.BuildHTTPHealthResponse("logic-service", utils.GetCurrentTimestamp())
	httpx.WriteObject(c, resp, nil)
}
