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

	// 健康检查和集群信息
	health := api.Group("/health")
	{
		health.GET("/", h.HealthCheck)
		health.GET("/cluster", h.GetClusterInfo)
	}
}
