package handler

import (
	"net/http"
	"websocket-server/apps/message/model"
	"websocket-server/apps/message/service"
	"websocket-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
	logger  logger.Logger
}

func NewHandler(service *service.Service, logger logger.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/message")
	{
		api.POST("/send", h.SendMessage)
		api.POST("/history", h.GetHistory)
	}
}

// SendMessage 发送消息
func (h *Handler) SendMessage(c *gin.Context) {
	ctx := c.Request.Context()
	var msg model.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		h.logger.Error(ctx, "Invalid send message request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.SendMessage(ctx, &msg); err != nil {
		h.logger.Error(ctx, "Send message failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "消息发送成功"})
}

// GetHistory 获取历史消息
func (h *Handler) GetHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		UserID  int64 `json:"user_id"`
		GroupID int64 `json:"group_id"`
		Page    int   `json:"page"`
		Size    int   `json:"size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get history request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	msgs, total, err := h.service.GetHistory(ctx, req.UserID, req.GroupID, req.Page, req.Size)
	if err != nil {
		h.logger.Error(ctx, "Get history failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs, "total": total, "page": req.Page, "size": req.Size})
}
