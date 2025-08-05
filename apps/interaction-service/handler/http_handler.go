package handler

import (
	"net/http"

	"goim-social/api/rest"
	"goim-social/apps/interaction-service/converter"
	"goim-social/apps/interaction-service/service"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"

	"github.com/gin-gonic/gin"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    log,
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

// DoInteraction 执行互动
func (h *HTTPHandler) DoInteraction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DoInteractionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid do interaction request", logger.F("error", err.Error()))
		res := &rest.DoInteractionResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

	interaction, err := h.svc.DoInteraction(
		ctx,
		req.UserId,
		req.ObjectId,
		objectType,
		interactionType,
		req.Metadata,
	)

	res := &rest.DoInteractionResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "操作成功"
		}(),
		Interaction: func() *rest.Interaction {
			if err != nil || interaction == nil {
				return nil
			}
			return h.converter.InteractionModelToProto(interaction)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Do interaction failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// UndoInteraction 取消互动
func (h *HTTPHandler) UndoInteraction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.UndoInteractionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid undo interaction request", logger.F("error", err.Error()))
		res := &rest.UndoInteractionResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

	err := h.svc.UndoInteraction(
		ctx,
		req.UserId,
		req.ObjectId,
		objectType,
		interactionType,
	)

	res := &rest.UndoInteractionResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "取消成功"
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Undo interaction failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// CheckInteraction 检查互动状态
func (h *HTTPHandler) CheckInteraction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CheckInteractionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid check interaction request", logger.F("error", err.Error()))
		res := &rest.CheckInteractionResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

	hasInteraction, interaction, err := h.svc.CheckInteraction(
		ctx,
		req.UserId,
		req.ObjectId,
		objectType,
		interactionType,
	)

	res := &rest.CheckInteractionResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "查询成功"
		}(),
		HasInteraction: hasInteraction,
		Interaction:    h.converter.InteractionModelToProto(interaction),
	}
	if err != nil {
		h.logger.Error(ctx, "Check interaction failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// BatchCheckInteraction 批量检查互动状态
func (h *HTTPHandler) BatchCheckInteraction(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.BatchCheckInteractionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid batch check interaction request", logger.F("error", err.Error()))
		res := &rest.BatchCheckInteractionResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)
	interactionType := h.converter.InteractionTypeFromProto(req.InteractionType)

	interactions, err := h.svc.BatchCheckInteraction(
		ctx,
		req.UserId,
		req.ObjectIds,
		objectType,
		interactionType,
	)

	res := &rest.BatchCheckInteractionResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "查询成功"
		}(),
		Interactions: interactions,
	}
	if err != nil {
		h.logger.Error(ctx, "Batch check interaction failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// GetObjectStats 获取对象统计
func (h *HTTPHandler) GetObjectStats(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetObjectStatsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get object stats request", logger.F("error", err.Error()))
		res := &rest.GetObjectStatsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)

	stats, err := h.svc.GetObjectStats(ctx, req.ObjectId, objectType)

	res := &rest.GetObjectStatsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Stats: func() *rest.InteractionStats {
			if err != nil || stats == nil {
				return nil
			}
			return h.converter.InteractionStatsModelToProto(stats)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Get object stats failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// GetBatchObjectStats 批量获取对象统计
func (h *HTTPHandler) GetBatchObjectStats(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetBatchObjectStatsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get batch object stats request", logger.F("error", err.Error()))
		res := &rest.GetBatchObjectStatsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	objectType := h.converter.ObjectTypeFromProto(req.InteractionObjectType)

	stats, err := h.svc.GetBatchObjectStats(ctx, req.ObjectIds, objectType)

	var protoStats []*rest.InteractionStats
	if err == nil {
		for _, stat := range stats {
			protoStats = append(protoStats, h.converter.InteractionStatsModelToProto(stat))
		}
	}

	res := &rest.GetBatchObjectStatsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Stats: protoStats,
	}
	if err != nil {
		h.logger.Error(ctx, "Get batch object stats failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
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
