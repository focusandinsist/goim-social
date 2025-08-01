package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/comment-service/model"
	"goim-social/apps/comment-service/service"
	"goim-social/pkg/logger"
	"goim-social/pkg/utils"
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

// CreateComment 创建评论
func (h *HTTPHandler) CreateComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create comment request", logger.F("error", err.Error()))
		res := &rest.CreateCommentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换对象类型
	objectType := convertCommentObjectTypeFromProto(req.ObjectType)

	params := &model.CreateCommentParams{
		ObjectID:        req.ObjectId,
		ObjectType:      objectType,
		UserID:          req.UserId,
		UserName:        req.UserName,
		UserAvatar:      req.UserAvatar,
		Content:         req.Content,
		ParentID:        req.ParentId,
		ReplyToUserID:   req.ReplyToUserId,
		ReplyToUserName: req.ReplyToUserName,
		IPAddress:       c.ClientIP(),
		UserAgent:       c.GetHeader("User-Agent"),
	}

	comment, err := h.svc.CreateComment(ctx, params)
	res := &rest.CreateCommentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "创建成功"
		}(),
		Comment: func() *rest.Comment {
			if err != nil {
				return nil
			}
			return convertCommentToProto(comment)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Create comment failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "评论创建成功",
		"data":    comment,
	})
}

// UpdateComment 更新评论
func (h *HTTPHandler) UpdateComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.UpdateCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid update comment request", logger.F("error", err.Error()))
		res := &rest.UpdateCommentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	params := &model.UpdateCommentParams{
		CommentID: req.CommentId,
		UserID:    req.UserId,
		Content:   req.Content,
	}

	comment, err := h.svc.UpdateComment(ctx, params)
	res := &rest.UpdateCommentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "评论更新成功"
		}(),
		Comment: func() *rest.Comment {
			if err != nil {
				return nil
			}
			return convertCommentToProto(comment)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Update comment failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// DeleteComment 删除评论
func (h *HTTPHandler) DeleteComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete comment request", logger.F("error", err.Error()))
		res := &rest.DeleteCommentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	params := &model.DeleteCommentParams{
		CommentID: req.CommentId,
		UserID:    req.UserId,
		IsAdmin:   req.IsAdmin,
	}

	err := h.svc.DeleteComment(ctx, params)
	res := &rest.DeleteCommentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "评论删除成功"
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Delete comment failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetComment 获取评论
func (h *HTTPHandler) GetComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comment request", logger.F("error", err.Error()))
		res := &rest.GetCommentResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	comment, err := h.svc.GetComment(ctx, req.CommentId)
	res := &rest.GetCommentResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Comment: func() *rest.Comment {
			if err != nil {
				return nil
			}
			return convertCommentToProto(comment)
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Get comment failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetComments 获取评论列表
func (h *HTTPHandler) GetComments(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetCommentsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comments request", logger.F("error", err.Error()))
		res := &rest.GetCommentsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换对象类型
	objectType := convertCommentObjectTypeFromProto(req.ObjectType)

	params := &model.GetCommentsParams{
		ObjectID:       req.ObjectId,
		ObjectType:     objectType,
		ParentID:       req.ParentId,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
		Page:           req.Page,
		PageSize:       req.PageSize,
		IncludeReplies: req.IncludeReplies,
		MaxReplyCount:  req.MaxReplyCount,
	}

	comments, total, err := h.svc.GetComments(ctx, params)

	// 转换评论列表
	var protoComments []*rest.Comment
	if err == nil {
		for _, comment := range comments {
			protoComments = append(protoComments, convertCommentToProto(comment))
		}
	}

	res := &rest.GetCommentsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Comments: protoComments,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if err != nil {
		h.logger.Error(ctx, "Get comments failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetUserComments 获取用户评论
func (h *HTTPHandler) GetUserComments(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserCommentsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user comments request", logger.F("error", err.Error()))
		res := &rest.GetUserCommentsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 转换状态枚举
	status := convertCommentStatusFromProto(req.Status)

	params := &model.GetUserCommentsParams{
		UserID:   req.UserId,
		Status:   status,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	comments, total, err := h.svc.GetUserComments(ctx, params)

	// 转换评论列表
	var protoComments []*rest.Comment
	if err == nil {
		for _, comment := range comments {
			protoComments = append(protoComments, convertCommentToProto(comment))
		}
	}

	res := &rest.GetUserCommentsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取成功"
		}(),
		Comments: protoComments,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if err != nil {
		h.logger.Error(ctx, "Get user comments failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
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

// convertCommentObjectTypeFromProto 将protobuf枚举转换为对象类型
func convertCommentObjectTypeFromProto(objectType rest.CommentObjectType) string {
	switch objectType {
	case rest.CommentObjectType_COMMENT_OBJECT_TYPE_POST:
		return "post"
	case rest.CommentObjectType_COMMENT_OBJECT_TYPE_ARTICLE:
		return "article"
	case rest.CommentObjectType_COMMENT_OBJECT_TYPE_VIDEO:
		return "video"
	case rest.CommentObjectType_COMMENT_OBJECT_TYPE_PRODUCT:
		return "product"
	default:
		return "post"
	}
}
