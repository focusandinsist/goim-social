package dao

import (
	"context"
	"fmt"

	"websocket-server/apps/group-service/model"
	"websocket-server/pkg/database"
)

// groupDAO .
type groupDAO struct {
	db *database.PostgreSQL
}

// NewGroupDAO 创建群组DAO
func NewGroupDAO(db *database.PostgreSQL) GroupDAO {
	return &groupDAO{db: db}
}

// CreateGroup 创建群组
func (d *groupDAO) CreateGroup(ctx context.Context, group *model.Group) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(group).Error; err != nil {
		return fmt.Errorf("failed to create group: %v", err)
	}
	return nil
}

// GetGroup 获取群组信息
func (d *groupDAO) GetGroup(ctx context.Context, groupID int64) (*model.Group, error) {
	var group model.Group
	db := d.db.GetDB()
	if err := db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to get group: %v", err)
	}
	return &group, nil
}

// UpdateGroup 更新群组信息
func (d *groupDAO) UpdateGroup(ctx context.Context, group *model.Group) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Save(group).Error; err != nil {
		return fmt.Errorf("failed to update group: %v", err)
	}
	return nil
}

// DeleteGroup 删除群组
func (d *groupDAO) DeleteGroup(ctx context.Context, groupID int64) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Delete(&model.Group{}, groupID).Error; err != nil {
		return fmt.Errorf("failed to delete group: %v", err)
	}
	return nil
}

// SearchGroups 搜索群组
func (d *groupDAO) SearchGroups(ctx context.Context, keyword string, page, pageSize int32) ([]*model.Group, int64, error) {
	var groups []*model.Group
	var total int64

	db := d.db.GetDB()
	query := db.WithContext(ctx).Model(&model.Group{}).Where("is_public = ?", true)
	if keyword != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count groups: %v", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Find(&groups).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search groups: %v", err)
	}

	return groups, total, nil
}

// GetUserGroups 获取用户加入的群组列表
func (d *groupDAO) GetUserGroups(ctx context.Context, userID int64, page, pageSize int32) ([]*model.Group, int64, error) {
	var groups []*model.Group
	var total int64

	// 通过群成员表关联查询
	db := d.db.GetDB()
	query := db.WithContext(ctx).Model(&model.Group{}).
		Joins("JOIN group_members ON groups.id = group_members.group_id").
		Where("group_members.user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count user groups: %v", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Find(&groups).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get user groups: %v", err)
	}

	return groups, total, nil
}

// AddMember 添加群成员
func (d *groupDAO) AddMember(ctx context.Context, member *model.GroupMember) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(member).Error; err != nil {
		return fmt.Errorf("failed to add member: %v", err)
	}
	return nil
}

// RemoveMember 移除群成员
func (d *groupDAO) RemoveMember(ctx context.Context, groupID, userID int64) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.GroupMember{}).Error; err != nil {
		return fmt.Errorf("failed to remove member: %v", err)
	}
	return nil
}

// GetGroupMembers 获取群成员列表
func (d *groupDAO) GetGroupMembers(ctx context.Context, groupID int64) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to get group members: %v", err)
	}
	return members, nil
}

// GetMember 获取群成员信息
func (d *groupDAO) GetMember(ctx context.Context, groupID, userID int64) (*model.GroupMember, error) {
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

// UpdateMemberRole 更新成员角色
func (d *groupDAO) UpdateMemberRole(ctx context.Context, groupID, userID int64, role string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("role", role).Error; err != nil {
		return fmt.Errorf("failed to update member role: %v", err)
	}
	return nil
}

// IsMember 检查是否为群成员
func (d *groupDAO) IsMember(ctx context.Context, groupID, userID int64) (bool, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check membership: %v", err)
	}
	return count > 0, nil
}

// GetMemberCount 获取群成员数量
func (d *groupDAO) GetMemberCount(ctx context.Context, groupID int64) (int32, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupMember{}).
		Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get member count: %v", err)
	}
	return int32(count), nil
}

// UpdateMemberCount 更新群成员数量
func (d *groupDAO) UpdateMemberCount(ctx context.Context, groupID int64, count int32) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.Group{}).
		Where("id = ?", groupID).Update("member_count", count).Error; err != nil {
		return fmt.Errorf("failed to update member count: %v", err)
	}
	return nil
}

// CreateInvitation 创建群邀请
func (d *groupDAO) CreateInvitation(ctx context.Context, invitation *model.GroupInvitation) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(invitation).Error; err != nil {
		return fmt.Errorf("failed to create invitation: %v", err)
	}
	return nil
}

// GetInvitation 获取群邀请
func (d *groupDAO) GetInvitation(ctx context.Context, groupID, inviteeID int64) (*model.GroupInvitation, error) {
	var invitation model.GroupInvitation
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ? AND invitee_id = ? AND status = ?",
		groupID, inviteeID, model.InvitationStatusPending).First(&invitation).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("invitation not found")
		}
		return nil, fmt.Errorf("failed to get invitation: %v", err)
	}
	return &invitation, nil
}

// UpdateInvitationStatus 更新邀请状态
func (d *groupDAO) UpdateInvitationStatus(ctx context.Context, invitationID int64, status string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupInvitation{}).
		Where("id = ?", invitationID).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update invitation status: %v", err)
	}
	return nil
}

// GetUserInvitations 获取用户的群邀请列表
func (d *groupDAO) GetUserInvitations(ctx context.Context, userID int64) ([]*model.GroupInvitation, error) {
	var invitations []*model.GroupInvitation
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("invitee_id = ? AND status = ?",
		userID, model.InvitationStatusPending).Find(&invitations).Error; err != nil {
		return nil, fmt.Errorf("failed to get user invitations: %v", err)
	}
	return invitations, nil
}

// CreateJoinRequest 创建加群申请
func (d *groupDAO) CreateJoinRequest(ctx context.Context, request *model.GroupJoinRequest) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(request).Error; err != nil {
		return fmt.Errorf("failed to create join request: %v", err)
	}
	return nil
}

// GetJoinRequest 获取加群申请
func (d *groupDAO) GetJoinRequest(ctx context.Context, groupID, userID int64) (*model.GroupJoinRequest, error) {
	var request model.GroupJoinRequest
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ? AND user_id = ? AND status = ?",
		groupID, userID, model.JoinRequestStatusPending).First(&request).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("join request not found")
		}
		return nil, fmt.Errorf("failed to get join request: %v", err)
	}
	return &request, nil
}

// UpdateJoinRequestStatus 更新加群申请状态
func (d *groupDAO) UpdateJoinRequestStatus(ctx context.Context, requestID int64, status string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.GroupJoinRequest{}).
		Where("id = ?", requestID).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update join request status: %v", err)
	}
	return nil
}

// GetGroupJoinRequests 获取群的加群申请列表
func (d *groupDAO) GetGroupJoinRequests(ctx context.Context, groupID int64) ([]*model.GroupJoinRequest, error) {
	var requests []*model.GroupJoinRequest
	db := d.db.GetDB()
	err := db.WithContext(ctx).
		Where("group_id = ? AND status = ?", groupID, model.JoinRequestStatusPending).
		Find(&requests).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get group join requests: %v", err)
	}
	return requests, nil
}

// CreateAnnouncement 创建群公告
func (d *groupDAO) CreateAnnouncement(ctx context.Context, announcement *model.GroupAnnouncement) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(announcement).Error; err != nil {
		return fmt.Errorf("failed to create announcement: %v", err)
	}
	return nil
}

// GetLatestAnnouncement 获取最新群公告
func (d *groupDAO) GetLatestAnnouncement(ctx context.Context, groupID int64) (*model.GroupAnnouncement, error) {
	var announcement model.GroupAnnouncement
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("group_id = ?", groupID).
		Order("created_at DESC").First(&announcement).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("announcement not found")
		}
		return nil, fmt.Errorf("failed to get latest announcement: %v", err)
	}
	return &announcement, nil
}

// UpdateGroupAnnouncement 更新群公告
func (d *groupDAO) UpdateGroupAnnouncement(ctx context.Context, groupID int64, content string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.Group{}).
		Where("id = ?", groupID).Update("announcement", content).Error; err != nil {
		return fmt.Errorf("failed to update group announcement: %v", err)
	}
	return nil
}
