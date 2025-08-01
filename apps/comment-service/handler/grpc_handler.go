package handler

import (
	"context"
	"time"

	"goim-social/api/rest"
	"goim-social/apps/comment-service/model"
	"goim-social/apps/comment-service/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedCommentServiceServer
	svc    *service.Service
	logger logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(svc *service.Service, log logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		svc:    svc,
		logger: log,
	}
}

// CreateComment 创建评论
func (h *GRPCHandler) CreateComment(ctx context.Context, req *rest.CreateCommentRequest) (*rest.CreateCommentResponse, error) {
	params := &model.CreateCommentParams{
		ObjectID:        req.ObjectId,
		ObjectType:      convertObjectTypeFromProto(req.ObjectType),
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
			logger.F("userID", req.UserId))
		return &rest.CreateCommentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.CreateCommentResponse{
		Success: true,
		Message: "评论创建成功",
		Comment: convertCommentToProto(comment),
	}, nil
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

	return &rest.UpdateCommentResponse{
		Success: true,
		Message: "评论更新成功",
		Comment: convertCommentToProto(comment),
	}, nil
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

	return &rest.DeleteCommentResponse{
		Success: true,
		Message: "评论删除成功",
	}, nil
}

// GetComment 获取评论
func (h *GRPCHandler) GetComment(ctx context.Context, req *rest.GetCommentRequest) (*rest.GetCommentResponse, error) {
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

	return &rest.GetCommentResponse{
		Success: true,
		Message: "获取成功",
		Comment: convertCommentToProto(comment),
	}, nil
}

// GetComments 获取评论列表
func (h *GRPCHandler) GetComments(ctx context.Context, req *rest.GetCommentsRequest) (*rest.GetCommentsResponse, error) {
	params := &model.GetCommentsParams{
		ObjectID:       req.ObjectId,
		ObjectType:     convertObjectTypeFromProto(req.ObjectType),
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
		return &rest.GetCommentsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoComments []*rest.Comment
	for _, comment := range comments {
		protoComments = append(protoComments, convertCommentToProto(comment))
	}

	return &rest.GetCommentsResponse{
		Success:  true,
		Message:  "获取成功",
		Comments: protoComments,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetUserComments 获取用户评论
func (h *GRPCHandler) GetUserComments(ctx context.Context, req *rest.GetUserCommentsRequest) (*rest.GetUserCommentsResponse, error) {
	params := &model.GetUserCommentsParams{
		UserID:   req.UserId,
		Status:   convertCommentStatusFromProto(req.Status),
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	comments, total, err := h.svc.GetUserComments(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user comments via gRPC",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		return &rest.GetUserCommentsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoComments []*rest.Comment
	for _, comment := range comments {
		protoComments = append(protoComments, convertCommentToProto(comment))
	}

	return &rest.GetUserCommentsResponse{
		Success:  true,
		Message:  "获取成功",
		Comments: protoComments,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ModerateComment 审核评论
func (h *GRPCHandler) ModerateComment(ctx context.Context, req *rest.ModerateCommentRequest) (*rest.ModerateCommentResponse, error) {
	params := &model.ModerateCommentParams{
		CommentID:   req.CommentId,
		ModeratorID: req.ModeratorId,
		NewStatus:   convertCommentStatusFromProto(req.NewStatus),
		Reason:      req.Reason,
	}

	comment, err := h.svc.ModerateComment(ctx, params)
	if err != nil {
		h.logger.Error(ctx, "Failed to moderate comment via gRPC",
			logger.F("error", err.Error()),
			logger.F("commentID", req.CommentId))
		return &rest.ModerateCommentResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.ModerateCommentResponse{
		Success: true,
		Message: "审核成功",
		Comment: convertCommentToProto(comment),
	}, nil
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

	return &rest.PinCommentResponse{
		Success: true,
		Message: message,
	}, nil
}

// GetCommentStats 获取评论统计
func (h *GRPCHandler) GetCommentStats(ctx context.Context, req *rest.GetCommentStatsRequest) (*rest.GetCommentStatsResponse, error) {
	stats, err := h.svc.GetCommentStats(ctx, req.ObjectId, convertObjectTypeFromProto(req.ObjectType))
	if err != nil {
		h.logger.Error(ctx, "Failed to get comment stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectID", req.ObjectId))
		return &rest.GetCommentStatsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &rest.GetCommentStatsResponse{
		Success: true,
		Message: "获取成功",
		Stats:   convertCommentStatsToProto(stats),
	}, nil
}

// GetBatchCommentStats 批量获取评论统计
func (h *GRPCHandler) GetBatchCommentStats(ctx context.Context, req *rest.GetBatchCommentStatsRequest) (*rest.GetBatchCommentStatsResponse, error) {
	statsList, err := h.svc.GetBatchCommentStats(ctx, req.ObjectIds, convertObjectTypeFromProto(req.ObjectType))
	if err != nil {
		h.logger.Error(ctx, "Failed to get batch comment stats via gRPC",
			logger.F("error", err.Error()),
			logger.F("objectIDs", req.ObjectIds))
		return &rest.GetBatchCommentStatsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var protoStats []*rest.CommentStats
	for _, stats := range statsList {
		protoStats = append(protoStats, convertCommentStatsToProto(stats))
	}

	return &rest.GetBatchCommentStatsResponse{
		Success: true,
		Message: "获取成功",
		Stats:   protoStats,
	}, nil
}

// 转换函数

// convertCommentToProto 将评论模型转换为protobuf格式
func convertCommentToProto(comment *model.Comment) *rest.Comment {
	if comment == nil {
		return nil
	}

	protoComment := &rest.Comment{
		Id:              comment.ID,
		ObjectId:        comment.ObjectID,
		ObjectType:      convertObjectTypeToProto(comment.ObjectType),
		UserId:          comment.UserID,
		UserName:        comment.UserName,
		UserAvatar:      comment.UserAvatar,
		Content:         comment.Content,
		ParentId:        comment.ParentID,
		RootId:          comment.RootID,
		ReplyToUserId:   comment.ReplyToUserID,
		ReplyToUserName: comment.ReplyToUserName,
		Status:          convertCommentStatusToProto(comment.Status),
		LikeCount:       comment.LikeCount,
		ReplyCount:      comment.ReplyCount,
		IsPinned:        comment.IsPinned,
		IsHot:           comment.IsHot,
		IpAddress:       comment.IPAddress,
		UserAgent:       comment.UserAgent,
		CreatedAt:       comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       comment.UpdatedAt.Format(time.RFC3339),
	}

	return protoComment
}

// convertCommentStatsToProto 将评论统计模型转换为protobuf格式
func convertCommentStatsToProto(stats *model.CommentStats) *rest.CommentStats {
	if stats == nil {
		return nil
	}

	return &rest.CommentStats{
		ObjectId:      stats.ObjectID,
		ObjectType:    convertObjectTypeToProto(stats.ObjectType),
		TotalCount:    stats.TotalCount,
		ApprovedCount: stats.ApprovedCount,
		PendingCount:  stats.PendingCount,
		TodayCount:    stats.TodayCount,
		HotCount:      stats.HotCount,
	}
}

// convertObjectTypeToProto 将对象类型转换为protobuf枚举
func convertObjectTypeToProto(objectType string) rest.CommentObjectType {
	switch objectType {
	case model.ObjectTypePost:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_POST
	case model.ObjectTypeArticle:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_ARTICLE
	case model.ObjectTypeVideo:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_VIDEO
	case model.ObjectTypeProduct:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_PRODUCT
	default:
		return rest.CommentObjectType_COMMENT_OBJECT_TYPE_UNSPECIFIED
	}
}

// convertObjectTypeFromProto 将protobuf枚举转换为对象类型
func convertObjectTypeFromProto(objectType rest.CommentObjectType) string {
	switch objectType {
	case rest.CommentObjectType_COMMENT_OBJECT_TYPE_POST:
		return model.ObjectTypePost
	case rest.CommentObjectType_COMMENT_OBJECT_TYPE_ARTICLE:
		return model.ObjectTypeArticle
	case rest.CommentObjectType_COMMENT_OBJECT_TYPE_VIDEO:
		return model.ObjectTypeVideo
	case rest.CommentObjectType_COMMENT_OBJECT_TYPE_PRODUCT:
		return model.ObjectTypeProduct
	default:
		return ""
	}
}

// convertCommentStatusToProto 将评论状态转换为protobuf枚举
func convertCommentStatusToProto(status string) rest.CommentStatus {
	switch status {
	case model.CommentStatusPending:
		return rest.CommentStatus_COMMENT_STATUS_PENDING
	case model.CommentStatusApproved:
		return rest.CommentStatus_COMMENT_STATUS_APPROVED
	case model.CommentStatusRejected:
		return rest.CommentStatus_COMMENT_STATUS_REJECTED
	case model.CommentStatusDeleted:
		return rest.CommentStatus_COMMENT_STATUS_DELETED
	default:
		return rest.CommentStatus_COMMENT_STATUS_UNSPECIFIED
	}
}

// convertCommentStatusFromProto 将protobuf枚举转换为评论状态
func convertCommentStatusFromProto(status rest.CommentStatus) string {
	switch status {
	case rest.CommentStatus_COMMENT_STATUS_PENDING:
		return model.CommentStatusPending
	case rest.CommentStatus_COMMENT_STATUS_APPROVED:
		return model.CommentStatusApproved
	case rest.CommentStatus_COMMENT_STATUS_REJECTED:
		return model.CommentStatusRejected
	case rest.CommentStatus_COMMENT_STATUS_DELETED:
		return model.CommentStatusDeleted
	default:
		return ""
	}
}
