package dao

import (
	"context"
	"fmt"

	"goim-social/apps/social-service/model"
	"goim-social/pkg/database"
)

// socialDAO 社交数据访问对象
type socialDAO struct {
	db *database.PostgreSQL
}

// NewSocialDAO 创建社交DAO实例
func NewSocialDAO(db *database.PostgreSQL) SocialDAO {
	return &socialDAO{db: db}
}

// ============ 好友关系管理 ============

// CreateFriend 创建好友关系
func (d *socialDAO) CreateFriend(ctx context.Context, friend *model.Friend) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(friend).Error; err != nil {
		return fmt.Errorf("failed to create friend: %v", err)
	}
	return nil
}

// DeleteFriend 删除好友关系
func (d *socialDAO) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	db := d.db.GetDB()
	// 删除双向关系
	if err := db.WithContext(ctx).Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, friendID, friendID, userID).Delete(&model.Friend{}).Error; err != nil {
		return fmt.Errorf("failed to delete friend: %v", err)
	}
	return nil
}

// GetFriend 获取好友信息
func (d *socialDAO) GetFriend(ctx context.Context, userID, friendID int64) (*model.Friend, error) {
	var friend model.Friend
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ? AND friend_id = ?", userID, friendID).
		First(&friend).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("friend not found")
		}
		return nil, fmt.Errorf("failed to get friend: %v", err)
	}
	return &friend, nil
}

// ListFriends 获取好友列表
func (d *socialDAO) ListFriends(ctx context.Context, userID int64) ([]*model.Friend, error) {
	var friends []*model.Friend
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ?", userID).Find(&friends).Error; err != nil {
		return nil, fmt.Errorf("failed to list friends: %v", err)
	}
	return friends, nil
}

// IsFriend 检查是否为好友
func (d *socialDAO) IsFriend(ctx context.Context, userID, friendID int64) (bool, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.Friend{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check friend: %v", err)
	}
	return count > 0, nil
}

// UpdateFriendRemark 更新好友备注
func (d *socialDAO) UpdateFriendRemark(ctx context.Context, userID, friendID int64, remark string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.Friend{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("remark", remark).Error; err != nil {
		return fmt.Errorf("failed to update friend remark: %v", err)
	}
	return nil
}

// ============ 好友申请管理 ============

// CreateFriendApply 创建好友申请
func (d *socialDAO) CreateFriendApply(ctx context.Context, apply *model.FriendApply) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(apply).Error; err != nil {
		return fmt.Errorf("failed to create friend apply: %v", err)
	}
	return nil
}

// GetFriendApply 获取好友申请
func (d *socialDAO) GetFriendApply(ctx context.Context, userID, applicantID int64) (*model.FriendApply, error) {
	var apply model.FriendApply
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ? AND applicant_id = ?", userID, applicantID).
		First(&apply).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get friend apply: %v", err)
	}
	return &apply, nil
}

// ListFriendApply 获取好友申请列表
func (d *socialDAO) ListFriendApply(ctx context.Context, userID int64) ([]*model.FriendApply, error) {
	var applies []*model.FriendApply
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Find(&applies).Error; err != nil {
		return nil, fmt.Errorf("failed to list friend applies: %v", err)
	}
	return applies, nil
}

// UpdateFriendApplyStatus 更新好友申请状态
func (d *socialDAO) UpdateFriendApplyStatus(ctx context.Context, userID, applicantID int64, status string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.FriendApply{}).
		Where("user_id = ? AND applicant_id = ?", userID, applicantID).
		Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update friend apply status: %v", err)
	}
	return nil
}

// ============ 群组管理 ============

// CreateGroup 创建群组
func (d *socialDAO) CreateGroup(ctx context.Context, group *model.Group) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(group).Error; err != nil {
		return fmt.Errorf("failed to create group: %v", err)
	}
	return nil
}

// GetGroup 获取群组信息
func (d *socialDAO) GetGroup(ctx context.Context, groupID int64) (*model.Group, error) {
	var group model.Group
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("id = ?", groupID).First(&group).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to get group: %v", err)
	}
	return &group, nil
}

// UpdateGroup 更新群组信息
func (d *socialDAO) UpdateGroup(ctx context.Context, group *model.Group) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Save(group).Error; err != nil {
		return fmt.Errorf("failed to update group: %v", err)
	}
	return nil
}

// DeleteGroup 删除群组
func (d *socialDAO) DeleteGroup(ctx context.Context, groupID int64) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Delete(&model.Group{}, groupID).Error; err != nil {
		return fmt.Errorf("failed to delete group: %v", err)
	}
	return nil
}

// SearchGroups 搜索群组
func (d *socialDAO) SearchGroups(ctx context.Context, keyword string, isPublic bool, limit, offset int) ([]*model.Group, int64, error) {
	var groups []*model.Group
	var total int64

	db := d.db.GetDB()
	query := db.WithContext(ctx).Model(&model.Group{})

	if keyword != "" {
		query = query.Where("name ILIKE ?", "%"+keyword+"%")
	}
	if isPublic {
		query = query.Where("is_public = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count groups: %v", err)
	}

	if err := query.Limit(limit).Offset(offset).Find(&groups).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search groups: %v", err)
	}

	return groups, total, nil
}

// UpdateMemberCount 更新群成员数量
func (d *socialDAO) UpdateMemberCount(ctx context.Context, groupID int64, count int32) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.Group{}).
		Where("id = ?", groupID).Update("member_count", count).Error; err != nil {
		return fmt.Errorf("failed to update member count: %v", err)
	}
	return nil
}

// ============ 群成员管理 ============

// AddMember 添加群成员
func (d *socialDAO) AddMember(ctx context.Context, member *model.GroupMember) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(member).Error; err != nil {
		return fmt.Errorf("failed to add member: %v", err)
	}
	return nil
}

// RemoveMember 移除群成员
func (d *socialDAO) RemoveMember(ctx context.Context, groupID, userID int64) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.GroupMember{}).Error; err != nil {
		return fmt.Errorf("failed to remove member: %v", err)
	}
	return nil
}

// GetMember 获取群成员信息
func (d *socialDAO) GetMember(ctx context.Context, groupID, userID int64) (*model.GroupMember, error) {
	var member model.GroupMember
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("member not found")
		}
		return nil, fmt.Errorf("failed to get member: %v", err)
	}
	return &member, nil
}

// GetGroupMembers 获取群成员列表
func (d *socialDAO) GetGroupMembers(ctx context.Context, groupID int64) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to get group members: %v", err)
	}
	return members, nil
}

// GetMemberIDs 获取群成员ID列表
func (d *socialDAO) GetMemberIDs(ctx context.Context, groupID int64) ([]int64, error) {
	var memberIDs []int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupMember{}).
		Where("group_id = ?", groupID).Pluck("user_id", &memberIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get member IDs: %v", err)
	}
	return memberIDs, nil
}

// IsMember 检查是否为群成员
func (d *socialDAO) IsMember(ctx context.Context, groupID, userID int64) (bool, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check member: %v", err)
	}
	return count > 0, nil
}

// UpdateMemberRole 更新成员角色
func (d *socialDAO) UpdateMemberRole(ctx context.Context, groupID, userID int64, role string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("role", role).Error; err != nil {
		return fmt.Errorf("failed to update member role: %v", err)
	}
	return nil
}

// UpdateMemberNickname 更新成员昵称
func (d *socialDAO) UpdateMemberNickname(ctx context.Context, groupID, userID int64, nickname string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("nickname", nickname).Error; err != nil {
		return fmt.Errorf("failed to update member nickname: %v", err)
	}
	return nil
}

// GetUserGroups 获取用户加入的群组列表
func (d *socialDAO) GetUserGroups(ctx context.Context, userID int64) ([]*model.Group, error) {
	var groups []*model.Group
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Table("groups").
		Joins("JOIN group_members ON groups.id = group_members.group_id").
		Where("group_members.user_id = ?", userID).Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to get user groups: %v", err)
	}
	return groups, nil
}

// ============ 群邀请管理 ============

// CreateInvitation 创建群邀请
func (d *socialDAO) CreateInvitation(ctx context.Context, invitation *model.GroupInvitation) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(invitation).Error; err != nil {
		return fmt.Errorf("failed to create invitation: %v", err)
	}
	return nil
}

// GetInvitation 获取群邀请
func (d *socialDAO) GetInvitation(ctx context.Context, invitationID int64) (*model.GroupInvitation, error) {
	var invitation model.GroupInvitation
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("id = ?", invitationID).First(&invitation).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("invitation not found")
		}
		return nil, fmt.Errorf("failed to get invitation: %v", err)
	}
	return &invitation, nil
}

// ListInvitations 获取用户的群邀请列表
func (d *socialDAO) ListInvitations(ctx context.Context, userID int64, status string) ([]*model.GroupInvitation, error) {
	var invitations []*model.GroupInvitation
	db := d.db.GetDB()
	query := db.WithContext(ctx).Where("invitee_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at DESC").Find(&invitations).Error; err != nil {
		return nil, fmt.Errorf("failed to list invitations: %v", err)
	}
	return invitations, nil
}

// UpdateInvitationStatus 更新邀请状态
func (d *socialDAO) UpdateInvitationStatus(ctx context.Context, invitationID int64, status string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupInvitation{}).
		Where("id = ?", invitationID).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update invitation status: %v", err)
	}
	return nil
}

// ============ 加群申请管理 ============

// CreateJoinRequest 创建加群申请
func (d *socialDAO) CreateJoinRequest(ctx context.Context, request *model.GroupJoinRequest) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(request).Error; err != nil {
		return fmt.Errorf("failed to create join request: %v", err)
	}
	return nil
}

// GetJoinRequest 获取加群申请
func (d *socialDAO) GetJoinRequest(ctx context.Context, groupID, userID int64) (*model.GroupJoinRequest, error) {
	var request model.GroupJoinRequest
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&request).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get join request: %v", err)
	}
	return &request, nil
}

// ListJoinRequests 获取群的加群申请列表
func (d *socialDAO) ListJoinRequests(ctx context.Context, groupID int64, status string) ([]*model.GroupJoinRequest, error) {
	var requests []*model.GroupJoinRequest
	db := d.db.GetDB()
	query := db.WithContext(ctx).Where("group_id = ?", groupID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at DESC").Find(&requests).Error; err != nil {
		return nil, fmt.Errorf("failed to list join requests: %v", err)
	}
	return requests, nil
}

// UpdateJoinRequestStatus 更新加群申请状态
func (d *socialDAO) UpdateJoinRequestStatus(ctx context.Context, requestID int64, status string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupJoinRequest{}).
		Where("id = ?", requestID).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update join request status: %v", err)
	}
	return nil
}

// ============ 统一社交关系查询接口 ============

// ValidateFriendship 验证好友关系
func (d *socialDAO) ValidateFriendship(ctx context.Context, userID, friendID int64) (bool, error) {
	return d.IsFriend(ctx, userID, friendID)
}

// ValidateGroupMembership 验证群成员关系
func (d *socialDAO) ValidateGroupMembership(ctx context.Context, userID, groupID int64) (bool, error) {
	return d.IsMember(ctx, groupID, userID)
}

// GetUserSocialInfo 获取用户社交信息汇总
func (d *socialDAO) GetUserSocialInfo(ctx context.Context, userID int64) (*SocialInfo, error) {
	// 获取好友列表
	friends, err := d.ListFriends(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get friends: %v", err)
	}

	// 获取群组列表
	groups, err := d.GetUserGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %v", err)
	}

	// 提取ID列表
	friendIDs := make([]int64, len(friends))
	for i, friend := range friends {
		friendIDs[i] = friend.FriendID
	}

	groupIDs := make([]int64, len(groups))
	for i, group := range groups {
		groupIDs[i] = group.ID
	}

	return &SocialInfo{
		UserID:      userID,
		FriendCount: len(friends),
		GroupCount:  len(groups),
		FriendIDs:   friendIDs,
		GroupIDs:    groupIDs,
	}, nil
}
