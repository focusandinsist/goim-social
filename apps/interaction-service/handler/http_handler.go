package handler

import (
	"net/http"

	"websocket-server/apps/interaction-service/service"
	"websocket-server/pkg/logger"

	"github.com/gin-gonic/gin"
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

// RegisterRoutes 注册HTTP路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/interaction")
	{
		// 基础互动操作
		api.POST("/do", h.DoInteraction)                  // 执行互动
		api.POST("/undo", h.UndoInteraction)              // 取消互动
		api.POST("/check", h.CheckInteraction)            // 检查互动状态
		api.POST("/batch_check", h.BatchCheckInteraction) // 批量检查互动状态

		// 统计查询
		api.POST("/stats", h.GetObjectStats)                     // 获取对象统计
		api.POST("/batch_stats", h.GetBatchObjectStats)          // 批量获取对象统计
		api.POST("/summary", h.GetInteractionSummary)            // 获取互动汇总
		api.POST("/batch_summary", h.BatchGetInteractionSummary) // 批量获取互动汇总

		// 列表查询
		api.POST("/user_interactions", h.GetUserInteractions)     // 获取用户互动列表
		api.POST("/object_interactions", h.GetObjectInteractions) // 获取对象互动列表
		api.POST("/hot_objects", h.GetHotObjects)                 // 获取热门对象
	}
}

// DoInteractionRequest 执行互动请求
type DoInteractionRequest struct {
	UserID          int64  `json:"user_id" binding:"required"`
	ObjectID        int64  `json:"object_id" binding:"required"`
	ObjectType      string `json:"object_type" binding:"required"`
	InteractionType string `json:"interaction_type" binding:"required"`
	Metadata        string `json:"metadata"`
}

// DoInteraction 执行互动
func (h *HTTPHandler) DoInteraction(c *gin.Context) {
	var req DoInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	interaction, err := h.svc.DoInteraction(
		c.Request.Context(),
		req.UserID,
		req.ObjectID,
		req.ObjectType,
		req.InteractionType,
		req.Metadata,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to do interaction",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserID),
			logger.F("objectID", req.ObjectID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "操作成功",
		"data":    interaction,
	})
}

// UndoInteractionRequest 取消互动请求
type UndoInteractionRequest struct {
	UserID          int64  `json:"user_id" binding:"required"`
	ObjectID        int64  `json:"object_id" binding:"required"`
	ObjectType      string `json:"object_type" binding:"required"`
	InteractionType string `json:"interaction_type" binding:"required"`
}

// UndoInteraction 取消互动
func (h *HTTPHandler) UndoInteraction(c *gin.Context) {
	var req UndoInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	err := h.svc.UndoInteraction(
		c.Request.Context(),
		req.UserID,
		req.ObjectID,
		req.ObjectType,
		req.InteractionType,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to undo interaction",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserID),
			logger.F("objectID", req.ObjectID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "取消成功",
	})
}

// CheckInteractionRequest 检查互动请求
type CheckInteractionRequest struct {
	UserID          int64  `json:"user_id" binding:"required"`
	ObjectID        int64  `json:"object_id" binding:"required"`
	ObjectType      string `json:"object_type" binding:"required"`
	InteractionType string `json:"interaction_type" binding:"required"`
}

// CheckInteraction 检查互动状态
func (h *HTTPHandler) CheckInteraction(c *gin.Context) {
	var req CheckInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	hasInteraction, interaction, err := h.svc.CheckInteraction(
		c.Request.Context(),
		req.UserID,
		req.ObjectID,
		req.ObjectType,
		req.InteractionType,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to check interaction",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserID),
			logger.F("objectID", req.ObjectID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"message":         "查询成功",
		"has_interaction": hasInteraction,
		"interaction":     interaction,
	})
}

// BatchCheckInteractionRequest 批量检查互动请求
type BatchCheckInteractionRequest struct {
	UserID          int64   `json:"user_id" binding:"required"`
	ObjectIDs       []int64 `json:"object_ids" binding:"required"`
	ObjectType      string  `json:"object_type" binding:"required"`
	InteractionType string  `json:"interaction_type" binding:"required"`
}

// BatchCheckInteraction 批量检查互动状态
func (h *HTTPHandler) BatchCheckInteraction(c *gin.Context) {
	var req BatchCheckInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	interactions, err := h.svc.BatchCheckInteraction(
		c.Request.Context(),
		req.UserID,
		req.ObjectIDs,
		req.ObjectType,
		req.InteractionType,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to batch check interaction",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserID),
			logger.F("objectIDs", req.ObjectIDs))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "查询成功",
		"interactions": interactions,
	})
}

// GetObjectStatsRequest 获取对象统计请求
type GetObjectStatsRequest struct {
	ObjectID   int64  `json:"object_id" binding:"required"`
	ObjectType string `json:"object_type" binding:"required"`
}

// GetObjectStats 获取对象统计
func (h *HTTPHandler) GetObjectStats(c *gin.Context) {
	var req GetObjectStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	stats, err := h.svc.GetObjectStats(c.Request.Context(), req.ObjectID, req.ObjectType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get object stats",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data":    stats,
	})
}

// GetBatchObjectStatsRequest 批量获取对象统计请求
type GetBatchObjectStatsRequest struct {
	ObjectIDs  []int64 `json:"object_ids" binding:"required"`
	ObjectType string  `json:"object_type" binding:"required"`
}

// GetBatchObjectStats 批量获取对象统计
func (h *HTTPHandler) GetBatchObjectStats(c *gin.Context) {
	var req GetBatchObjectStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	stats, err := h.svc.GetBatchObjectStats(c.Request.Context(), req.ObjectIDs, req.ObjectType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get batch object stats",
			logger.F("error", err.Error()),
			logger.F("objectIDs", req.ObjectIDs))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data":    stats,
	})
}

// GetInteractionSummaryRequest 获取互动汇总请求
type GetInteractionSummaryRequest struct {
	ObjectID   int64  `json:"object_id" binding:"required"`
	UserID     int64  `json:"user_id"`
	ObjectType string `json:"object_type" binding:"required"`
}

// GetInteractionSummary 获取互动汇总
func (h *HTTPHandler) GetInteractionSummary(c *gin.Context) {
	var req GetInteractionSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	summary, err := h.svc.GetInteractionSummary(c.Request.Context(), req.ObjectID, req.UserID, req.ObjectType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get interaction summary",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data":    summary,
	})
}

// BatchGetInteractionSummaryRequest 批量获取互动汇总请求
type BatchGetInteractionSummaryRequest struct {
	ObjectIDs  []int64 `json:"object_ids" binding:"required"`
	UserID     int64   `json:"user_id"`
	ObjectType string  `json:"object_type" binding:"required"`
}

// BatchGetInteractionSummary 批量获取互动汇总
func (h *HTTPHandler) BatchGetInteractionSummary(c *gin.Context) {
	var req BatchGetInteractionSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	summaries, err := h.svc.BatchGetInteractionSummary(c.Request.Context(), req.ObjectIDs, req.UserID, req.ObjectType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to batch get interaction summary",
			logger.F("error", err.Error()),
			logger.F("objectIDs", req.ObjectIDs))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data":    summaries,
	})
}

// GetUserInteractionsRequest 获取用户互动列表请求
type GetUserInteractionsRequest struct {
	UserID          int64  `json:"user_id" binding:"required"`
	ObjectType      string `json:"object_type"`
	InteractionType string `json:"interaction_type"`
	Page            int32  `json:"page"`
	PageSize        int32  `json:"page_size"`
}

// GetUserInteractions 获取用户互动列表
func (h *HTTPHandler) GetUserInteractions(c *gin.Context) {
	var req GetUserInteractionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	interactions, total, err := h.svc.GetUserInteractions(
		c.Request.Context(),
		req.UserID,
		req.ObjectType,
		req.InteractionType,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get user interactions",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data": gin.H{
			"interactions": interactions,
			"total":        total,
			"page":         req.Page,
			"page_size":    req.PageSize,
		},
	})
}

// GetObjectInteractionsRequest 获取对象互动列表请求
type GetObjectInteractionsRequest struct {
	ObjectID        int64  `json:"object_id" binding:"required"`
	ObjectType      string `json:"object_type" binding:"required"`
	InteractionType string `json:"interaction_type"`
	Page            int32  `json:"page"`
	PageSize        int32  `json:"page_size"`
}

// GetObjectInteractions 获取对象互动列表
func (h *HTTPHandler) GetObjectInteractions(c *gin.Context) {
	var req GetObjectInteractionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	interactions, total, err := h.svc.GetObjectInteractions(
		c.Request.Context(),
		req.ObjectID,
		req.ObjectType,
		req.InteractionType,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get object interactions",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data": gin.H{
			"interactions": interactions,
			"total":        total,
			"page":         req.Page,
			"page_size":    req.PageSize,
		},
	})
}

// GetHotObjectsRequest 获取热门对象请求
type GetHotObjectsRequest struct {
	ObjectType      string `json:"object_type" binding:"required"`
	InteractionType string `json:"interaction_type"`
	Limit           int32  `json:"limit"`
}

// GetHotObjects 获取热门对象
func (h *HTTPHandler) GetHotObjects(c *gin.Context) {
	var req GetHotObjectsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	hotObjects, err := h.svc.GetHotObjects(
		c.Request.Context(),
		req.ObjectType,
		req.InteractionType,
		req.Limit,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get hot objects",
			logger.F("error", err.Error()),
			logger.F("objectType", req.ObjectType))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data":    hotObjects,
	})
}
