package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"websocket-server/apps/friend/model"
	"websocket-server/apps/friend/service"
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
	api := r.Group("/api/v1/friend")
	{
		api.POST("/add", h.AddFriend)
		api.POST("/delete", h.DeleteFriend)
		api.POST("/list", h.ListFriends)
	}
}

func (h *Handler) AddFriend(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.AddFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid add friend request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.AddFriend(ctx, req.UserID, req.FriendID, req.Remark); err != nil {
		h.logger.Error(ctx, "Add friend failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "添加好友成功"})
}

func (h *Handler) DeleteFriend(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.DeleteFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete friend request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.DeleteFriend(ctx, req.UserID, req.FriendID); err != nil {
		h.logger.Error(ctx, "Delete friend failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除好友成功"})
}

func (h *Handler) ListFriends(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.ListFriendsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid list friends request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	friends, err := h.service.ListFriends(ctx, req.UserID)
	if err != nil {
		h.logger.Error(ctx, "List friends failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"friends": friends})
}
