package converter

import (
	"goim-social/api/rest"
	"goim-social/apps/social-service/internal/dao"
	"goim-social/apps/social-service/internal/model"
)

// Converter 转换器
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// ============ 好友相关转换 ============

// BuildSendFriendRequestResponse 构建发送好友申请响应
func (c *Converter) BuildSendFriendRequestResponse(success bool, message string) *rest.ApplyFriendResponse {
	return &rest.ApplyFriendResponse{
		Success: success,
		Message: message,
	}
}

// BuildAcceptFriendRequestResponse 构建接受好友申请响应
func (c *Converter) BuildAcceptFriendRequestResponse(success bool, message string) *rest.RespondFriendApplyResponse {
	return &rest.RespondFriendApplyResponse{
		Success: success,
		Message: message,
	}
}

// BuildRejectFriendRequestResponse 构建拒绝好友申请响应
func (c *Converter) BuildRejectFriendRequestResponse(success bool, message string) *rest.RespondFriendApplyResponse {
	return &rest.RespondFriendApplyResponse{
		Success: success,
		Message: message,
	}
}

// BuildDeleteFriendResponse 构建删除好友响应
func (c *Converter) BuildDeleteFriendResponse(success bool, message string) *rest.DeleteFriendResponse {
	return &rest.DeleteFriendResponse{
		Success: success,
		Message: message,
	}
}

// BuildGetFriendListResponse 构建获取好友列表响应
func (c *Converter) BuildGetFriendListResponse(success bool, message string, friends []*model.Friend) *rest.ListFriendsResponse {
	var friendInfos []*rest.FriendInfo
	if friends != nil {
		friendInfos = make([]*rest.FriendInfo, len(friends))
		for i, friend := range friends {
			friendInfos[i] = &rest.FriendInfo{
				UserId:    friend.UserID,
				FriendId:  friend.FriendID,
				Remark:    friend.Remark,
				CreatedAt: friend.CreatedAt.Unix(),
			}
		}
	}

	return &rest.ListFriendsResponse{
		Success: success,
		Message: message,
		Friends: friendInfos,
	}
}

// BuildGetFriendApplyListResponse 构建获取好友申请列表响应
func (c *Converter) BuildGetFriendApplyListResponse(success bool, message string, applies []*model.FriendApply) *rest.ListFriendApplyResponse {
	var applyInfos []*rest.FriendApplyInfo
	if applies != nil {
		applyInfos = make([]*rest.FriendApplyInfo, len(applies))
		for i, apply := range applies {
			applyInfos[i] = &rest.FriendApplyInfo{
				ApplicantId: apply.ApplicantID,
				Remark:      apply.Remark,
				Status:      apply.Status,
				Timestamp:   apply.CreatedAt.Unix(),
			}
		}
	}

	return &rest.ListFriendApplyResponse{
		Applies: applyInfos,
	}
}

// ============ 群组相关转换 ============

// BuildCreateGroupResponse 构建创建群组响应
func (c *Converter) BuildCreateGroupResponse(success bool, message string, group *model.Group) *rest.CreateGroupResponse {
	var groupInfo *rest.GroupInfo
	if group != nil {
		groupInfo = &rest.GroupInfo{
			Id:           group.ID,
			Name:         group.Name,
			Description:  group.Description,
			Avatar:       group.Avatar,
			OwnerId:      group.OwnerID,
			MemberCount:  group.MemberCount,
			MaxMembers:   group.MaxMembers,
			IsPublic:     group.IsPublic,
			Announcement: group.Announcement,
			CreatedAt:    group.CreatedAt.Unix(),
			UpdatedAt:    group.UpdatedAt.Unix(),
		}
	}

	return &rest.CreateGroupResponse{
		Success: success,
		Message: message,
		Group:   groupInfo,
	}
}

// BuildGetGroupResponse 构建获取群组信息响应
func (c *Converter) BuildGetGroupResponse(success bool, message string, group *model.Group) *rest.GetGroupInfoResponse {
	var groupInfo *rest.GroupInfo
	if group != nil {
		groupInfo = &rest.GroupInfo{
			Id:           group.ID,
			Name:         group.Name,
			Description:  group.Description,
			Avatar:       group.Avatar,
			OwnerId:      group.OwnerID,
			MemberCount:  group.MemberCount,
			MaxMembers:   group.MaxMembers,
			IsPublic:     group.IsPublic,
			Announcement: group.Announcement,
			CreatedAt:    group.CreatedAt.Unix(),
			UpdatedAt:    group.UpdatedAt.Unix(),
		}
	}

	return &rest.GetGroupInfoResponse{
		Success: success,
		Message: message,
		Group:   groupInfo,
	}
}

// BuildUpdateGroupResponse 构建更新群组响应
func (c *Converter) BuildUpdateGroupResponse(success bool, message string) *rest.GetGroupInfoResponse {
	return &rest.GetGroupInfoResponse{
		Success: success,
		Message: message,
	}
}

// BuildJoinGroupResponse 构建加入群组响应
func (c *Converter) BuildJoinGroupResponse(success bool, message string) *rest.JoinGroupResponse {
	return &rest.JoinGroupResponse{
		Success: success,
		Message: message,
	}
}

// BuildLeaveGroupResponse 构建离开群组响应
func (c *Converter) BuildLeaveGroupResponse(success bool, message string) *rest.LeaveGroupResponse {
	return &rest.LeaveGroupResponse{
		Success: success,
		Message: message,
	}
}

// BuildGetGroupMembersResponse 构建获取群成员列表响应
func (c *Converter) BuildGetGroupMembersResponse(success bool, message string, members []*model.GroupMember) *rest.GetGroupInfoResponse {
	var memberInfos []*rest.GroupMemberInfo
	if members != nil {
		memberInfos = make([]*rest.GroupMemberInfo, len(members))
		for i, member := range members {
			memberInfos[i] = &rest.GroupMemberInfo{
				UserId:   member.UserID,
				GroupId:  member.GroupID,
				Role:     member.Role,
				Nickname: member.Nickname,
				JoinedAt: member.JoinedAt.Unix(),
			}
		}
	}

	return &rest.GetGroupInfoResponse{
		Success: success,
		Message: message,
		Members: memberInfos,
	}
}

// ============ 统一社交关系查询转换 ============

// BuildValidateFriendshipResponse 构建验证好友关系响应
func (c *Converter) BuildValidateFriendshipResponse(success bool, message string, isFriend bool) *rest.ValidateFriendshipResponse {
	return &rest.ValidateFriendshipResponse{
		Success:  success,
		Message:  message,
		IsFriend: isFriend,
	}
}

// BuildValidateGroupMembershipResponse 构建验证群成员关系响应
func (c *Converter) BuildValidateGroupMembershipResponse(success bool, message string, isMember bool) *rest.GetGroupInfoResponse {
	// 简化实现，只返回基本信息
	return &rest.GetGroupInfoResponse{
		Success: success,
		Message: message,
	}
}

// BuildGetUserSocialInfoResponse 构建获取用户社交信息响应
func (c *Converter) BuildGetUserSocialInfoResponse(success bool, message string, socialInfo *dao.SocialInfo) *rest.GetUserSocialInfoResponse {
	var info *rest.UserSocialInfo
	if socialInfo != nil {
		info = &rest.UserSocialInfo{
			UserId:      socialInfo.UserID,
			FriendCount: int32(socialInfo.FriendCount),
			GroupCount:  int32(socialInfo.GroupCount),
			FriendIds:   socialInfo.FriendIDs,
			GroupIds:    socialInfo.GroupIDs,
		}
	}

	return &rest.GetUserSocialInfoResponse{
		Success:    success,
		Message:    message,
		SocialInfo: info,
	}
}

// ============ 错误响应构建 ============

// BuildErrorSendFriendRequestResponse 构建发送好友申请错误响应
func (c *Converter) BuildErrorSendFriendRequestResponse(message string) *rest.ApplyFriendResponse {
	return c.BuildSendFriendRequestResponse(false, message)
}

// BuildErrorAcceptFriendRequestResponse 构建接受好友申请错误响应
func (c *Converter) BuildErrorAcceptFriendRequestResponse(message string) *rest.RespondFriendApplyResponse {
	return c.BuildAcceptFriendRequestResponse(false, message)
}

// BuildErrorRejectFriendRequestResponse 构建拒绝好友申请错误响应
func (c *Converter) BuildErrorRejectFriendRequestResponse(message string) *rest.RespondFriendApplyResponse {
	return c.BuildRejectFriendRequestResponse(false, message)
}

// BuildErrorDeleteFriendResponse 构建删除好友错误响应
func (c *Converter) BuildErrorDeleteFriendResponse(message string) *rest.DeleteFriendResponse {
	return c.BuildDeleteFriendResponse(false, message)
}

// BuildErrorGetFriendListResponse 构建获取好友列表错误响应
func (c *Converter) BuildErrorGetFriendListResponse(message string) *rest.ListFriendsResponse {
	return c.BuildGetFriendListResponse(false, message, nil)
}

// BuildErrorGetFriendApplyListResponse 构建获取好友申请列表错误响应
func (c *Converter) BuildErrorGetFriendApplyListResponse(message string) *rest.ListFriendApplyResponse {
	return c.BuildGetFriendApplyListResponse(false, message, nil)
}

// BuildErrorCreateGroupResponse 构建创建群组错误响应
func (c *Converter) BuildErrorCreateGroupResponse(message string) *rest.CreateGroupResponse {
	return c.BuildCreateGroupResponse(false, message, nil)
}

// BuildErrorGetGroupResponse 构建获取群组信息错误响应
func (c *Converter) BuildErrorGetGroupResponse(message string) *rest.GetGroupInfoResponse {
	return c.BuildGetGroupResponse(false, message, nil)
}

// BuildErrorUpdateGroupResponse 构建更新群组错误响应
func (c *Converter) BuildErrorUpdateGroupResponse(message string) *rest.GetGroupInfoResponse {
	return c.BuildUpdateGroupResponse(false, message)
}

// BuildErrorJoinGroupResponse 构建加入群组错误响应
func (c *Converter) BuildErrorJoinGroupResponse(message string) *rest.JoinGroupResponse {
	return c.BuildJoinGroupResponse(false, message)
}

// BuildErrorLeaveGroupResponse 构建离开群组错误响应
func (c *Converter) BuildErrorLeaveGroupResponse(message string) *rest.LeaveGroupResponse {
	return c.BuildLeaveGroupResponse(false, message)
}

// BuildErrorGetGroupMembersResponse 构建获取群成员列表错误响应
func (c *Converter) BuildErrorGetGroupMembersResponse(message string) *rest.GetGroupInfoResponse {
	return c.BuildGetGroupMembersResponse(false, message, nil)
}

// BuildErrorValidateFriendshipResponse 构建验证好友关系错误响应
func (c *Converter) BuildErrorValidateFriendshipResponse(message string) *rest.ValidateFriendshipResponse {
	return c.BuildValidateFriendshipResponse(false, message, false)
}

// BuildErrorValidateGroupMembershipResponse 构建验证群成员关系错误响应
func (c *Converter) BuildErrorValidateGroupMembershipResponse(message string) *rest.GetGroupInfoResponse {
	return c.BuildValidateGroupMembershipResponse(false, message, false)
}

// BuildErrorGetUserSocialInfoResponse 构建获取用户社交信息错误响应
func (c *Converter) BuildErrorGetUserSocialInfoResponse(message string) *rest.GetUserSocialInfoResponse {
	return c.BuildGetUserSocialInfoResponse(false, message, nil)
}
