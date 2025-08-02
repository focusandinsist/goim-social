package handler

import (
	"goim-social/api/rest"
	"goim-social/apps/interaction-service/model"
	"time"
)

// convertInteractionToProto 将互动模型转换为protobuf格式
func convertInteractionToProto(interaction *model.Interaction) *rest.Interaction {
	if interaction == nil {
		return nil
	}
	return &rest.Interaction{
		Id:                    interaction.ID,
		UserId:                interaction.UserID,
		ObjectId:              interaction.ObjectID,
		InteractionObjectType: convertObjectTypeToProto(interaction.ObjectType),
		InteractionType:       convertInteractionTypeToProto(interaction.InteractionType),
		Metadata:              interaction.Metadata,
		CreatedAt:             interaction.CreatedAt.Format(time.RFC3339),
	}
}

// convertStatsToProto 将统计模型转换为protobuf格式
func convertStatsToProto(stats *model.InteractionStats) *rest.InteractionStats {
	if stats == nil {
		return nil
	}
	return &rest.InteractionStats{
		ObjectId:              stats.ObjectID,
		InteractionObjectType: convertObjectTypeToProto(stats.ObjectType),
		LikeCount:             stats.LikeCount,
		FavoriteCount:         stats.FavoriteCount,
		ShareCount:            stats.ShareCount,
		RepostCount:           stats.RepostCount,
	}
}

// convertObjectTypeToProto 将对象类型转换为protobuf枚举
func convertObjectTypeToProto(objectType string) rest.InteractionObjectType {
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

// convertObjectTypeFromProto 将protobuf枚举转换为对象类型
func convertObjectTypeFromProto(objectType rest.InteractionObjectType) string {
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

// convertInteractionTypeToProto 将互动类型转换为protobuf枚举
func convertInteractionTypeToProto(interactionType string) rest.InteractionType {
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

// convertInteractionTypeFromProto 将protobuf枚举转换为互动类型
func convertInteractionTypeFromProto(interactionType rest.InteractionType) string {
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
