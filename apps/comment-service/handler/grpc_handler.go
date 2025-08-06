package handler

import (
	"context"

	"goim-social/api/rest"
	"goim-social/apps/comment-service/converter"
	"goim-social/apps/comment-service/model"
	"goim-social/apps/comment-service/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedCommentServiceServer
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		logger:    log,
	}
}

// CreateComment 创建评论
func (h *GRPCHandler) CreateComment(ctx context.Context, req *rest.CreateCommentRequest) (*rest.CreateCommentResponse, error) {
	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	params := &model.CreateCommentParams{
		ObjectID:        req.ObjectId,
		ObjectType:      h.converter.ObjectTypeFromProto(req.ObjectType),
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
		h.logger.Error(ctx, "Failed to create comment via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("objectID", req.ObjectId))
		return h.converter.BuildErrorCreateCommentResponse(err.Error()), nil
	}

	h.logger.Info(ctx, "Create comment via gRPC successful",
		logger.F("commentID", comment.ID),
		logger.F("userID", req.UserId),
		logger.F("objectID", req.ObjectId))

	return h.converter.BuildSuccessCreateCommentResponse(comment), nil
}

// UpdateComment 更新评论
func (h *GRPCHandler) UpdateComment(ctx context.Context, req *rest.UpdateCommentRequest) (*rest.UpdateCommentResponse, error) {
	params := &model.UpdateCommentParams{
		CommentID: req.CommentId,
		UserID:    req.UserId,
		Content:   req.Content,
	}

	comment, err := h.svc.UpdateComment(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to update comment via gRPC",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentId))
		return &rest.UpdateCommentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return h.converter.BuildSuccessUpdateCommentResponse(comment), nil
}

// DeleteComment 删除评论
func (h *GRPCHandler) DeleteComment(ctx context.Context, req *rest.DeleteCommentRequest) (*rest.DeleteCommentResponse, error) {
	params := &model.DeleteCommentParams{
		CommentID: req.CommentId,
		UserID:    req.UserId,
		IsAdmin:   req.IsAdmin,
	}

	err := h.svc.DeleteComment(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete comment via gRPC",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentId))
		return &rest.DeleteCommentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return h.converter.BuildSuccessDeleteCommentResponse(), nil
}

// GetComment 获取评论
func (h *GRPCHandler) GetComment(ctx context.Context, req *rest.GetCommentRequest) (*rest.GetCommentResponse, error) {
	h.logger.Info(ctx, "gRPC GetComment request", logger.F("commentID", req.CommentId))

	comment, err := h.svc.GetComment(ctx, req.CommentId)
	if err != nil {
		h.logger.Error(ctx, "Failed to get comment via gRPC",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentId))
		return &rest.GetCommentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	h.logger.Info(ctx, "gRPC GetComment successful",
		logger.F("commentID", req.CommentId),
		logger.F("userID", comment.UserID))

	return h.converter.BuildSuccessGetCommentResponse(comment), nil
}

// GetComments 获取评论列表
func (h *GRPCHandler) GetComments(ctx context.Context, req *rest.GetCommentsRequest) (*rest.GetCommentsResponse, error) {
	params := &model.GetCommentsParams{
		ObjectID:       req.ObjectId,
		ObjectType:     h.converter.ObjectTypeFromProto(req.ObjectType),
		ParentID:       req.ParentId,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
		Page:           req.Page,
		PageSize:       req.PageSize,
		IncludeReplies: req.IncludeReplies,
		MaxReplyCount:  req.MaxReplyCount,
	}

	comments, total, err := h.svc.GetComments(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get comments via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectId))
		return h.converter.BuildGetCommentsResponse(false, err.Error(), nil, 0, req.Page, req.PageSize), nil
	}

	return h.converter.BuildSuccessGetCommentsResponse(comments, total, req.Page, req.PageSize), nil
}

// GetUserComments 获取用户评论
func (h *GRPCHandler) GetUserComments(ctx context.Context, req *rest.GetUserCommentsRequest) (*rest.GetUserCommentsResponse, error) {
	params := &model.GetUserCommentsParams{
		UserID:   req.UserId,
		Status:   h.converter.CommentStatusFromProto(req.Status),
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	comments, total, err := h.svc.GetUserComments(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user comments via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return h.converter.BuildGetUserCommentsResponse(false, err.Error(), nil, 0, req.Page, req.PageSize), nil
	}

	return h.converter.BuildSuccessGetUserCommentsResponse(comments, total, req.Page, req.PageSize), nil
}

// ModerateComment 审核评论
func (h *GRPCHandler) ModerateComment(ctx context.Context, req *rest.ModerateCommentRequest) (*rest.ModerateCommentResponse, error) {
	params := &model.ModerateCommentParams{
		CommentID:   req.CommentId,
		ModeratorID: req.ModeratorId,
		NewStatus:   h.converter.CommentStatusFromProto(req.NewStatus),
		Reason:      req.Reason,
	}

	comment, err := h.svc.ModerateComment(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to moderate comment via gRPC",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentId))
		return h.converter.BuildModerateCommentResponse(false, err.Error(), nil), nil
	}

	return h.converter.BuildModerateCommentResponse(true, "审核成功", comment), nil
}

// PinComment 置顶评论
func (h *GRPCHandler) PinComment(ctx context.Context, req *rest.PinCommentRequest) (*rest.PinCommentResponse, error) {
	params := &model.PinCommentParams{
		CommentID:  req.CommentId,
		OperatorID: req.OperatorId,
		IsPinned:   req.IsPinned,
	}

	err := h.svc.PinComment(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to pin comment via gRPC",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentId))
		return &rest.PinCommentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	message := "置顶成功"
	if !req.IsPinned {
		message = "取消置顶成功"
	}

	return h.converter.BuildPinCommentResponse(true, message), nil
}

// GetCommentStats 获取评论统计
func (h *GRPCHandler) GetCommentStats(ctx context.Context, req *rest.GetCommentStatsRequest) (*rest.GetCommentStatsResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.ObjectType)
	stats, err := h.svc.GetCommentStats(ctx, req.ObjectId, objectType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get comment stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectId))
		return h.converter.BuildGetCommentStatsResponse(false, err.Error(), nil), nil
	}

	return h.converter.BuildGetCommentStatsResponse(true, "获取成功", stats), nil
}

// GetBatchCommentStats 批量获取评论统计
func (h *GRPCHandler) GetBatchCommentStats(ctx context.Context, req *rest.GetBatchCommentStatsRequest) (*rest.GetBatchCommentStatsResponse, error) {
	objectType := h.converter.ObjectTypeFromProto(req.ObjectType)
	statsList, err := h.svc.GetBatchCommentStats(ctx, req.ObjectIds, objectType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get batch comment stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectIDs", req.ObjectIds))
		return h.converter.BuildGetBatchCommentStatsResponse(false, err.Error(), nil), nil
	}

	return h.converter.BuildGetBatchCommentStatsResponse(true, "获取成功", statsList), nil
}
