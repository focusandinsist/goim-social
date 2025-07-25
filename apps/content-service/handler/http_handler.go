package handler

import (
	"net/http"

	"websocket-server/apps/content-service/model"
	"websocket-server/apps/content-service/service"
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
	api := r.Group("/api/v1/content")
	{
		// 内容管理
		api.POST("/create", h.CreateContent)              // 创建内容
		api.POST("/update", h.UpdateContent)              // 更新内容
		api.POST("/get", h.GetContent)                    // 获取内容详情
		api.POST("/delete", h.DeleteContent)              // 删除内容
		api.POST("/publish", h.PublishContent)            // 发布内容
		api.POST("/change_status", h.ChangeContentStatus) // 变更内容状态

		// 内容查询
		api.POST("/search", h.SearchContent)        // 搜索内容
		api.POST("/user_content", h.GetUserContent) // 获取用户内容列表
		api.POST("/stats", h.GetContentStats)       // 获取内容统计

		// 标签管理
		api.POST("/tag/create", h.CreateTag) // 创建标签
		api.POST("/tag/list", h.GetTags)     // 获取标签列表

		// 话题管理
		api.POST("/topic/create", h.CreateTopic) // 创建话题
		api.POST("/topic/list", h.GetTopics)     // 获取话题列表
	}
}

// CreateContentRequest 创建内容请求
type CreateContentRequest struct {
	AuthorID     int64                    `json:"author_id" binding:"required"`
	Title        string                   `json:"title" binding:"required"`
	Content      string                   `json:"content"`
	Type         string                   `json:"type" binding:"required"`
	MediaFiles   []model.ContentMediaFile `json:"media_files"`
	TagIDs       []int64                  `json:"tag_ids"`
	TopicIDs     []int64                  `json:"topic_ids"`
	TemplateData string                   `json:"template_data"`
	SaveAsDraft  bool                     `json:"save_as_draft"`
}

// CreateContent 创建内容
func (h *HTTPHandler) CreateContent(c *gin.Context) {
	var req CreateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	content, err := h.svc.CreateContent(
		c.Request.Context(),
		req.AuthorID,
		req.Title,
		req.Content,
		req.Type,
		req.MediaFiles,
		req.TagIDs,
		req.TopicIDs,
		req.TemplateData,
		req.SaveAsDraft,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to create content",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "创建成功",
		"data":    content,
	})
}

// UpdateContentRequest 更新内容请求
type UpdateContentRequest struct {
	ContentID    int64                    `json:"content_id" binding:"required"`
	AuthorID     int64                    `json:"author_id" binding:"required"`
	Title        string                   `json:"title" binding:"required"`
	Content      string                   `json:"content"`
	Type         string                   `json:"type" binding:"required"`
	MediaFiles   []model.ContentMediaFile `json:"media_files"`
	TagIDs       []int64                  `json:"tag_ids"`
	TopicIDs     []int64                  `json:"topic_ids"`
	TemplateData string                   `json:"template_data"`
}

// UpdateContent 更新内容
func (h *HTTPHandler) UpdateContent(c *gin.Context) {
	var req UpdateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	content, err := h.svc.UpdateContent(
		c.Request.Context(),
		req.ContentID,
		req.AuthorID,
		req.Title,
		req.Content,
		req.Type,
		req.MediaFiles,
		req.TagIDs,
		req.TopicIDs,
		req.TemplateData,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to update content",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    content,
	})
}

// GetContentRequest 获取内容请求
type GetContentRequest struct {
	ContentID int64 `json:"content_id" binding:"required"`
	UserID    int64 `json:"user_id"`
}

// GetContent 获取内容详情
func (h *HTTPHandler) GetContent(c *gin.Context) {
	var req GetContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	content, err := h.svc.GetContent(c.Request.Context(), req.ContentID, req.UserID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get content",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data":    content,
	})
}

// DeleteContentRequest 删除内容请求
type DeleteContentRequest struct {
	ContentID int64 `json:"content_id" binding:"required"`
	AuthorID  int64 `json:"author_id" binding:"required"`
}

// DeleteContent 删除内容
func (h *HTTPHandler) DeleteContent(c *gin.Context) {
	var req DeleteContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	err := h.svc.DeleteContent(c.Request.Context(), req.ContentID, req.AuthorID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to delete content",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// PublishContentRequest 发布内容请求
type PublishContentRequest struct {
	ContentID int64 `json:"content_id" binding:"required"`
	AuthorID  int64 `json:"author_id" binding:"required"`
}

// PublishContent 发布内容
func (h *HTTPHandler) PublishContent(c *gin.Context) {
	var req PublishContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	content, err := h.svc.PublishContent(c.Request.Context(), req.ContentID, req.AuthorID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to publish content",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "发布成功",
		"data":    content,
	})
}

// ChangeContentStatusRequest 变更内容状态请求
type ChangeContentStatusRequest struct {
	ContentID  int64  `json:"content_id" binding:"required"`
	OperatorID int64  `json:"operator_id" binding:"required"`
	NewStatus  string `json:"new_status" binding:"required"`
	Reason     string `json:"reason"`
}

// ChangeContentStatus 变更内容状态
func (h *HTTPHandler) ChangeContentStatus(c *gin.Context) {
	var req ChangeContentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	content, err := h.svc.ChangeContentStatus(c.Request.Context(), req.ContentID, req.OperatorID, req.NewStatus, req.Reason)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to change content status",
			logger.F("error", err.Error()),
			logger.F("contentID", req.ContentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "状态变更成功",
		"data":    content,
	})
}

// SearchContentRequest 搜索内容请求
type SearchContentRequest struct {
	Keyword   string  `json:"keyword"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	TagIDs    []int64 `json:"tag_ids"`
	TopicIDs  []int64 `json:"topic_ids"`
	AuthorID  int64   `json:"author_id"`
	Page      int32   `json:"page"`
	PageSize  int32   `json:"page_size"`
	SortBy    string  `json:"sort_by"`
	SortOrder string  `json:"sort_order"`
}

// SearchContent 搜索内容
func (h *HTTPHandler) SearchContent(c *gin.Context) {
	var req SearchContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.SearchContentParams{
		Keyword:   req.Keyword,
		Type:      req.Type,
		Status:    req.Status,
		TagIDs:    req.TagIDs,
		TopicIDs:  req.TopicIDs,
		AuthorID:  req.AuthorID,
		Page:      req.Page,
		PageSize:  req.PageSize,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	contents, total, err := h.svc.SearchContent(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to search content",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "搜索成功",
		"data": gin.H{
			"contents":  contents,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// GetUserContentRequest 获取用户内容请求
type GetUserContentRequest struct {
	AuthorID int64  `json:"author_id" binding:"required"`
	Status   string `json:"status"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
}

// GetUserContent 获取用户内容列表
func (h *HTTPHandler) GetUserContent(c *gin.Context) {
	var req GetUserContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	contents, total, err := h.svc.GetUserContent(c.Request.Context(), req.AuthorID, req.Status, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get user content",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorID))
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
			"contents":  contents,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// GetContentStatsRequest 获取内容统计请求
type GetContentStatsRequest struct {
	AuthorID int64 `json:"author_id"`
}

// GetContentStats 获取内容统计
func (h *HTTPHandler) GetContentStats(c *gin.Context) {
	var req GetContentStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	stats, err := h.svc.GetContentStats(c.Request.Context(), req.AuthorID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get content stats",
			logger.F("error", err.Error()),
			logger.F("authorID", req.AuthorID))
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

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateTag 创建标签
func (h *HTTPHandler) CreateTag(c *gin.Context) {
	var req CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	tag, err := h.svc.CreateTag(c.Request.Context(), req.Name)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to create tag",
			logger.F("error", err.Error()),
			logger.F("name", req.Name))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "创建成功",
		"data":    tag,
	})
}

// GetTagsRequest 获取标签列表请求
type GetTagsRequest struct {
	Keyword  string `json:"keyword"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
}

// GetTags 获取标签列表
func (h *HTTPHandler) GetTags(c *gin.Context) {
	var req GetTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	tags, total, err := h.svc.GetTags(c.Request.Context(), req.Keyword, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get tags",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
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
			"tags":      tags,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// CreateTopicRequest 创建话题请求
type CreateTopicRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	CoverImage  string `json:"cover_image"`
}

// CreateTopic 创建话题
func (h *HTTPHandler) CreateTopic(c *gin.Context) {
	var req CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	topic, err := h.svc.CreateTopic(c.Request.Context(), req.Name, req.Description, req.CoverImage)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to create topic",
			logger.F("error", err.Error()),
			logger.F("name", req.Name))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "创建成功",
		"data":    topic,
	})
}

// GetTopicsRequest 获取话题列表请求
type GetTopicsRequest struct {
	Keyword  string `json:"keyword"`
	HotOnly  bool   `json:"hot_only"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
}

// GetTopics 获取话题列表
func (h *HTTPHandler) GetTopics(c *gin.Context) {
	var req GetTopicsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	topics, total, err := h.svc.GetTopics(c.Request.Context(), req.Keyword, req.HotOnly, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get topics",
			logger.F("error", err.Error()),
			logger.F("keyword", req.Keyword))
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
			"topics":    topics,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}
