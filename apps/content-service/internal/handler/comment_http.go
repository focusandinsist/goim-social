package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/content-service/internal/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// CreateComment 创建评论
func (h *HTTPHandler) CreateComment(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.CreateCommentRequest
		resp *rest.CreateCommentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create comment request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorCreateCommentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)
	ctx = tracecontext.WithContentID(ctx, req.TargetId)

	// 构建创建评论参数
	params := service.CreateCommentParams{
		TargetID:        req.TargetId,
		TargetType:      req.TargetType.String(),
		UserID:          req.UserId,
		UserName:        req.UserName,
		UserAvatar:      req.UserAvatar,
		Content:         req.Content,
		ParentID:        req.ParentId,
		ReplyToUserID:   req.ReplyToUserId,
		ReplyToUserName: req.ReplyToUserName,
		IPAddress:       req.IpAddress,
		UserAgent:       req.UserAgent,
	}

	comment, err := h.svc.CreateComment(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Create comment failed", logger.F("error", err.Error()), logger.F("targetID", req.TargetId))
		resp = h.converter.BuildErrorCreateCommentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Create comment successful", logger.F("commentID", comment.ID), logger.F("targetID", req.TargetId))
		resp = h.converter.BuildCreateCommentResponse(true, "创建评论成功", comment)
	}

	httpx.WriteObject(c, resp, err)
}

// DeleteComment 删除评论
func (h *HTTPHandler) DeleteComment(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.DeleteCommentRequest
		resp *rest.DeleteCommentResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete comment request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorDeleteCommentResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err = h.svc.DeleteComment(ctx, req.CommentId, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "Delete comment failed", logger.F("error", err.Error()), logger.F("commentID", req.CommentId))
		resp = h.converter.BuildErrorDeleteCommentResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Delete comment successful", logger.F("commentID", req.CommentId))
		resp = h.converter.BuildDeleteCommentResponse(true, "删除评论成功")
	}

	httpx.WriteObject(c, resp, err)
}

// GetComments 获取评论列表
func (h *HTTPHandler) GetComments(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetCommentsRequest
		resp *rest.GetCommentsResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comments request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetCommentsResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithContentID(ctx, req.TargetId)

	comments, total, err := h.svc.GetComments(ctx, req.TargetId, req.TargetType.String(), req.ParentId, req.SortBy, req.SortOrder, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Get comments failed", logger.F("error", err.Error()), logger.F("targetID", req.TargetId))
		resp = h.converter.BuildErrorGetCommentsResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get comments successful", logger.F("targetID", req.TargetId), logger.F("count", len(comments)))
		resp = h.converter.BuildGetCommentsResponse(true, "获取评论列表成功", comments, total)
	}

	httpx.WriteObject(c, resp, err)
}

// GetCommentReplies 获取评论回复
func (h *HTTPHandler) GetCommentReplies(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.GetCommentRepliesRequest
		resp *rest.GetCommentRepliesResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get comment replies request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorGetCommentRepliesResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	replies, total, err := h.svc.GetCommentReplies(ctx, req.CommentId, req.SortBy, req.SortOrder, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error(ctx, "Get comment replies failed", logger.F("error", err.Error()), logger.F("commentID", req.CommentId))
		resp = h.converter.BuildErrorGetCommentRepliesResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Get comment replies successful", logger.F("commentID", req.CommentId), logger.F("count", len(replies)))
		resp = h.converter.BuildGetCommentRepliesResponse(true, "获取评论回复成功", replies, total)
	}

	httpx.WriteObject(c, resp, err)
}
