package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/comment-service/converter"
	"goim-social/apps/comment-service/model"
	"goim-social/apps/comment-service/service"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, logger logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    logger,
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

		// 统计查询
		api.POST("/stats", h.GetCommentStats)
		api.POST("/batch_stats", h.GetBatchCommentStats)

	}
}

// CreateComment 创建评论
func (h *HTTPHandler) CreateComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create comment request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorCreateCommentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换对象类型
	objectType := h.converter.ObjectTypeFromProto(req.ObjectType)

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

	var res *rest.CreateCommentResponse
	if err != nil {
		h.logger.Error(ctx, "Create comment failed", logger.F("error", err.Error()))
		res = h.converter.BuildCreateCommentResponse(false, err.Error(), nil)
	} else {
		res = h.converter.BuildSuccessCreateCommentResponse(comment)
	}

	httpx.WriteObject(c, res, err)
}

// UpdateComment 更新评论
func (h *HTTPHandler) UpdateComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.UpdateCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid update comment request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorUpdateCommentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	params := &model.UpdateCommentParams{
		CommentID: req.CommentId,
		UserID:    req.UserId,
		Content:   req.Content,
	}

	comment, err := h.svc.UpdateComment(ctx, params)

	var res *rest.UpdateCommentResponse
	if err != nil {
		h.logger.Error(ctx, "Update comment failed", logger.F("error", err.Error()))
		res = h.converter.BuildUpdateCommentResponse(false, err.Error(), nil)
	} else {
		res = h.converter.BuildSuccessUpdateCommentResponse(comment)
	}

	httpx.WriteObject(c, res, err)
}

// DeleteComment 删除评论
func (h *HTTPHandler) DeleteComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete comment request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorDeleteCommentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	params := &model.DeleteCommentParams{
		CommentID: req.CommentId,
		UserID:    req.UserId,
		IsAdmin:   req.IsAdmin,
	}

	err := h.svc.DeleteComment(ctx, params)

	var res *rest.DeleteCommentResponse
	if err != nil {
		h.logger.Error(ctx, "Delete comment failed", logger.F("error", err.Error()))
		res = h.converter.BuildDeleteCommentResponse(false, err.Error())
	} else {
		res = h.converter.BuildSuccessDeleteCommentResponse()
	}

	httpx.WriteObject(c, res, err)
}

// GetComment 获取评论
func (h *HTTPHandler) GetComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comment request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetCommentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	comment, err := h.svc.GetComment(ctx, req.CommentId)

	var res *rest.GetCommentResponse
	if err != nil {
		h.logger.Error(ctx, "Get comment failed", logger.F("error", err.Error()))
		res = h.converter.BuildGetCommentResponse(false, err.Error(), nil)
	} else {
		res = h.converter.BuildSuccessGetCommentResponse(comment)
	}

	httpx.WriteObject(c, res, err)
}

// GetComments 获取评论列表
func (h *HTTPHandler) GetComments(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetCommentsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comments request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetCommentsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换对象类型
	objectType := h.converter.ObjectTypeFromProto(req.ObjectType)

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

	var res *rest.GetCommentsResponse
	if err != nil {
		h.logger.Error(ctx, "Get comments failed", logger.F("error", err.Error()))
		res = h.converter.BuildGetCommentsResponse(false, err.Error(), nil, 0, req.Page, req.PageSize)
	} else {
		res = h.converter.BuildSuccessGetCommentsResponse(comments, total, req.Page, req.PageSize)
	}

	httpx.WriteObject(c, res, err)
}

// GetUserComments 获取用户评论
func (h *HTTPHandler) GetUserComments(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserCommentsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user comments request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetUserCommentsResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换状态枚举
	status := h.converter.CommentStatusFromProto(req.Status)

	params := &model.GetUserCommentsParams{
		UserID:   req.UserId,
		Status:   status,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	comments, total, err := h.svc.GetUserComments(ctx, params)

	var res *rest.GetUserCommentsResponse
	if err != nil {
		h.logger.Error(ctx, "Get user comments failed", logger.F("error", err.Error()))
		res = h.converter.BuildGetUserCommentsResponse(false, err.Error(), nil, 0, req.Page, req.PageSize)
	} else {
		res = h.converter.BuildSuccessGetUserCommentsResponse(comments, total, req.Page, req.PageSize)
	}

	httpx.WriteObject(c, res, err)
}

// ModerateComment 审核评论
func (h *HTTPHandler) ModerateComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ModerateCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid moderate comment request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorModerateCommentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 转换状态枚举
	newStatus := h.converter.CommentStatusFromProto(req.NewStatus)

	params := &model.ModerateCommentParams{
		CommentID:   req.CommentId,
		ModeratorID: req.ModeratorId,
		NewStatus:   newStatus,
		Reason:      req.Reason,
	}

	comment, err := h.svc.ModerateComment(ctx, params)

	var res *rest.ModerateCommentResponse
	if err != nil {
		h.logger.Error(ctx, "Failed to moderate comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentId))
		res = h.converter.BuildModerateCommentResponse(false, err.Error(), nil)
	} else {
		res = h.converter.BuildModerateCommentResponse(true, "审核成功", comment)
	}

	httpx.WriteObject(c, res, err)
}

// PinComment 置顶评论
func (h *HTTPHandler) PinComment(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.PinCommentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid pin comment request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorPinCommentResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	params := &model.PinCommentParams{
		CommentID:  req.CommentId,
		OperatorID: req.OperatorId,
		IsPinned:   req.IsPinned,
	}

	err := h.svc.PinComment(ctx, params)

	var res *rest.PinCommentResponse
	if err != nil {
		h.logger.Error(ctx, "Failed to pin comment",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentId))
		res = h.converter.BuildPinCommentResponse(false, err.Error())
	} else {
		message := "置顶成功"
		if !req.IsPinned {
			message = "取消置顶成功"
		}
		res = h.converter.BuildPinCommentResponse(true, message)
	}

	httpx.WriteObject(c, res, err)
}
