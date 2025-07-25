package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"websocket-server/apps/history-service/model"
	"websocket-server/apps/history-service/service"
	"websocket-server/pkg/logger"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	svc    *service.Service
	logger logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, logger logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:    svc,
		logger: logger,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(engine *gin.Engine) {
	api := engine.Group("/api/v1/history")
	{
		// 基础历史记录操作
		api.POST("/create", h.CreateHistory)
		api.POST("/batch_create", h.BatchCreateHistory)
		api.POST("/user_history", h.GetUserHistory)
		api.POST("/object_history", h.GetObjectHistory)
		api.POST("/delete", h.DeleteHistory)
		api.POST("/clear", h.ClearUserHistory)

		// 统计分析
		api.POST("/user_stats", h.GetUserActionStats)
		api.POST("/hot_objects", h.GetHotObjects)
		api.POST("/activity_stats", h.GetUserActivityStats)
	}
}

// CreateHistoryRequest 创建历史记录请求
type CreateHistoryRequest struct {
	UserID      int64  `json:"user_id" binding:"required"`
	ActionType  string `json:"action_type" binding:"required"`
	ObjectType  string `json:"object_type" binding:"required"`
	ObjectID    int64  `json:"object_id" binding:"required"`
	ObjectTitle string `json:"object_title"`
	ObjectURL   string `json:"object_url"`
	Metadata    string `json:"metadata"`
	DeviceInfo  string `json:"device_info"`
	Location    string `json:"location"`
	Duration    int64  `json:"duration"`
}

// CreateHistory 创建历史记录
func (h *HTTPHandler) CreateHistory(c *gin.Context) {
	var req CreateHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.CreateHistoryParams{
		UserID:      req.UserID,
		ActionType:  req.ActionType,
		ObjectType:  req.ObjectType,
		ObjectID:    req.ObjectID,
		ObjectTitle: req.ObjectTitle,
		ObjectURL:   req.ObjectURL,
		Metadata:    req.Metadata,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		DeviceInfo:  req.DeviceInfo,
		Location:    req.Location,
		Duration:    req.Duration,
	}

	record, err := h.svc.CreateHistory(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to create history",
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
		"message": "历史记录创建成功",
		"data":    record,
	})
}

// BatchCreateHistoryRequest 批量创建历史记录请求
type BatchCreateHistoryRequest struct {
	Records []CreateHistoryRequest `json:"records" binding:"required"`
}

// BatchCreateHistory 批量创建历史记录
func (h *HTTPHandler) BatchCreateHistory(c *gin.Context) {
	var req BatchCreateHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	var paramsList []*model.CreateHistoryParams
	for _, r := range req.Records {
		params := &model.CreateHistoryParams{
			UserID:      r.UserID,
			ActionType:  r.ActionType,
			ObjectType:  r.ObjectType,
			ObjectID:    r.ObjectID,
			ObjectTitle: r.ObjectTitle,
			ObjectURL:   r.ObjectURL,
			Metadata:    r.Metadata,
			IPAddress:   c.ClientIP(),
			UserAgent:   c.GetHeader("User-Agent"),
			DeviceInfo:  r.DeviceInfo,
			Location:    r.Location,
			Duration:    r.Duration,
		}
		paramsList = append(paramsList, params)
	}

	records, err := h.svc.BatchCreateHistory(c.Request.Context(), paramsList)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to batch create history",
			logger.F("error", err.Error()),
			logger.F("count", len(req.Records)))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "批量创建成功",
		"created_count": len(records),
		"data":          records,
	})
}

// GetUserHistoryRequest 获取用户历史记录请求
type GetUserHistoryRequest struct {
	UserID     int64  `json:"user_id" binding:"required"`
	ActionType string `json:"action_type"`
	ObjectType string `json:"object_type"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Page       int32  `json:"page"`
	PageSize   int32  `json:"page_size"`
}

// GetUserHistory 获取用户历史记录
func (h *HTTPHandler) GetUserHistory(c *gin.Context) {
	var req GetUserHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.GetUserHistoryParams{
		UserID:     req.UserID,
		ActionType: req.ActionType,
		ObjectType: req.ObjectType,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	// 解析时间参数
	if req.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			params.StartTime = startTime
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			params.EndTime = endTime
		}
	}

	records, total, err := h.svc.GetUserHistory(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get user history",
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
			"records":   records,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// GetObjectHistoryRequest 获取对象历史记录请求
type GetObjectHistoryRequest struct {
	ObjectType string `json:"object_type" binding:"required"`
	ObjectID   int64  `json:"object_id" binding:"required"`
	ActionType string `json:"action_type"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Page       int32  `json:"page"`
	PageSize   int32  `json:"page_size"`
}

// GetObjectHistory 获取对象历史记录
func (h *HTTPHandler) GetObjectHistory(c *gin.Context) {
	var req GetObjectHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.GetObjectHistoryParams{
		ObjectType: req.ObjectType,
		ObjectID:   req.ObjectID,
		ActionType: req.ActionType,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	// 解析时间参数
	if req.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			params.StartTime = startTime
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			params.EndTime = endTime
		}
	}

	records, total, err := h.svc.GetObjectHistory(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get object history",
			logger.F("error", err.Error()),
			logger.F("objectType", req.ObjectType),
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
			"records":   records,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// DeleteHistoryRequest 删除历史记录请求
type DeleteHistoryRequest struct {
	UserID    int64   `json:"user_id" binding:"required"`
	RecordIDs []int64 `json:"record_ids" binding:"required"`
}

// DeleteHistory 删除历史记录
func (h *HTTPHandler) DeleteHistory(c *gin.Context) {
	var req DeleteHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.DeleteHistoryParams{
		UserID:    req.UserID,
		RecordIDs: req.RecordIDs,
	}

	deletedCount, err := h.svc.DeleteHistory(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to delete history",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "删除成功",
		"deleted_count": deletedCount,
	})
}

// ClearUserHistoryRequest 清空用户历史记录请求
type ClearUserHistoryRequest struct {
	UserID     int64  `json:"user_id" binding:"required"`
	ActionType string `json:"action_type"`
	ObjectType string `json:"object_type"`
	BeforeTime string `json:"before_time"`
}

// ClearUserHistory 清空用户历史记录
func (h *HTTPHandler) ClearUserHistory(c *gin.Context) {
	var req ClearUserHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.ClearUserHistoryParams{
		UserID:     req.UserID,
		ActionType: req.ActionType,
		ObjectType: req.ObjectType,
	}

	// 解析时间参数
	if req.BeforeTime != "" {
		if beforeTime, err := time.Parse(time.RFC3339, req.BeforeTime); err == nil {
			params.BeforeTime = beforeTime
		}
	}

	deletedCount, err := h.svc.ClearUserHistory(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to clear user history",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "清空成功",
		"deleted_count": deletedCount,
	})
}

// GetUserActionStatsRequest 获取用户行为统计请求
type GetUserActionStatsRequest struct {
	UserID     int64  `json:"user_id" binding:"required"`
	ActionType string `json:"action_type"`
}

// GetUserActionStats 获取用户行为统计
func (h *HTTPHandler) GetUserActionStats(c *gin.Context) {
	var req GetUserActionStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	stats, err := h.svc.GetUserActionStats(c.Request.Context(), req.UserID, req.ActionType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get user action stats",
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
		"data":    stats,
	})
}

// GetHotObjectsRequest 获取热门对象请求
type GetHotObjectsRequest struct {
	ObjectType string `json:"object_type"`
	TimeRange  string `json:"time_range"`
	Limit      int32  `json:"limit"`
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

	params := &model.GetHotObjectsParams{
		ObjectType: req.ObjectType,
		TimeRange:  req.TimeRange,
		Limit:      req.Limit,
	}

	objects, err := h.svc.GetHotObjects(c.Request.Context(), params)
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
		"data":    objects,
	})
}

// GetUserActivityStatsRequest 获取用户活跃度统计请求
type GetUserActivityStatsRequest struct {
	UserID    int64  `json:"user_id" binding:"required"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// GetUserActivityStats 获取用户活跃度统计
func (h *HTTPHandler) GetUserActivityStats(c *gin.Context) {
	var req GetUserActivityStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.GetUserActivityStatsParams{
		UserID: req.UserID,
	}

	// 解析日期参数
	if req.StartDate != "" {
		if startDate, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			params.StartDate = startDate
		}
	}
	if req.EndDate != "" {
		if endDate, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			params.EndDate = endDate
		}
	}

	stats, err := h.svc.GetUserActivityStats(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get user activity stats",
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
		"data":    stats,
	})
}
