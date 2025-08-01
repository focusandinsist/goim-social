package handler

import (
	"net/http"

	"goim-social/apps/im-gateway-service/service"
	"goim-social/pkg/logger"

	"github.com/gin-gonic/gin"
)

// HTTPHandler HTTP协议处理器
type HTTPHandler struct {
	svc *service.Service
	log logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc: svc,
		log: log,
	}
}

// RegisterRoutes 注册HTTP路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/connect")
	{
		api.POST("/online_status", h.OnlineStatus) // 查询在线状态
	}
}

// OnlineStatus 查询在线状态
func (h *HTTPHandler) OnlineStatus(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		UserIDs []int64 `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error(ctx, "Invalid online status request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, err := h.svc.OnlineStatus(ctx, req.UserIDs)
	if err != nil {
		h.log.Error(ctx, "Online status failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}
