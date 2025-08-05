package converter

import (
	"goim-social/api/rest"
	"goim-social/apps/friend-service/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// FriendModelToProto 将好友Model转换为Protobuf
func (c *Converter) FriendModelToProto(friend *model.Friend) *rest.FriendInfo {
	if friend == nil {
		return nil
	}
	return &rest.FriendInfo{
		UserId:    friend.UserID,
		FriendId:  friend.FriendID,
		Remark:    friend.Remark,
		CreatedAt: friend.CreatedAt.Unix(),
	}
}

// FriendModelsToProto 将好友Model列表转换为Protobuf列表
func (c *Converter) FriendModelsToProto(friends []*model.Friend) []*rest.FriendInfo {
	if friends == nil {
		return []*rest.FriendInfo{}
	}

	result := make([]*rest.FriendInfo, 0, len(friends))
	for _, friend := range friends {
		if protoFriend := c.FriendModelToProto(friend); protoFriend != nil {
			result = append(result, protoFriend)
		}
	}
	return result
}

// FriendApplyModelToProto 将好友申请Model转换为Protobuf
func (c *Converter) FriendApplyModelToProto(apply *model.FriendApply) *rest.FriendApplyInfo {
	if apply == nil {
		return nil
	}
	return &rest.FriendApplyInfo{
		ApplicantId: apply.ApplicantID,
		Remark:      apply.Remark,
		Timestamp:   apply.CreatedAt.Unix(),
		Status:      apply.Status,
	}
}

// FriendApplyModelsToProto 将好友申请Model列表转换为Protobuf列表
func (c *Converter) FriendApplyModelsToProto(applies []*model.FriendApply) []*rest.FriendApplyInfo {
	if applies == nil {
		return []*rest.FriendApplyInfo{}
	}

	result := make([]*rest.FriendApplyInfo, 0, len(applies))
	for _, apply := range applies {
		if protoApply := c.FriendApplyModelToProto(apply); protoApply != nil {
			result = append(result, protoApply)
		}
	}
	return result
}

// 响应构建方法

// BuildDeleteFriendResponse 构建删除好友响应
func (c *Converter) BuildDeleteFriendResponse(success bool, message string) *rest.DeleteFriendResponse {
	return &rest.DeleteFriendResponse{
		Success: success,
		Message: message,
	}
}

// BuildListFriendsResponse 构建好友列表响应
func (c *Converter) BuildListFriendsResponse(success bool, message string, friends []*model.Friend, page, pageSize int32) *rest.ListFriendsResponse {
	return &rest.ListFriendsResponse{
		Success:  success,
		Message:  message,
		Friends:  c.FriendModelsToProto(friends),
		Total:    int32(len(friends)),
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildFriendProfileResponse 构建好友简介响应
func (c *Converter) BuildFriendProfileResponse(success bool, message, alias, nickname, avatar, gender string, age int32) *rest.FriendProfileResponse {
	return &rest.FriendProfileResponse{
		Success:  success,
		Message:  message,
		Alias:    alias,
		Nickname: nickname,
		Avatar:   avatar,
		Age:      age,
		Gender:   gender,
	}
}

// BuildSetFriendAliasResponse 构建设置好友备注响应
func (c *Converter) BuildSetFriendAliasResponse(success bool, message string) *rest.SetFriendAliasResponse {
	return &rest.SetFriendAliasResponse{
		Success: success,
		Message: message,
	}
}

// BuildListFriendApplyResponse 构建好友申请列表响应
func (c *Converter) BuildListFriendApplyResponse(applies []*model.FriendApply) *rest.ListFriendApplyResponse {
	return &rest.ListFriendApplyResponse{
		Applies: c.FriendApplyModelsToProto(applies),
	}
}

// BuildApplyFriendResponse 构建申请好友响应
func (c *Converter) BuildApplyFriendResponse(success bool, message string) *rest.ApplyFriendResponse {
	return &rest.ApplyFriendResponse{
		Success: success,
		Message: message,
	}
}

// BuildRespondFriendApplyResponse 构建回应好友申请响应
func (c *Converter) BuildRespondFriendApplyResponse(success bool, message string) *rest.RespondFriendApplyResponse {
	return &rest.RespondFriendApplyResponse{
		Success: success,
		Message: message,
	}
}

// BuildNotifyFriendEventResponse 构建通知好友事件响应
func (c *Converter) BuildNotifyFriendEventResponse(success bool, message string) *rest.NotifyFriendEventResponse {
	return &rest.NotifyFriendEventResponse{
		Success: success,
		Message: message,
	}
}
