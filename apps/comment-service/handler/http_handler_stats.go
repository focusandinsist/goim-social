package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"websocket-server/pkg/logger"
)

// IsCommentLiked 检查是否已点赞
func (h *HTTPHandler) IsCommentLiked(c *gin.Context) {
	var req LikeCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	isLiked, err := h.svc.IsCommentLiked(c.Request.Context(), req.CommentID, req.UserID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to check comment like status",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "查询成功",
		"is_liked": isLiked,
	})
}

// GetCommentStatsRequest 获取评论统计请求
type GetCommentStatsRequest struct {
	ObjectID   int64  `json:"object_id" binding:"required"`
	ObjectType string `json:"object_type" binding:"required"`
}

// GetCommentStats 获取评论统计
func (h *HTTPHandler) GetCommentStats(c *gin.Context) {
	var req GetCommentStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	stats, err := h.svc.GetCommentStats(c.Request.Context(), req.ObjectID, req.ObjectType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get comment stats",
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
		"data":    stats,
	})
}

// GetBatchCommentStatsRequest 批量获取评论统计请求
type GetBatchCommentStatsRequest struct {
	ObjectIDs  []int64 `json:"object_ids" binding:"required"`
	ObjectType string  `json:"object_type" binding:"required"`
}

// GetBatchCommentStats 批量获取评论统计
func (h *HTTPHandler) GetBatchCommentStats(c *gin.Context) {
	var req GetBatchCommentStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	statsList, err := h.svc.GetBatchCommentStats(c.Request.Context(), req.ObjectIDs, req.ObjectType)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get batch comment stats",
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
		"data":    statsList,
	})
}

// GetPendingCommentsRequest 获取待审核评论请求
type GetPendingCommentsRequest struct {
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
}

// GetPendingComments 获取待审核评论
func (h *HTTPHandler) GetPendingComments(c *gin.Context) {
	var req GetPendingCommentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	comments, total, err := h.svc.GetPendingComments(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get pending comments",
			logger.F("error", err.Error()))
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

// GetCommentsByStatusRequest 根据状态获取评论请求
type GetCommentsByStatusRequest struct {
	Status   string `json:"status" binding:"required"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
}

// GetCommentsByStatus 根据状态获取评论
func (h *HTTPHandler) GetCommentsByStatus(c *gin.Context) {
	var req GetCommentsByStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	comments, total, err := h.svc.GetCommentsByStatus(c.Request.Context(), req.Status, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get comments by status",
			logger.F("error", err.Error()),
			logger.F("status", req.Status))
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
