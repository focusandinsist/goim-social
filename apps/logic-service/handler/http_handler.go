package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"websocket-server/api/rest"
	"websocket-server/apps/logic-service/service"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/utils"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	svc    *service.Service
	logger logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:    svc,
		logger: log,
	}
}

// RegisterRoutes 注册HTTP路由,Logic服务是内部服务，只提供健康检查和测试接口
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/logic")
	{
		api.GET("/health", h.HealthCheck)  // 健康检查
		api.POST("/route", h.RouteMessage) // 消息路由测试
	}
}

// RouteMessage 消息路由测试接口
func (h *HTTPHandler) RouteMessage(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		From        int64  `json:"from" binding:"required"`
		To          int64  `json:"to"`
		GroupID     int64  `json:"group_id"`
		Content     string `json:"content" binding:"required"`
		MessageType int32  `json:"message_type"`
		ChatType    int32  `json:"chat_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid route message request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "消息路由失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       result.Success,
		"message":       result.Message,
		"message_id":    result.MessageID,
		"success_count": result.SuccessCount,
		"failure_count": result.FailureCount,
		"failed_users":  result.FailedUsers,
	})
}

// HealthCheck 健康检查
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "logic-service",
		"timestamp": utils.GetCurrentTimestamp(),
	})
}
