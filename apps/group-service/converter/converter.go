package converter

import (
	"goim-social/api/rest"
	"goim-social/apps/group-service/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// GroupModelToProto 将群组Model转换为Protobuf
func (c *Converter) GroupModelToProto(group *model.Group) *rest.GroupInfo {
	if group == nil {
		return nil
	}
	return &rest.GroupInfo{
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

// GroupModelsToProto 将群组Model列表转换为Protobuf列表
func (c *Converter) GroupModelsToProto(groups []*model.Group) []*rest.GroupInfo {
	if groups == nil {
		return []*rest.GroupInfo{}
	}

	result := make([]*rest.GroupInfo, 0, len(groups))
	for _, group := range groups {
		if protoGroup := c.GroupModelToProto(group); protoGroup != nil {
			result = append(result, protoGroup)
		}
	}
	return result
}

// GroupMemberModelToProto 将群成员Model转换为Protobuf
func (c *Converter) GroupMemberModelToProto(member *model.GroupMember) *rest.GroupMemberInfo {
	if member == nil {
		return nil
	}
	return &rest.GroupMemberInfo{
		UserId:   member.UserID,
		GroupId:  member.GroupID,
		Role:     member.Role,
		Nickname: member.Nickname,
		JoinedAt: member.JoinedAt.Unix(),
	}
}

// GroupMemberModelsToProto 将群成员Model列表转换为Protobuf列表
func (c *Converter) GroupMemberModelsToProto(members []*model.GroupMember) []*rest.GroupMemberInfo {
	if members == nil {
		return []*rest.GroupMemberInfo{}
	}

	result := make([]*rest.GroupMemberInfo, 0, len(members))
	for _, member := range members {
		if protoMember := c.GroupMemberModelToProto(member); protoMember != nil {
			result = append(result, protoMember)
		}
	}
	return result
}

// 响应构建方法

// BuildCreateGroupResponse 构建创建群组响应
func (c *Converter) BuildCreateGroupResponse(success bool, message string, group *model.Group) *rest.CreateGroupResponse {
	return &rest.CreateGroupResponse{
		Success: success,
		Message: message,
		Group:   c.GroupModelToProto(group),
	}
}

// BuildSearchGroupResponse 构建搜索群组响应
func (c *Converter) BuildSearchGroupResponse(success bool, message string, groups []*model.Group, total int64, page, pageSize int32) *rest.SearchGroupResponse {
	return &rest.SearchGroupResponse{
		Success:  success,
		Message:  message,
		Groups:   c.GroupModelsToProto(groups),
		Total:    int32(total),
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildGetGroupInfoResponse 构建获取群组信息响应
func (c *Converter) BuildGetGroupInfoResponse(success bool, message string, group *model.Group, members []*model.GroupMember) *rest.GetGroupInfoResponse {
	return &rest.GetGroupInfoResponse{
		Success: success,
		Message: message,
		Group:   c.GroupModelToProto(group),
		Members: c.GroupMemberModelsToProto(members),
	}
}

// BuildGetUserGroupsResponse 构建获取用户群组列表响应
func (c *Converter) BuildGetUserGroupsResponse(success bool, message string, groups []*model.Group, total int64, page, pageSize int32) *rest.GetUserGroupsResponse {
	return &rest.GetUserGroupsResponse{
		Success:  success,
		Message:  message,
		Groups:   c.GroupModelsToProto(groups),
		Total:    int32(total),
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildDisbandGroupResponse 构建解散群组响应
func (c *Converter) BuildDisbandGroupResponse(success bool, message string) *rest.DisbandGroupResponse {
	return &rest.DisbandGroupResponse{
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

// BuildLeaveGroupResponse 构建退出群组响应
func (c *Converter) BuildLeaveGroupResponse(success bool, message string) *rest.LeaveGroupResponse {
	return &rest.LeaveGroupResponse{
		Success: success,
		Message: message,
	}
}

// BuildKickMemberResponse 构建踢出成员响应
func (c *Converter) BuildKickMemberResponse(success bool, message string) *rest.KickMemberResponse {
	return &rest.KickMemberResponse{
		Success: success,
		Message: message,
	}
}

// BuildInviteToGroupResponse 构建邀请加入群组响应
func (c *Converter) BuildInviteToGroupResponse(success bool, message string) *rest.InviteToGroupResponse {
	return &rest.InviteToGroupResponse{
		Success: success,
		Message: message,
	}
}

// BuildPublishAnnouncementResponse 构建发布群公告响应
func (c *Converter) BuildPublishAnnouncementResponse(success bool, message string) *rest.PublishAnnouncementResponse {
	return &rest.PublishAnnouncementResponse{
		Success: success,
		Message: message,
	}
}
