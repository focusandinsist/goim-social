package handler

import (
	"net/http"
	"websocket-server/apps/group/service"
	"websocket-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
	logger  logger.Logger
}

func NewHandler(service *service.Service, logger logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/group")
	{
		api.POST("/create", h.CreateGroup)
		api.POST("/add", h.AddToGroup)
		api.POST("/get", h.GetGroup)
		api.POST("/delete", h.DeleteGroup)
		api.POST("/list", h.GetGroupList)
		api.POST("/info", h.GetGroupInfo)
	}
}

// CreateGroup 创建群组
func (h *Handler) CreateGroup(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		OwnerID     int64   `json:"owner_id" binding:"required"`
		MemberIDs   []int64 `json:"member_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid create group request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Create group attempt", logger.F("name", req.Name), logger.F("owner_id", req.OwnerID))

	// 调用service层
	group, err := h.service.CreateGroup(ctx, req.Name, req.Description, req.OwnerID, req.MemberIDs)
	if err != nil {
		h.logger.Error(ctx, "Create group failed", logger.F("name", req.Name), logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Create group successful", logger.F("group_id", group.ID))
	c.JSON(http.StatusOK, gin.H{
		"message": "群组创建成功",
		"data":    group,
	})
}

// AddToGroup 添加成员到群组
func (h *Handler) AddToGroup(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		GroupID int64   `json:"group_id" binding:"required"`
		UserIDs []int64 `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid add to group request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Add members to group", logger.F("group_id", req.GroupID), logger.F("user_count", len(req.UserIDs)))

	err := h.service.AddMembers(ctx, req.GroupID, req.UserIDs)
	if err != nil {
		h.logger.Error(ctx, "Add members failed", logger.F("group_id", req.GroupID), logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Add members successful", logger.F("group_id", req.GroupID))
	c.JSON(http.StatusOK, gin.H{
		"message": "成员添加成功",
	})
}

// GetGroup 获取群组信息
func (h *Handler) GetGroup(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		GroupID int64 `json:"group_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get group request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Get group", logger.F("group_id", req.GroupID))

	group, err := h.service.GetGroup(ctx, req.GroupID)
	if err != nil {
		h.logger.Error(ctx, "Get group failed", logger.F("group_id", req.GroupID), logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取群组信息成功",
		"data":    group,
	})
}

// DeleteGroup 删除群组
func (h *Handler) DeleteGroup(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		GroupID int64 `json:"group_id" binding:"required"`
		UserID  int64 `json:"user_id" binding:"required"` // 操作者ID
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete group request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Delete group", logger.F("group_id", req.GroupID), logger.F("user_id", req.UserID))

	err := h.service.DeleteGroup(ctx, req.GroupID, req.UserID)
	if err != nil {
		h.logger.Error(ctx, "Delete group failed", logger.F("group_id", req.GroupID), logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Delete group successful", logger.F("group_id", req.GroupID))
	c.JSON(http.StatusOK, gin.H{
		"message": "群组删除成功",
	})
}

// GetGroupList 获取群组列表
func (h *Handler) GetGroupList(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
		Page   int   `json:"page"`
		Size   int   `json:"size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get group list request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	h.logger.Info(ctx, "Get group list", logger.F("user_id", req.UserID), logger.F("page", req.Page))

	groups, total, err := h.service.GetGroupList(ctx, req.UserID, req.Page, req.Size)
	if err != nil {
		h.logger.Error(ctx, "Get group list failed", logger.F("user_id", req.UserID), logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取群组列表成功",
		"data": gin.H{
			"groups": groups,
			"total":  total,
			"page":   req.Page,
			"size":   req.Size,
		},
	})
}

// GetGroupInfo 获取群组详细信息
func (h *Handler) GetGroupInfo(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		GroupID int64 `json:"group_id" binding:"required"`
		UserID  int64 `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get group info request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Get group info", logger.F("group_id", req.GroupID), logger.F("user_id", req.UserID))

	groupInfo, err := h.service.GetGroupInfo(ctx, req.GroupID, req.UserID)
	if err != nil {
		h.logger.Error(ctx, "Get group info failed", logger.F("group_id", req.GroupID), logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取群组详细信息成功",
		"data":    groupInfo,
	})
}
