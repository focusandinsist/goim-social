package handler

import (
	"goim-social/apps/content-service/internal/converter"
	"goim-social/apps/content-service/internal/service"
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
		api.POST("/user_content", h.GetUserContent) // 获取用户内容列表
		api.POST("/stats", h.GetContentStats)       // 获取内容统计

		// 标签管理
		api.POST("/tag/create", h.CreateTag) // 创建标签
		api.POST("/tag/list", h.GetTags)     // 获取标签列表

		// 话题管理
		api.POST("/topic/create", h.CreateTopic) // 创建话题
		api.POST("/topic/list", h.GetTopics)     // 获取话题列表

		// 评论管理
		api.POST("/comment/create", h.CreateComment)      // 创建评论
		api.POST("/comment/delete", h.DeleteComment)      // 删除评论
		api.POST("/comment/list", h.GetComments)          // 获取评论列表
		api.POST("/comment/replies", h.GetCommentReplies) // 获取评论回复

		// 互动管理
		api.POST("/interaction/do", h.DoInteraction)          // 执行互动（点赞/收藏/分享等）
		api.POST("/interaction/undo", h.UndoInteraction)      // 取消互动
		api.POST("/interaction/check", h.CheckInteraction)    // 检查互动状态
		api.POST("/interaction/stats", h.GetInteractionStats) // 获取互动统计

		// 聚合查询
		api.POST("/detail", h.GetContentDetail)     // 获取内容详情（包含评论和互动）
		api.POST("/feed", h.GetContentFeed)         // 获取内容流
		api.POST("/trending", h.GetTrendingContent) // 获取热门内容
	}
}
