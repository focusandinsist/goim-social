package converter

import (
	"time"

	"goim-social/api/rest"
	"goim-social/apps/interaction-service/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// InteractionModelToProto 将互动Model转换为Protobuf
func (c *Converter) InteractionModelToProto(interaction *model.Interaction) *rest.Interaction {
	if interaction == nil {
		return nil
	}

	return &rest.Interaction{
		Id:                    interaction.ID,
		UserId:                interaction.UserID,
		ObjectId:              interaction.ObjectID,
		InteractionObjectType: c.ObjectTypeToProto(interaction.ObjectType),
		InteractionType:       c.InteractionTypeToProto(interaction.InteractionType),
		Metadata:              interaction.Metadata,
		CreatedAt:             interaction.CreatedAt.Format(time.RFC3339),
	}
}

// InteractionModelsToProto 将互动Model列表转换为Protobuf列表
func (c *Converter) InteractionModelsToProto(interactions []*model.Interaction) []*rest.Interaction {
	if interactions == nil {
		return []*rest.Interaction{}
	}

	result := make([]*rest.Interaction, 0, len(interactions))
	for _, interaction := range interactions {
		if protoInteraction := c.InteractionModelToProto(interaction); protoInteraction != nil {
			result = append(result, protoInteraction)
		}
	}
	return result
}

// InteractionStatsModelToProto 将互动统计Model转换为Protobuf
func (c *Converter) InteractionStatsModelToProto(stats *model.InteractionStats) *rest.InteractionStats {
	if stats == nil {
		return nil
	}

	return &rest.InteractionStats{
		ObjectId:              stats.ObjectID,
		InteractionObjectType: c.ObjectTypeToProto(stats.ObjectType),
		LikeCount:             stats.LikeCount,
		FavoriteCount:         stats.FavoriteCount,
		ShareCount:            stats.ShareCount,
		RepostCount:           stats.RepostCount,
	}
}

// InteractionStatsModelsToProto 将互动统计Model列表转换为Protobuf列表
func (c *Converter) InteractionStatsModelsToProto(statsList []*model.InteractionStats) []*rest.InteractionStats {
	if statsList == nil {
		return []*rest.InteractionStats{}
	}

	result := make([]*rest.InteractionStats, 0, len(statsList))
	for _, stats := range statsList {
		if protoStats := c.InteractionStatsModelToProto(stats); protoStats != nil {
			result = append(result, protoStats)
		}
	}
	return result
}

// 枚举转换方法

// ObjectTypeToProto 将对象类型转换为protobuf枚举
func (c *Converter) ObjectTypeToProto(objectType string) rest.InteractionObjectType {
	switch objectType {
	case model.ObjectTypePost:
		return rest.InteractionObjectType_OBJECT_TYPE_POST
	case model.ObjectTypeComment:
		return rest.InteractionObjectType_OBJECT_TYPE_COMMENT
	case model.ObjectTypeUser:
		return rest.InteractionObjectType_OBJECT_TYPE_USER
	default:
		return rest.InteractionObjectType_OBJECT_TYPE_UNSPECIFIED
	}
}

// ObjectTypeFromProto 将protobuf枚举转换为对象类型
func (c *Converter) ObjectTypeFromProto(objectType rest.InteractionObjectType) string {
	switch objectType {
	case rest.InteractionObjectType_OBJECT_TYPE_POST:
		return model.ObjectTypePost
	case rest.InteractionObjectType_OBJECT_TYPE_COMMENT:
		return model.ObjectTypeComment
	case rest.InteractionObjectType_OBJECT_TYPE_USER:
		return model.ObjectTypeUser
	default:
		return ""
	}
}

// InteractionTypeToProto 将互动类型转换为protobuf枚举
func (c *Converter) InteractionTypeToProto(interactionType string) rest.InteractionType {
	switch interactionType {
	case model.InteractionTypeLike:
		return rest.InteractionType_INTERACTION_TYPE_LIKE
	case model.InteractionTypeFavorite:
		return rest.InteractionType_INTERACTION_TYPE_FAVORITE
	case model.InteractionTypeShare:
		return rest.InteractionType_INTERACTION_TYPE_SHARE
	case model.InteractionTypeRepost:
		return rest.InteractionType_INTERACTION_TYPE_REPOST
	default:
		return rest.InteractionType_INTERACTION_TYPE_UNSPECIFIED
	}
}

// InteractionTypeFromProto 将protobuf枚举转换为互动类型
func (c *Converter) InteractionTypeFromProto(interactionType rest.InteractionType) string {
	switch interactionType {
	case rest.InteractionType_INTERACTION_TYPE_LIKE:
		return model.InteractionTypeLike
	case rest.InteractionType_INTERACTION_TYPE_FAVORITE:
		return model.InteractionTypeFavorite
	case rest.InteractionType_INTERACTION_TYPE_SHARE:
		return model.InteractionTypeShare
	case rest.InteractionType_INTERACTION_TYPE_REPOST:
		return model.InteractionTypeRepost
	default:
		return ""
	}
}

// 响应构建方法

// BuildDoInteractionResponse 构建执行互动响应
func (c *Converter) BuildDoInteractionResponse(success bool, message string, interaction *model.Interaction) *rest.DoInteractionResponse {
	return &rest.DoInteractionResponse{
		Success:     success,
		Message:     message,
		Interaction: c.InteractionModelToProto(interaction),
	}
}

// BuildUndoInteractionResponse 构建取消互动响应
func (c *Converter) BuildUndoInteractionResponse(success bool, message string) *rest.UndoInteractionResponse {
	return &rest.UndoInteractionResponse{
		Success: success,
		Message: message,
	}
}

// BuildCheckInteractionResponse 构建检查互动状态响应
func (c *Converter) BuildCheckInteractionResponse(success bool, message string, hasInteraction bool, interaction *model.Interaction) *rest.CheckInteractionResponse {
	return &rest.CheckInteractionResponse{
		Success:        success,
		Message:        message,
		HasInteraction: hasInteraction,
		Interaction:    c.InteractionModelToProto(interaction),
	}
}

// BuildBatchCheckInteractionResponse 构建批量检查互动状态响应
func (c *Converter) BuildBatchCheckInteractionResponse(success bool, message string, interactions map[int64]bool) *rest.BatchCheckInteractionResponse {
	return &rest.BatchCheckInteractionResponse{
		Success:      success,
		Message:      message,
		Interactions: interactions,
	}
}

// BuildGetObjectStatsResponse 构建获取对象统计响应
func (c *Converter) BuildGetObjectStatsResponse(success bool, message string, stats *model.InteractionStats) *rest.GetObjectStatsResponse {
	return &rest.GetObjectStatsResponse{
		Success: success,
		Message: message,
		Stats:   c.InteractionStatsModelToProto(stats),
	}
}

// BuildGetBatchObjectStatsResponse 构建批量获取对象统计响应
func (c *Converter) BuildGetBatchObjectStatsResponse(success bool, message string, statsList []*model.InteractionStats) *rest.GetBatchObjectStatsResponse {
	return &rest.GetBatchObjectStatsResponse{
		Success: success,
		Message: message,
		Stats:   c.InteractionStatsModelsToProto(statsList),
	}
}

// BuildGetUserInteractionsResponse 构建获取用户互动列表响应
func (c *Converter) BuildGetUserInteractionsResponse(success bool, message string, interactions []*model.Interaction, total int64, page, pageSize int32) *rest.GetUserInteractionsResponse {
	return &rest.GetUserInteractionsResponse{
		Success:      success,
		Message:      message,
		Interactions: c.InteractionModelsToProto(interactions),
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}
}

// BuildGetObjectInteractionsResponse 构建获取对象互动列表响应
func (c *Converter) BuildGetObjectInteractionsResponse(success bool, message string, interactions []*model.Interaction, total int64, page, pageSize int32) *rest.GetObjectInteractionsResponse {
	return &rest.GetObjectInteractionsResponse{
		Success:      success,
		Message:      message,
		Interactions: c.InteractionModelsToProto(interactions),
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}
}

// 便捷方法：构建错误响应

// BuildErrorDoInteractionResponse 构建执行互动错误响应
func (c *Converter) BuildErrorDoInteractionResponse(message string) *rest.DoInteractionResponse {
	return c.BuildDoInteractionResponse(false, message, nil)
}

// BuildErrorUndoInteractionResponse 构建取消互动错误响应
func (c *Converter) BuildErrorUndoInteractionResponse(message string) *rest.UndoInteractionResponse {
	return c.BuildUndoInteractionResponse(false, message)
}

// BuildErrorCheckInteractionResponse 构建检查互动状态错误响应
func (c *Converter) BuildErrorCheckInteractionResponse(message string) *rest.CheckInteractionResponse {
	return c.BuildCheckInteractionResponse(false, message, false, nil)
}

// BuildErrorBatchCheckInteractionResponse 构建批量检查互动状态错误响应
func (c *Converter) BuildErrorBatchCheckInteractionResponse(message string) *rest.BatchCheckInteractionResponse {
	return c.BuildBatchCheckInteractionResponse(false, message, nil)
}

// BuildErrorGetObjectStatsResponse 构建获取对象统计错误响应
func (c *Converter) BuildErrorGetObjectStatsResponse(message string) *rest.GetObjectStatsResponse {
	return c.BuildGetObjectStatsResponse(false, message, nil)
}

// BuildErrorGetBatchObjectStatsResponse 构建批量获取对象统计错误响应
func (c *Converter) BuildErrorGetBatchObjectStatsResponse(message string) *rest.GetBatchObjectStatsResponse {
	return c.BuildGetBatchObjectStatsResponse(false, message, nil)
}

// BuildErrorGetUserInteractionsResponse 构建获取用户互动列表错误响应
func (c *Converter) BuildErrorGetUserInteractionsResponse(message string) *rest.GetUserInteractionsResponse {
	return c.BuildGetUserInteractionsResponse(false, message, nil, 0, 0, 0)
}

// BuildErrorGetObjectInteractionsResponse 构建获取对象互动列表错误响应
func (c *Converter) BuildErrorGetObjectInteractionsResponse(message string) *rest.GetObjectInteractionsResponse {
	return c.BuildGetObjectInteractionsResponse(false, message, nil, 0, 0, 0)
}

// 便捷方法：构建成功响应

// BuildSuccessDoInteractionResponse 构建执行互动成功响应
func (c *Converter) BuildSuccessDoInteractionResponse(interaction *model.Interaction) *rest.DoInteractionResponse {
	return c.BuildDoInteractionResponse(true, "操作成功", interaction)
}

// BuildSuccessUndoInteractionResponse 构建取消互动成功响应
func (c *Converter) BuildSuccessUndoInteractionResponse() *rest.UndoInteractionResponse {
	return c.BuildUndoInteractionResponse(true, "取消成功")
}

// BuildSuccessCheckInteractionResponse 构建检查互动状态成功响应
func (c *Converter) BuildSuccessCheckInteractionResponse(hasInteraction bool, interaction *model.Interaction) *rest.CheckInteractionResponse {
	return c.BuildCheckInteractionResponse(true, "查询成功", hasInteraction, interaction)
}

// BuildSuccessBatchCheckInteractionResponse 构建批量检查互动状态成功响应
func (c *Converter) BuildSuccessBatchCheckInteractionResponse(interactions map[int64]bool) *rest.BatchCheckInteractionResponse {
	return c.BuildBatchCheckInteractionResponse(true, "查询成功", interactions)
}

// BuildSuccessGetObjectStatsResponse 构建获取对象统计成功响应
func (c *Converter) BuildSuccessGetObjectStatsResponse(stats *model.InteractionStats) *rest.GetObjectStatsResponse {
	return c.BuildGetObjectStatsResponse(true, "获取成功", stats)
}

// BuildSuccessGetBatchObjectStatsResponse 构建批量获取对象统计成功响应
func (c *Converter) BuildSuccessGetBatchObjectStatsResponse(statsList []*model.InteractionStats) *rest.GetBatchObjectStatsResponse {
	return c.BuildGetBatchObjectStatsResponse(true, "获取成功", statsList)
}

// BuildSuccessGetUserInteractionsResponse 构建获取用户互动列表成功响应
func (c *Converter) BuildSuccessGetUserInteractionsResponse(interactions []*model.Interaction, total int64, page, pageSize int32) *rest.GetUserInteractionsResponse {
	return c.BuildGetUserInteractionsResponse(true, "获取成功", interactions, total, page, pageSize)
}

// BuildSuccessGetObjectInteractionsResponse 构建获取对象互动列表成功响应
func (c *Converter) BuildSuccessGetObjectInteractionsResponse(interactions []*model.Interaction, total int64, page, pageSize int32) *rest.GetObjectInteractionsResponse {
	return c.BuildGetObjectInteractionsResponse(true, "获取成功", interactions, total, page, pageSize)
}
