package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/apps/search-service/converter"
	"goim-social/apps/search-service/service"
	"goim-social/pkg/logger"
	"goim-social/pkg/middleware"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	searchService service.SearchService
	indexService  service.IndexService
	converter     *converter.Converter
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
		converter:     converter.NewConverter(),
		logger:        log,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	// 应用中间件
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimit())
	r.Use(middleware.Recovery(h.logger))

	api := r.Group("/api/v1")

	// 搜索相关路由
	search := api.Group("/search")
	{
		search.POST("/", h.Search)
		search.POST("/content", h.SearchContent)
		search.POST("/users", h.SearchUsers)
		search.POST("/messages", h.SearchMessages)
		search.POST("/groups", h.SearchGroups)
		search.POST("/multi", h.MultiSearch)
		search.POST("/suggestions", h.GetSuggestions)
		search.POST("/autocomplete", h.GetAutoComplete)
		search.POST("/hot", h.GetHotSearches)
	}

	// 搜索历史相关路由
	history := api.Group("/search/history")
	{
		history.POST("/get", h.GetUserSearchHistory)
		history.POST("/clear", h.ClearUserSearchHistory)
		history.POST("/delete", h.DeleteSearchHistory)
	}

	// 用户偏好相关路由
	preference := api.Group("/search/preference")
	{
		preference.POST("/get", h.GetUserPreference)
		preference.POST("/update", h.UpdateUserPreference)
	}

	// 搜索统计相关路由
	stats := api.Group("/search/stats")
	{
		stats.POST("/get", h.GetSearchStats)
		stats.POST("/analytics", h.GetSearchAnalytics)
	}

	// 索引管理相关路由（管理员接口）
	admin := api.Group("/admin")
	{
		index := admin.Group("/index")
		{
			index.POST("/create", h.CreateIndex)
			index.POST("/delete", h.DeleteIndex)
			index.POST("/reindex", h.ReindexAll)
			index.POST("/reindex_by_type", h.ReindexByType)
		}

		document := admin.Group("/document")
		{
			document.POST("/index", h.IndexDocument)
			document.POST("/update", h.UpdateDocument)
			document.POST("/delete", h.DeleteDocument)
			document.POST("/bulk_index", h.BulkIndexDocuments)
		}

		sync := admin.Group("/sync")
		{
			sync.POST("/start", h.SyncFromDatabase)
			sync.POST("/status", h.GetSyncStatus)
			sync.POST("/statuses", h.ListSyncStatuses)
		}
	}

	// 健康检查和集群信息
	health := api.Group("/health")
	{
		health.POST("/check", h.HealthCheck)
		health.POST("/cluster", h.GetClusterInfo)
	}
}
