package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"goim-social/apps/search-service/model"
	"goim-social/pkg/logger"
)

// ============ HTTP响应结构 ============

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ============ 响应辅助方法 ============

// respondSuccess 成功响应
func (h *HTTPHandler) respondSuccess(c *gin.Context, data interface{}) {
	response := Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
	c.JSON(http.StatusOK, response)
}

// respondError 错误响应
func (h *HTTPHandler) respondError(c *gin.Context, statusCode int, message string, error string) {
	response := Response{
		Code:    statusCode,
		Message: message,
		Error:   error,
	}
	c.JSON(statusCode, response)
}

// ============ 请求绑定辅助方法 ============

// bindSearchRequest 绑定搜索请求
func (h *HTTPHandler) bindSearchRequest(c *gin.Context, req *model.SearchRequest) error {
	// 绑定查询参数
	req.Query = c.Query("q")
	if req.Query == "" {
		req.Query = c.Query("query")
	}
	
	if req.Type == "" {
		req.Type = c.DefaultQuery("type", model.SearchTypeContent)
	}
	
	// 分页参数
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		req.Page = page
	} else {
		req.Page = 1
	}
	
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil {
		req.PageSize = pageSize
	} else {
		req.PageSize = 20
	}
	
	// 排序参数
	req.SortBy = c.DefaultQuery("sort_by", model.SortByRelevance)
	req.SortOrder = c.DefaultQuery("sort_order", model.SortOrderDesc)
	
	// 高亮参数
	if highlight, err := strconv.ParseBool(c.DefaultQuery("highlight", "true")); err == nil {
		req.Highlight = highlight
	} else {
		req.Highlight = true
	}
	
	// 用户ID
	req.UserID = h.getUserID(c)
	
	// 过滤器参数
	req.Filters = make(map[string]string)
	
	// 解析过滤器参数
	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "filter_") {
			filterKey := strings.TrimPrefix(key, "filter_")
			if len(values) > 0 {
				req.Filters[filterKey] = values[0]
			}
		}
	}
	
	// 特定过滤器
	if authorID := c.Query("author_id"); authorID != "" {
		req.Filters["author_id"] = authorID
	}
	
	if category := c.Query("category"); category != "" {
		req.Filters["category"] = category
	}
	
	if status := c.Query("status"); status != "" {
		req.Filters["status"] = status
	}
	
	if tags := c.Query("tags"); tags != "" {
		req.Filters["tags"] = tags
	}
	
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		req.Filters["date_from"] = dateFrom
	}
	
	if dateTo := c.Query("date_to"); dateTo != "" {
		req.Filters["date_to"] = dateTo
	}
	
	if isPublic := c.Query("is_public"); isPublic != "" {
		req.Filters["is_public"] = isPublic
	}
	
	if messageType := c.Query("message_type"); messageType != "" {
		req.Filters["message_type"] = messageType
	}
	
	if groupID := c.Query("group_id"); groupID != "" {
		req.Filters["group_id"] = groupID
	}
	
	return nil
}

// ============ 用户认证辅助方法 ============

// getUserID 获取用户ID
func (h *HTTPHandler) getUserID(c *gin.Context) int64 {
	// 从JWT token中获取用户ID
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(int64); ok {
			return uid
		}
		if uid, ok := userID.(float64); ok {
			return int64(uid)
		}
		if uid, ok := userID.(string); ok {
			if parsedUID, err := strconv.ParseInt(uid, 10, 64); err == nil {
				return parsedUID
			}
		}
	}
	
	// 从Header中获取用户ID（用于测试）
	if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
		if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			return userID
		}
	}
	
	// 从查询参数中获取用户ID（用于测试，生产环境应该移除）
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			return userID
		}
	}
	
	return 0
}

// isAdmin 检查是否为管理员
func (h *HTTPHandler) isAdmin(c *gin.Context) bool {
	// 从JWT token中获取用户角色
	if role, exists := c.Get("user_role"); exists {
		if roleStr, ok := role.(string); ok {
			return roleStr == "admin" || roleStr == "super_admin"
		}
	}
	
	// 从Header中获取管理员标识（用于测试）
	if adminStr := c.GetHeader("X-Admin"); adminStr != "" {
		if isAdmin, err := strconv.ParseBool(adminStr); err == nil {
			return isAdmin
		}
	}
	
	return false
}

// requireAuth 需要认证的中间件
func (h *HTTPHandler) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := h.getUserID(c)
		if userID <= 0 {
			h.respondError(c, http.StatusUnauthorized, "authentication required", "user not authenticated")
			c.Abort()
			return
		}
		c.Next()
	}
}

// requireAdmin 需要管理员权限的中间件
func (h *HTTPHandler) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.isAdmin(c) {
			h.respondError(c, http.StatusForbidden, "admin access required", "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

// ============ 请求日志中间件 ============

// requestLogger 请求日志中间件
func (h *HTTPHandler) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := c.GetTime("start_time")
		if start.IsZero() {
			start = c.GetTime("request_start")
		}
		
		// 记录请求开始
		h.logger.Info(c.Request.Context(), "HTTP request started",
			logger.F("method", c.Request.Method),
			logger.F("path", c.Request.URL.Path),
			logger.F("query", c.Request.URL.RawQuery),
			logger.F("client_ip", c.ClientIP()),
			logger.F("user_agent", c.Request.UserAgent()),
			logger.F("user_id", h.getUserID(c)))
		
		c.Next()
		
		// 记录请求结束
		duration := c.GetDuration("duration")
		statusCode := c.Writer.Status()
		
		logLevel := "info"
		if statusCode >= 400 && statusCode < 500 {
			logLevel = "warn"
		} else if statusCode >= 500 {
			logLevel = "error"
		}
		
		fields := []logger.Field{
			logger.F("method", c.Request.Method),
			logger.F("path", c.Request.URL.Path),
			logger.F("status_code", statusCode),
			logger.F("duration_ms", duration.Milliseconds()),
			logger.F("response_size", c.Writer.Size()),
			logger.F("user_id", h.getUserID(c)),
		}
		
		switch logLevel {
		case "info":
			h.logger.Info(c.Request.Context(), "HTTP request completed", fields...)
		case "warn":
			h.logger.Warn(c.Request.Context(), "HTTP request completed with warning", fields...)
		case "error":
			h.logger.Error(c.Request.Context(), "HTTP request completed with error", fields...)
		}
	}
}

// ============ 错误处理中间件 ============

// errorHandler 错误处理中间件
func (h *HTTPHandler) errorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error(c.Request.Context(), "HTTP request panic",
					logger.F("error", err),
					logger.F("method", c.Request.Method),
					logger.F("path", c.Request.URL.Path),
					logger.F("user_id", h.getUserID(c)))
				
				h.respondError(c, http.StatusInternalServerError, "internal server error", "unexpected error occurred")
				c.Abort()
			}
		}()
		
		c.Next()
	}
}

// ============ CORS中间件 ============

// corsHandler CORS处理中间件
func (h *HTTPHandler) corsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-ID, X-Admin")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// ============ 限流中间件 ============

// rateLimiter 简单的限流中间件
func (h *HTTPHandler) rateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 这里可以实现基于IP或用户的限流逻辑
		// 暂时跳过实现
		c.Next()
	}
}
