package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"websocket-server/apps/message/model"
	"websocket-server/apps/message/service"
	"websocket-server/pkg/logger"
)

type Handler struct {
	service *service.Service
	logger  logger.Logger
}

func NewHandler(service *service.Service, logger logger.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/messages")
	{
		api.POST("/history", h.GetHistory)         // 获取历史消息
		api.POST("/unread", h.GetUnreadMessages)   // 获取未读消息
		api.POST("/mark-read", h.MarkMessagesRead) // 标记消息已读
		api.POST("/send", h.SendMessage)           // 特殊场景下的短连接消息，如测试，某些网络下的备用通道
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

// GetUnreadMessages 获取未读消息
func (h *Handler) GetUnreadMessages(c *gin.Context) {
	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 调用service层获取未读消息
	messages, err := h.service.GetUnreadMessages(c.Request.Context(), req.UserID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "获取未读消息失败", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取未读消息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"total":    len(messages),
	})
}

// MarkMessagesRead 标记消息已读
func (h *Handler) MarkMessagesRead(c *gin.Context) {
	var req struct {
		UserID     int64    `json:"user_id" binding:"required"`
		MessageIDs []string `json:"message_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 调用service层标记消息已读
	err := h.service.MarkMessagesAsRead(c.Request.Context(), req.UserID, req.MessageIDs)
	if err != nil {
		h.logger.Error(c.Request.Context(), "标记消息已读失败", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "标记消息已读失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "标记已读成功",
	})
}
