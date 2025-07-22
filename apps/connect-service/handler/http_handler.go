package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"websocket-server/pkg/logger"
)

// RegisterRoutes 注册HTTP路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/connect")
	{
		api.GET("/ws", h.WebSocketHandler)         // WebSocket长连接
		api.POST("/online_status", h.OnlineStatus) // 查询在线状态
	}
}

// OnlineStatus 查询在线状态
func (h *Handler) OnlineStatus(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		UserIDs []int64 `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid online status request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, err := h.service.OnlineStatus(ctx, req.UserIDs)
	if err != nil {
		h.logger.Error(ctx, "Online status failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}
