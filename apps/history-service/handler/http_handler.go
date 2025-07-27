package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"websocket-server/api/rest"
	"websocket-server/apps/history-service/model"
	"websocket-server/apps/history-service/service"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/utils"
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

// CreateHistory 创建历史记录
func (h *HTTPHandler) CreateHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create history request", logger.F("error", err.Error()))
		res := &rest.CreateHistoryResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	actionType := convertActionTypeFromProto(req.ActionType)
	objectType := convertHistoryObjectTypeFromProto(req.HistoryObjectType)

	params := &model.CreateHistoryParams{
		UserID:      req.UserId,
		ActionType:  actionType,
		ObjectType:  objectType,
		ObjectID:    req.ObjectId,
		ObjectTitle: req.ObjectTitle,
		ObjectURL:   req.ObjectUrl,
		Metadata:    req.Metadata,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		DeviceInfo:  req.DeviceInfo,
		Location:    req.Location,
		Duration:    req.Duration,
	}

	record, err := h.svc.CreateHistory(ctx, params)
	res := &rest.CreateHistoryResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "历史记录创建成功"
		}(),
		Record: func() *rest.HistoryRecord {
			if err != nil {
				return nil
			}
			return convertHistoryRecordToProto(record)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Create history failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// BatchCreateHistory 批量创建历史记录
func (h *HTTPHandler) BatchCreateHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.BatchCreateHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid batch create history request", logger.F("error", err.Error()))
		res := &rest.BatchCreateHistoryResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	var paramsList []*model.CreateHistoryParams
	for i := range req.Records {
		r := req.Records[i] // 避免range var copies lock问题
		// 转换枚举类型
		actionType := convertActionTypeFromProto(r.ActionType)
		objectType := convertHistoryObjectTypeFromProto(r.HistoryObjectType)

		params := &model.CreateHistoryParams{
			UserID:      r.UserId,
			ActionType:  actionType,
			ObjectType:  objectType,
			ObjectID:    r.ObjectId,
			ObjectTitle: r.ObjectTitle,
			ObjectURL:   r.ObjectUrl,
			Metadata:    r.Metadata,
			IPAddress:   c.ClientIP(),
			UserAgent:   c.GetHeader("User-Agent"),
			DeviceInfo:  r.DeviceInfo,
			Location:    r.Location,
			Duration:    r.Duration,
		}
		paramsList = append(paramsList, params)
	}

	records, err := h.svc.BatchCreateHistory(ctx, paramsList)

	res := &rest.BatchCreateHistoryResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "批量创建成功"
		}(),
		CreatedCount: int32(len(records)),
		Records: func() []*rest.HistoryRecord {
			if err != nil {
				return nil
			}
			var protoRecords []*rest.HistoryRecord
			for _, record := range records {
				protoRecords = append(protoRecords, convertHistoryRecordToProto(record))
			}
			return protoRecords
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Batch create history failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetUserHistory 获取用户历史记录
func (h *HTTPHandler) GetUserHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user history request", logger.F("error", err.Error()))
		res := &rest.GetUserHistoryResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	actionType := convertActionTypeFromProto(req.ActionType)
	objectType := convertHistoryObjectTypeFromProto(req.HistoryObjectType)

	params := &model.GetUserHistoryParams{
		UserID:     req.UserId,
		ActionType: actionType,
		ObjectType: objectType,
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

	records, total, err := h.svc.GetUserHistory(ctx, params)

	// 转换历史记录列表
	var protoRecords []*rest.HistoryRecord
	if err == nil {
		for _, record := range records {
			protoRecords = append(protoRecords, convertHistoryRecordToProto(record))
		}
	}

	res := &rest.GetUserHistoryResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Records:  protoRecords,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if err != nil {
		h.logger.Error(ctx, "Get user history failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
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

// DeleteHistory 删除历史记录
func (h *HTTPHandler) DeleteHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete history request", logger.F("error", err.Error()))
		res := &rest.DeleteHistoryResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	params := &model.DeleteHistoryParams{
		UserID:    req.UserId,
		RecordIDs: req.RecordIds,
	}

	deletedCount, err := h.svc.DeleteHistory(ctx, params)
	res := &rest.DeleteHistoryResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "删除成功"
		}(),
		DeletedCount: func() int32 {
			if err != nil {
				return 0
			}
			return deletedCount
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Delete history failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// ClearUserHistory 清空用户历史记录
func (h *HTTPHandler) ClearUserHistory(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ClearUserHistoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid clear user history request", logger.F("error", err.Error()))
		res := &rest.ClearUserHistoryResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换枚举类型
	actionType := convertActionTypeFromProto(req.ActionType)
	objectType := convertHistoryObjectTypeFromProto(req.HistoryObjectType)

	params := &model.ClearUserHistoryParams{
		UserID:     req.UserId,
		ActionType: actionType,
		ObjectType: objectType,
	}

	// 解析时间参数
	if req.BeforeTime != "" {
		if beforeTime, err := time.Parse(time.RFC3339, req.BeforeTime); err == nil {
			params.BeforeTime = beforeTime
		}
	}

	deletedCount, err := h.svc.ClearUserHistory(ctx, params)
	res := &rest.ClearUserHistoryResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "清空成功"
		}(),
		DeletedCount: func() int32 {
			if err != nil {
				return 0
			}
			return deletedCount
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Clear user history failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
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

// convertHistoryObjectTypeFromProto 将protobuf枚举转换为对象类型
func convertHistoryObjectTypeFromProto(objectType rest.HistoryObjectType) string {
	switch objectType {
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_POST:
		return "post"
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_ARTICLE:
		return "article"
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_VIDEO:
		return "video"
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_USER:
		return "user"
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_PRODUCT:
		return "product"
	case rest.HistoryObjectType_HISTORY_OBJECT_TYPE_GROUP:
		return "group"
	default:
		return "post"
	}
}
