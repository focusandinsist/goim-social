package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"websocket-server/apps/comment-service/model"
	"websocket-server/apps/comment-service/service"
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
	api := engine.Group("/api/v1/comment")
	{
		// 基础评论操作
		api.POST("/create", h.CreateComment)
		api.POST("/update", h.UpdateComment)
		api.POST("/delete", h.DeleteComment)
		api.POST("/get", h.GetComment)

		// 评论列表查询
		api.POST("/list", h.GetComments)
		api.POST("/user_comments", h.GetUserComments)

		// 评论管理
		api.POST("/moderate", h.ModerateComment)
		api.POST("/pin", h.PinComment)

		// 点赞操作
		api.POST("/like", h.LikeComment)
		api.POST("/unlike", h.UnlikeComment)
		api.POST("/is_liked", h.IsCommentLiked)

		// 统计查询
		api.POST("/stats", h.GetCommentStats)
		api.POST("/batch_stats", h.GetBatchCommentStats)

		// 管理员操作
		api.POST("/pending", h.GetPendingComments)
		api.POST("/by_status", h.GetCommentsByStatus)
	}
}

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	ObjectID        int64  `json:"object_id" binding:"required"`
	ObjectType      string `json:"object_type" binding:"required"`
	UserID          int64  `json:"user_id" binding:"required"`
	UserName        string `json:"user_name" binding:"required"`
	UserAvatar      string `json:"user_avatar"`
	Content         string `json:"content" binding:"required"`
	ParentID        int64  `json:"parent_id"`
	ReplyToUserID   int64  `json:"reply_to_user_id"`
	ReplyToUserName string `json:"reply_to_user_name"`
}

// CreateComment 创建评论
func (h *HTTPHandler) CreateComment(c *gin.Context) {
	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.CreateCommentParams{
		ObjectID:        req.ObjectID,
		ObjectType:      req.ObjectType,
		UserID:          req.UserID,
		UserName:        req.UserName,
		UserAvatar:      req.UserAvatar,
		Content:         req.Content,
		ParentID:        req.ParentID,
		ReplyToUserID:   req.ReplyToUserID,
		ReplyToUserName: req.ReplyToUserName,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.GetHeader("User-Agent"),
	}

	comment, err := h.svc.CreateComment(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to create comment",
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
		"message": "评论创建成功",
		"data":    comment,
	})
}

// UpdateCommentRequest 更新评论请求
type UpdateCommentRequest struct {
	CommentID int64  `json:"comment_id" binding:"required"`
	UserID    int64  `json:"user_id" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

// UpdateComment 更新评论
func (h *HTTPHandler) UpdateComment(c *gin.Context) {
	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.UpdateCommentParams{
		CommentID: req.CommentID,
		UserID:    req.UserID,
		Content:   req.Content,
	}

	comment, err := h.svc.UpdateComment(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to update comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "评论更新成功",
		"data":    comment,
	})
}

// DeleteCommentRequest 删除评论请求
type DeleteCommentRequest struct {
	CommentID int64 `json:"comment_id" binding:"required"`
	UserID    int64 `json:"user_id" binding:"required"`
	IsAdmin   bool  `json:"is_admin"`
}

// DeleteComment 删除评论
func (h *HTTPHandler) DeleteComment(c *gin.Context) {
	var req DeleteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.DeleteCommentParams{
		CommentID: req.CommentID,
		UserID:    req.UserID,
		IsAdmin:   req.IsAdmin,
	}

	err := h.svc.DeleteComment(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to delete comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "评论删除成功",
	})
}

// GetCommentRequest 获取评论请求
type GetCommentRequest struct {
	CommentID int64 `json:"comment_id" binding:"required"`
}

// GetComment 获取评论
func (h *HTTPHandler) GetComment(c *gin.Context) {
	var req GetCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	comment, err := h.svc.GetComment(c.Request.Context(), req.CommentID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取成功",
		"data":    comment,
	})
}

// GetCommentsRequest 获取评论列表请求
type GetCommentsRequest struct {
	ObjectID       int64  `json:"object_id" binding:"required"`
	ObjectType     string `json:"object_type" binding:"required"`
	ParentID       int64  `json:"parent_id"`
	SortBy         string `json:"sort_by"`
	SortOrder      string `json:"sort_order"`
	Page           int32  `json:"page"`
	PageSize       int32  `json:"page_size"`
	IncludeReplies bool   `json:"include_replies"`
	MaxReplyCount  int32  `json:"max_reply_count"`
}

// GetComments 获取评论列表
func (h *HTTPHandler) GetComments(c *gin.Context) {
	var req GetCommentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.GetCommentsParams{
		ObjectID:       req.ObjectID,
		ObjectType:     req.ObjectType,
		ParentID:       req.ParentID,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
		Page:           req.Page,
		PageSize:       req.PageSize,
		IncludeReplies: req.IncludeReplies,
		MaxReplyCount:  req.MaxReplyCount,
	}

	comments, total, err := h.svc.GetComments(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get comments",
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
			"comments":  comments,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// GetUserCommentsRequest 获取用户评论请求
type GetUserCommentsRequest struct {
	UserID   int64  `json:"user_id" binding:"required"`
	Status   string `json:"status"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
}

// GetUserComments 获取用户评论
func (h *HTTPHandler) GetUserComments(c *gin.Context) {
	var req GetUserCommentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.GetUserCommentsParams{
		UserID:   req.UserID,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	comments, total, err := h.svc.GetUserComments(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get user comments",
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
			"comments":  comments,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// ModerateCommentRequest 审核评论请求
type ModerateCommentRequest struct {
	CommentID   int64  `json:"comment_id" binding:"required"`
	ModeratorID int64  `json:"moderator_id" binding:"required"`
	NewStatus   string `json:"new_status" binding:"required"`
	Reason      string `json:"reason"`
}

// ModerateComment 审核评论
func (h *HTTPHandler) ModerateComment(c *gin.Context) {
	var req ModerateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.ModerateCommentParams{
		CommentID:   req.CommentID,
		ModeratorID: req.ModeratorID,
		NewStatus:   req.NewStatus,
		Reason:      req.Reason,
	}

	comment, err := h.svc.ModerateComment(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to moderate comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "审核成功",
		"data":    comment,
	})
}

// PinCommentRequest 置顶评论请求
type PinCommentRequest struct {
	CommentID  int64 `json:"comment_id" binding:"required"`
	OperatorID int64 `json:"operator_id" binding:"required"`
	IsPinned   bool  `json:"is_pinned"`
}

// PinComment 置顶评论
func (h *HTTPHandler) PinComment(c *gin.Context) {
	var req PinCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	params := &model.PinCommentParams{
		CommentID:  req.CommentID,
		OperatorID: req.OperatorID,
		IsPinned:   req.IsPinned,
	}

	err := h.svc.PinComment(c.Request.Context(), params)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to pin comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	message := "置顶成功"
	if !req.IsPinned {
		message = "取消置顶成功"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// LikeCommentRequest 点赞评论请求
type LikeCommentRequest struct {
	CommentID int64 `json:"comment_id" binding:"required"`
	UserID    int64 `json:"user_id" binding:"required"`
}

// LikeComment 点赞评论
func (h *HTTPHandler) LikeComment(c *gin.Context) {
	var req LikeCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	err := h.svc.LikeComment(c.Request.Context(), req.CommentID, req.UserID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to like comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "点赞成功",
	})
}

// UnlikeComment 取消点赞评论
func (h *HTTPHandler) UnlikeComment(c *gin.Context) {
	var req LikeCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	err := h.svc.UnlikeComment(c.Request.Context(), req.CommentID, req.UserID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to unlike comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "取消点赞成功",
	})
}
