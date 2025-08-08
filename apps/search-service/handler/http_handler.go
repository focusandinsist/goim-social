package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"goim-social/apps/search-service/model"
	"goim-social/apps/search-service/service"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	searchService service.SearchService
	indexService  service.IndexService
	logger        logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(
	searchService service.SearchService,
	indexService service.IndexService,
	log logger.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		searchService: searchService,
		indexService:  indexService,
		logger:        log,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	// 搜索相关路由
	search := api.Group("/search")
	{
		search.GET("/", h.Search)
		search.GET("/content", h.SearchContent)
		search.GET("/users", h.SearchUsers)
		search.GET("/messages", h.SearchMessages)
		search.GET("/groups", h.SearchGroups)
		search.GET("/multi", h.MultiSearch)
		search.GET("/suggestions", h.GetSuggestions)
		search.GET("/autocomplete", h.GetAutoComplete)
		search.GET("/hot", h.GetHotSearches)
	}

	// 搜索历史相关路由
	history := api.Group("/search/history")
	{
		history.GET("/", h.GetUserSearchHistory)
		history.DELETE("/", h.ClearUserSearchHistory)
		history.DELETE("/:id", h.DeleteSearchHistory)
	}

	// 用户偏好相关路由
	preference := api.Group("/search/preference")
	{
		preference.GET("/", h.GetUserPreference)
		preference.PUT("/", h.UpdateUserPreference)
	}

	// 搜索统计相关路由
	stats := api.Group("/search/stats")
	{
		stats.GET("/", h.GetSearchStats)
		stats.GET("/analytics", h.GetSearchAnalytics)
	}

	// 索引管理相关路由（管理员接口）
	admin := api.Group("/admin")
	{
		index := admin.Group("/index")
		{
			index.POST("/", h.CreateIndex)
			index.DELETE("/:name", h.DeleteIndex)
			index.POST("/reindex", h.ReindexAll)
			index.POST("/reindex/:type", h.ReindexByType)
		}

		document := admin.Group("/document")
		{
			document.POST("/:type/:id", h.IndexDocument)
			document.PUT("/:type/:id", h.UpdateDocument)
			document.DELETE("/:type/:id", h.DeleteDocument)
			document.POST("/:type/bulk", h.BulkIndexDocuments)
		}

		sync := admin.Group("/sync")
		{
			sync.POST("/", h.SyncFromDatabase)
			sync.GET("/status", h.GetSyncStatus)
			sync.GET("/statuses", h.ListSyncStatuses)
		}
	}

	// 健康检查
	api.GET("/health", h.HealthCheck)
	api.GET("/cluster/info", h.GetClusterInfo)
}

// ============ 搜索接口 ============

// Search 通用搜索
func (h *HTTPHandler) Search(c *gin.Context) {
	req := &model.SearchRequest{}
	if err := h.bindSearchRequest(c, req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	response, err := h.searchService.Search(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Search failed",
			logger.F("query", req.Query),
			logger.F("type", req.Type),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "search failed", err.Error())
		return
	}

	h.respondSuccess(c, response)
}

// SearchContent 内容搜索
func (h *HTTPHandler) SearchContent(c *gin.Context) {
	req := &model.SearchRequest{Type: model.SearchTypeContent}
	if err := h.bindSearchRequest(c, req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	results, total, err := h.searchService.SearchContent(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Content search failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "content search failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"query":     req.Query,
		"type":      req.Type,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
		"results":   results,
	}

	h.respondSuccess(c, response)
}

// SearchUsers 用户搜索
func (h *HTTPHandler) SearchUsers(c *gin.Context) {
	req := &model.SearchRequest{Type: model.SearchTypeUser}
	if err := h.bindSearchRequest(c, req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	results, total, err := h.searchService.SearchUsers(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "User search failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "user search failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"query":     req.Query,
		"type":      req.Type,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
		"results":   results,
	}

	h.respondSuccess(c, response)
}

// SearchMessages 消息搜索
func (h *HTTPHandler) SearchMessages(c *gin.Context) {
	req := &model.SearchRequest{Type: model.SearchTypeMessage}
	if err := h.bindSearchRequest(c, req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	// 消息搜索需要用户ID
	if req.UserID <= 0 {
		h.respondError(c, http.StatusBadRequest, "user ID is required for message search", "")
		return
	}

	results, total, err := h.searchService.SearchMessages(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Message search failed",
			logger.F("query", req.Query),
			logger.F("user_id", req.UserID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "message search failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"query":     req.Query,
		"type":      req.Type,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
		"results":   results,
	}

	h.respondSuccess(c, response)
}

// SearchGroups 群组搜索
func (h *HTTPHandler) SearchGroups(c *gin.Context) {
	req := &model.SearchRequest{Type: model.SearchTypeGroup}
	if err := h.bindSearchRequest(c, req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	results, total, err := h.searchService.SearchGroups(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Group search failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "group search failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"query":     req.Query,
		"type":      req.Type,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
		"results":   results,
	}

	h.respondSuccess(c, response)
}

// MultiSearch 多类型搜索
func (h *HTTPHandler) MultiSearch(c *gin.Context) {
	req := &model.SearchRequest{Type: model.SearchTypeAll}
	if err := h.bindSearchRequest(c, req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	response, err := h.searchService.MultiSearch(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Multi search failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "multi search failed", err.Error())
		return
	}

	h.respondSuccess(c, response)
}

// GetSuggestions 获取搜索建议
func (h *HTTPHandler) GetSuggestions(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		h.respondError(c, http.StatusBadRequest, "query parameter is required", "")
		return
	}

	searchType := c.DefaultQuery("type", model.SearchTypeContent)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	userID := h.getUserID(c)

	suggestions, err := h.searchService.GetSuggestions(c.Request.Context(), query, searchType, limit, userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get suggestions failed",
			logger.F("query", query),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get suggestions failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"query":       query,
		"suggestions": suggestions,
	}

	h.respondSuccess(c, response)
}

// GetAutoComplete 获取自动完成建议
func (h *HTTPHandler) GetAutoComplete(c *gin.Context) {
	req := &model.AutoCompleteRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	if req.Query == "" {
		h.respondError(c, http.StatusBadRequest, "query parameter is required", "")
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	req.UserID = h.getUserID(c)

	response, err := h.searchService.GetAutoComplete(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get autocomplete failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get autocomplete failed", err.Error())
		return
	}

	h.respondSuccess(c, response)
}

// GetHotSearches 获取热门搜索
func (h *HTTPHandler) GetHotSearches(c *gin.Context) {
	searchType := c.DefaultQuery("type", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	hotSearches, err := h.searchService.GetHotSearches(c.Request.Context(), searchType, limit)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get hot searches failed",
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get hot searches failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"hot_searches": hotSearches,
	}

	h.respondSuccess(c, response)
}

// ============ 搜索历史接口 ============

// GetUserSearchHistory 获取用户搜索历史
func (h *HTTPHandler) GetUserSearchHistory(c *gin.Context) {
	userID := h.getUserID(c)
	if userID <= 0 {
		h.respondError(c, http.StatusUnauthorized, "user not authenticated", "")
		return
	}

	searchType := c.Query("type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	history, err := h.searchService.GetUserSearchHistory(c.Request.Context(), userID, searchType, limit)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get user search history failed",
			logger.F("user_id", userID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get user search history failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"history": history,
	}

	h.respondSuccess(c, response)
}

// ClearUserSearchHistory 清空用户搜索历史
func (h *HTTPHandler) ClearUserSearchHistory(c *gin.Context) {
	userID := h.getUserID(c)
	if userID <= 0 {
		h.respondError(c, http.StatusUnauthorized, "user not authenticated", "")
		return
	}

	searchType := c.Query("type")

	err := h.searchService.ClearUserSearchHistory(c.Request.Context(), userID, searchType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Clear user search history failed",
			logger.F("user_id", userID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "clear user search history failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "search history cleared successfully",
	})
}

// DeleteSearchHistory 删除特定搜索历史
func (h *HTTPHandler) DeleteSearchHistory(c *gin.Context) {
	userID := h.getUserID(c)
	if userID <= 0 {
		h.respondError(c, http.StatusUnauthorized, "user not authenticated", "")
		return
	}

	historyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid history ID", err.Error())
		return
	}

	err = h.searchService.DeleteSearchHistory(c.Request.Context(), userID, historyID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Delete search history failed",
			logger.F("user_id", userID),
			logger.F("history_id", historyID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "delete search history failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "search history deleted successfully",
	})
}

// ============ 用户偏好接口 ============

// GetUserPreference 获取用户搜索偏好
func (h *HTTPHandler) GetUserPreference(c *gin.Context) {
	userID := h.getUserID(c)
	if userID <= 0 {
		h.respondError(c, http.StatusUnauthorized, "user not authenticated", "")
		return
	}

	preference, err := h.searchService.GetUserPreference(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get user preference failed",
			logger.F("user_id", userID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get user preference failed", err.Error())
		return
	}

	h.respondSuccess(c, preference)
}

// UpdateUserPreference 更新用户搜索偏好
func (h *HTTPHandler) UpdateUserPreference(c *gin.Context) {
	userID := h.getUserID(c)
	if userID <= 0 {
		h.respondError(c, http.StatusUnauthorized, "user not authenticated", "")
		return
	}

	var preference model.UserSearchPreference
	if err := c.ShouldBindJSON(&preference); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	preference.UserID = userID

	err := h.searchService.UpdateUserPreference(c.Request.Context(), &preference)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Update user preference failed",
			logger.F("user_id", userID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "update user preference failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "user preference updated successfully",
	})
}

// ============ 搜索统计接口 ============

// GetSearchStats 获取搜索统计
func (h *HTTPHandler) GetSearchStats(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "24h")

	stats, err := h.searchService.GetSearchStats(c.Request.Context(), timeRange)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get search stats failed",
			logger.F("time_range", timeRange),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get search stats failed", err.Error())
		return
	}

	h.respondSuccess(c, stats)
}

// GetSearchAnalytics 获取搜索分析数据
func (h *HTTPHandler) GetSearchAnalytics(c *gin.Context) {
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	searchType := c.Query("type")

	analytics, err := h.searchService.GetSearchAnalytics(c.Request.Context(), startTime, endTime, searchType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get search analytics failed",
			logger.F("start_time", startTime),
			logger.F("end_time", endTime),
			logger.F("search_type", searchType),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get search analytics failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"analytics": analytics,
	}

	h.respondSuccess(c, response)
}

// ============ 管理员接口 ============

// CreateIndex 创建索引
func (h *HTTPHandler) CreateIndex(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	var req struct {
		IndexName string `json:"index_name" binding:"required"`
		IndexType string `json:"index_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	err := h.indexService.CreateIndex(c.Request.Context(), req.IndexName, req.IndexType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Create index failed",
			logger.F("index_name", req.IndexName),
			logger.F("index_type", req.IndexType),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "create index failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "index created successfully",
	})
}

// DeleteIndex 删除索引
func (h *HTTPHandler) DeleteIndex(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	indexName := c.Param("name")
	if indexName == "" {
		h.respondError(c, http.StatusBadRequest, "index name is required", "")
		return
	}

	err := h.indexService.DeleteIndex(c.Request.Context(), indexName)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Delete index failed",
			logger.F("index_name", indexName),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "delete index failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "index deleted successfully",
	})
}

// ReindexAll 重建所有索引
func (h *HTTPHandler) ReindexAll(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	err := h.indexService.ReindexAll(c.Request.Context())
	if err != nil {
		h.logger.Error(c.Request.Context(), "Reindex all failed",
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "reindex all failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "reindex all started successfully",
	})
}

// ReindexByType 按类型重建索引
func (h *HTTPHandler) ReindexByType(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	indexType := c.Param("type")
	if indexType == "" {
		h.respondError(c, http.StatusBadRequest, "index type is required", "")
		return
	}

	err := h.indexService.ReindexByType(c.Request.Context(), indexType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Reindex by type failed",
			logger.F("index_type", indexType),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "reindex by type failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "reindex by type started successfully",
	})
}

// IndexDocument 索引文档
func (h *HTTPHandler) IndexDocument(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	indexType := c.Param("type")
	docID := c.Param("id")

	if indexType == "" || docID == "" {
		h.respondError(c, http.StatusBadRequest, "index type and document ID are required", "")
		return
	}

	var document map[string]interface{}
	if err := c.ShouldBindJSON(&document); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid document", err.Error())
		return
	}

	err := h.indexService.IndexDocument(c.Request.Context(), indexType, docID, document)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Index document failed",
			logger.F("index_type", indexType),
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "index document failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "document indexed successfully",
	})
}

// UpdateDocument 更新文档
func (h *HTTPHandler) UpdateDocument(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	indexType := c.Param("type")
	docID := c.Param("id")

	if indexType == "" || docID == "" {
		h.respondError(c, http.StatusBadRequest, "index type and document ID are required", "")
		return
	}

	var document map[string]interface{}
	if err := c.ShouldBindJSON(&document); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid document", err.Error())
		return
	}

	err := h.indexService.UpdateDocument(c.Request.Context(), indexType, docID, document)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Update document failed",
			logger.F("index_type", indexType),
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "update document failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "document updated successfully",
	})
}

// DeleteDocument 删除文档
func (h *HTTPHandler) DeleteDocument(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	indexType := c.Param("type")
	docID := c.Param("id")

	if indexType == "" || docID == "" {
		h.respondError(c, http.StatusBadRequest, "index type and document ID are required", "")
		return
	}

	err := h.indexService.DeleteDocument(c.Request.Context(), indexType, docID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Delete document failed",
			logger.F("index_type", indexType),
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "delete document failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "document deleted successfully",
	})
}

// BulkIndexDocuments 批量索引文档
func (h *HTTPHandler) BulkIndexDocuments(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	indexType := c.Param("type")
	if indexType == "" {
		h.respondError(c, http.StatusBadRequest, "index type is required", "")
		return
	}

	var req struct {
		Documents []service.IndexDocument `json:"documents" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	err := h.indexService.BulkIndexDocuments(c.Request.Context(), indexType, req.Documents)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Bulk index documents failed",
			logger.F("index_type", indexType),
			logger.F("count", len(req.Documents)),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "bulk index documents failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "documents bulk indexed successfully",
		"count":   len(req.Documents),
	})
}

// SyncFromDatabase 从数据库同步数据
func (h *HTTPHandler) SyncFromDatabase(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	var req struct {
		SourceService string `json:"source_service" binding:"required"`
		SourceTable   string `json:"source_table" binding:"required"`
		TargetIndex   string `json:"target_index" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	err := h.indexService.SyncFromDatabase(c.Request.Context(), req.SourceService, req.SourceTable, req.TargetIndex)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Sync from database failed",
			logger.F("source_service", req.SourceService),
			logger.F("source_table", req.SourceTable),
			logger.F("target_index", req.TargetIndex),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "sync from database failed", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"message": "sync from database started successfully",
	})
}

// GetSyncStatus 获取同步状态
func (h *HTTPHandler) GetSyncStatus(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	sourceTable := c.Query("source_table")
	targetIndex := c.Query("target_index")

	if sourceTable == "" || targetIndex == "" {
		h.respondError(c, http.StatusBadRequest, "source_table and target_index are required", "")
		return
	}

	syncStatus, err := h.indexService.GetSyncStatus(c.Request.Context(), sourceTable, targetIndex)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get sync status failed",
			logger.F("source_table", sourceTable),
			logger.F("target_index", targetIndex),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get sync status failed", err.Error())
		return
	}

	h.respondSuccess(c, syncStatus)
}

// ListSyncStatuses 列出同步状态
func (h *HTTPHandler) ListSyncStatuses(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	sourceService := c.Query("source_service")

	statuses, err := h.indexService.ListSyncStatuses(c.Request.Context(), sourceService)
	if err != nil {
		h.logger.Error(c.Request.Context(), "List sync statuses failed",
			logger.F("source_service", sourceService),
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "list sync statuses failed", err.Error())
		return
	}

	response := map[string]interface{}{
		"statuses": statuses,
	}

	h.respondSuccess(c, response)
}

// ============ 健康检查接口 ============

// HealthCheck 健康检查
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	err := h.indexService.HealthCheck(c.Request.Context())
	if err != nil {
		h.logger.Error(c.Request.Context(), "Health check failed",
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusServiceUnavailable, "service unhealthy", err.Error())
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "search-service",
	}

	h.respondSuccess(c, response)
}

// GetClusterInfo 获取集群信息
func (h *HTTPHandler) GetClusterInfo(c *gin.Context) {
	if !h.isAdmin(c) {
		h.respondError(c, http.StatusForbidden, "admin access required", "")
		return
	}

	clusterInfo, err := h.indexService.GetClusterInfo(c.Request.Context())
	if err != nil {
		h.logger.Error(c.Request.Context(), "Get cluster info failed",
			logger.F("error", err.Error()))
		h.respondError(c, http.StatusInternalServerError, "get cluster info failed", err.Error())
		return
	}

	h.respondSuccess(c, clusterInfo)
}
