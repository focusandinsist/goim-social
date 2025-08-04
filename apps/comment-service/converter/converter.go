package converter

import (
	"time"

	"goim-social/api/rest"
	"goim-social/apps/comment-service/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// CommentModelToProto 将评论Model转换为Protobuf
func (c *Converter) CommentModelToProto(comment *model.Comment) *rest.Comment {
	if comment == nil {
		return nil
	}

	return &rest.Comment{
		Id:              comment.ID,
		ObjectId:        comment.ObjectID,
		ObjectType:      c.ObjectTypeToProto(comment.ObjectType),
		UserId:          comment.UserID,
		UserName:        comment.UserName,
		UserAvatar:      comment.UserAvatar,
		Content:         comment.Content,
		ParentId:        comment.ParentID,
		RootId:          comment.RootID,
		ReplyToUserId:   comment.ReplyToUserID,
		ReplyToUserName: comment.ReplyToUserName,
		Status:          c.CommentStatusToProto(comment.Status),
		LikeCount:       comment.LikeCount,
		ReplyCount:      comment.ReplyCount,
		IsPinned:        comment.IsPinned,
		IsHot:           comment.IsHot,
		IpAddress:       comment.IPAddress,
		UserAgent:       comment.UserAgent,
		CreatedAt:       comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       comment.UpdatedAt.Format(time.RFC3339),
	}
}

// CommentModelsToProto 将评论Model列表转换为Protobuf列表
func (c *Converter) CommentModelsToProto(comments []*model.Comment) []*rest.Comment {
	if comments == nil {
		return []*rest.Comment{}
	}

	result := make([]*rest.Comment, 0, len(comments))
	for _, comment := range comments {
		if protoComment := c.CommentModelToProto(comment); protoComment != nil {
			result = append(result, protoComment)
		}
	}
	return result
}

// CommentStatsModelToProto 将评论统计Model转换为Protobuf
func (c *Converter) CommentStatsModelToProto(stats *model.CommentStats) *rest.CommentStats {
	if stats == nil {
		return nil
	}

	return &rest.CommentStats{
		ObjectId:      stats.ObjectID,
		ObjectType:    c.ObjectTypeToProto(stats.ObjectType),
		TotalCount:    stats.TotalCount,
		ApprovedCount: stats.ApprovedCount,
		PendingCount:  stats.PendingCount,
		TodayCount:    stats.TodayCount,
		HotCount:      stats.HotCount,
	}
}

// CommentStatsModelsToProto 将评论统计Model列表转换为Protobuf列表
func (c *Converter) CommentStatsModelsToProto(statsList []*model.CommentStats) []*rest.CommentStats {
	if statsList == nil {
		return []*rest.CommentStats{}
	}

	result := make([]*rest.CommentStats, 0, len(statsList))
	for _, stats := range statsList {
		if protoStats := c.CommentStatsModelToProto(stats); protoStats != nil {
			result = append(result, protoStats)
		}
	}
	return result
}

// 枚举转换方法

// ObjectTypeToProto 将对象类型转换为protobuf枚举
func (c *Converter) ObjectTypeToProto(objectType string) rest.CommentObjectType {
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

// ObjectTypeFromProto 将protobuf枚举转换为对象类型
func (c *Converter) ObjectTypeFromProto(objectType rest.CommentObjectType) string {
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

// CommentStatusToProto 将评论状态转换为protobuf枚举
func (c *Converter) CommentStatusToProto(status string) rest.CommentStatus {
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

// CommentStatusFromProto 将protobuf枚举转换为评论状态
func (c *Converter) CommentStatusFromProto(status rest.CommentStatus) string {
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

// 响应构建方法

// BuildCreateCommentResponse 构建创建评论响应
func (c *Converter) BuildCreateCommentResponse(success bool, message string, comment *model.Comment) *rest.CreateCommentResponse {
	return &rest.CreateCommentResponse{
		Success: success,
		Message: message,
		Comment: c.CommentModelToProto(comment),
	}
}

// BuildUpdateCommentResponse 构建更新评论响应
func (c *Converter) BuildUpdateCommentResponse(success bool, message string, comment *model.Comment) *rest.UpdateCommentResponse {
	return &rest.UpdateCommentResponse{
		Success: success,
		Message: message,
		Comment: c.CommentModelToProto(comment),
	}
}

// BuildDeleteCommentResponse 构建删除评论响应
func (c *Converter) BuildDeleteCommentResponse(success bool, message string) *rest.DeleteCommentResponse {
	return &rest.DeleteCommentResponse{
		Success: success,
		Message: message,
	}
}

// BuildGetCommentResponse 构建获取评论响应
func (c *Converter) BuildGetCommentResponse(success bool, message string, comment *model.Comment) *rest.GetCommentResponse {
	return &rest.GetCommentResponse{
		Success: success,
		Message: message,
		Comment: c.CommentModelToProto(comment),
	}
}

// BuildGetCommentsResponse 构建获取评论列表响应
func (c *Converter) BuildGetCommentsResponse(success bool, message string, comments []*model.Comment, total int64, page, pageSize int32) *rest.GetCommentsResponse {
	return &rest.GetCommentsResponse{
		Success:  success,
		Message:  message,
		Comments: c.CommentModelsToProto(comments),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildGetUserCommentsResponse 构建获取用户评论响应
func (c *Converter) BuildGetUserCommentsResponse(success bool, message string, comments []*model.Comment, total int64, page, pageSize int32) *rest.GetUserCommentsResponse {
	return &rest.GetUserCommentsResponse{
		Success:  success,
		Message:  message,
		Comments: c.CommentModelsToProto(comments),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildGetCommentStatsResponse 构建获取评论统计响应
func (c *Converter) BuildGetCommentStatsResponse(success bool, message string, stats *model.CommentStats) *rest.GetCommentStatsResponse {
	return &rest.GetCommentStatsResponse{
		Success: success,
		Message: message,
		Stats:   c.CommentStatsModelToProto(stats),
	}
}

// BuildGetBatchCommentStatsResponse 构建批量获取评论统计响应
func (c *Converter) BuildGetBatchCommentStatsResponse(success bool, message string, statsList []*model.CommentStats) *rest.GetBatchCommentStatsResponse {
	return &rest.GetBatchCommentStatsResponse{
		Success: success,
		Message: message,
		Stats:   c.CommentStatsModelsToProto(statsList),
	}
}

// BuildModerateCommentResponse 构建审核评论响应
func (c *Converter) BuildModerateCommentResponse(success bool, message string, comment *model.Comment) *rest.ModerateCommentResponse {
	return &rest.ModerateCommentResponse{
		Success: success,
		Message: message,
		Comment: c.CommentModelToProto(comment),
	}
}

// BuildPinCommentResponse 构建置顶评论响应
func (c *Converter) BuildPinCommentResponse(success bool, message string) *rest.PinCommentResponse {
	return &rest.PinCommentResponse{
		Success: success,
		Message: message,
	}
}

// 便捷方法：构建错误响应

// BuildErrorCreateCommentResponse 构建创建评论错误响应
func (c *Converter) BuildErrorCreateCommentResponse(message string) *rest.CreateCommentResponse {
	return c.BuildCreateCommentResponse(false, message, nil)
}

// BuildErrorUpdateCommentResponse 构建更新评论错误响应
func (c *Converter) BuildErrorUpdateCommentResponse(message string) *rest.UpdateCommentResponse {
	return c.BuildUpdateCommentResponse(false, message, nil)
}

// BuildErrorDeleteCommentResponse 构建删除评论错误响应
func (c *Converter) BuildErrorDeleteCommentResponse(message string) *rest.DeleteCommentResponse {
	return c.BuildDeleteCommentResponse(false, message)
}

// BuildErrorGetCommentResponse 构建获取评论错误响应
func (c *Converter) BuildErrorGetCommentResponse(message string) *rest.GetCommentResponse {
	return c.BuildGetCommentResponse(false, message, nil)
}

// BuildErrorGetCommentsResponse 构建获取评论列表错误响应
func (c *Converter) BuildErrorGetCommentsResponse(message string) *rest.GetCommentsResponse {
	return c.BuildGetCommentsResponse(false, message, nil, 0, 0, 0)
}

// BuildErrorGetUserCommentsResponse 构建获取用户评论错误响应
func (c *Converter) BuildErrorGetUserCommentsResponse(message string) *rest.GetUserCommentsResponse {
	return c.BuildGetUserCommentsResponse(false, message, nil, 0, 0, 0)
}

// BuildErrorGetCommentStatsResponse 构建获取评论统计错误响应
func (c *Converter) BuildErrorGetCommentStatsResponse(message string) *rest.GetCommentStatsResponse {
	return c.BuildGetCommentStatsResponse(false, message, nil)
}

// BuildErrorGetBatchCommentStatsResponse 构建批量获取评论统计错误响应
func (c *Converter) BuildErrorGetBatchCommentStatsResponse(message string) *rest.GetBatchCommentStatsResponse {
	return c.BuildGetBatchCommentStatsResponse(false, message, nil)
}

// BuildErrorModerateCommentResponse 构建审核评论错误响应
func (c *Converter) BuildErrorModerateCommentResponse(message string) *rest.ModerateCommentResponse {
	return c.BuildModerateCommentResponse(false, message, nil)
}

// BuildErrorPinCommentResponse 构建置顶评论错误响应
func (c *Converter) BuildErrorPinCommentResponse(message string) *rest.PinCommentResponse {
	return c.BuildPinCommentResponse(false, message)
}

// 便捷方法：构建成功响应

// BuildSuccessCreateCommentResponse 构建创建评论成功响应
func (c *Converter) BuildSuccessCreateCommentResponse(comment *model.Comment) *rest.CreateCommentResponse {
	return c.BuildCreateCommentResponse(true, "创建成功", comment)
}

// BuildSuccessUpdateCommentResponse 构建更新评论成功响应
func (c *Converter) BuildSuccessUpdateCommentResponse(comment *model.Comment) *rest.UpdateCommentResponse {
	return c.BuildUpdateCommentResponse(true, "评论更新成功", comment)
}

// BuildSuccessDeleteCommentResponse 构建删除评论成功响应
func (c *Converter) BuildSuccessDeleteCommentResponse() *rest.DeleteCommentResponse {
	return c.BuildDeleteCommentResponse(true, "评论删除成功")
}

// BuildSuccessGetCommentResponse 构建获取评论成功响应
func (c *Converter) BuildSuccessGetCommentResponse(comment *model.Comment) *rest.GetCommentResponse {
	return c.BuildGetCommentResponse(true, "获取成功", comment)
}

// BuildSuccessGetCommentsResponse 构建获取评论列表成功响应
func (c *Converter) BuildSuccessGetCommentsResponse(comments []*model.Comment, total int64, page, pageSize int32) *rest.GetCommentsResponse {
	return c.BuildGetCommentsResponse(true, "获取成功", comments, total, page, pageSize)
}

// BuildSuccessGetUserCommentsResponse 构建获取用户评论成功响应
func (c *Converter) BuildSuccessGetUserCommentsResponse(comments []*model.Comment, total int64, page, pageSize int32) *rest.GetUserCommentsResponse {
	return c.BuildGetUserCommentsResponse(true, "获取成功", comments, total, page, pageSize)
}
