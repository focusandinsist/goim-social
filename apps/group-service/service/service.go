package service

import (
	"context"
	"fmt"
	"time"

	"websocket-server/apps/group-service/dao"
	"websocket-server/apps/group-service/model"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/redis"
)

// Service 群组服务
type Service struct {
	dao    dao.GroupDAO
	redis  *redis.RedisClient
	kafka  *kafka.Producer
	logger logger.Logger
}

// NewService 创建群组服务实例
func NewService(groupDAO dao.GroupDAO, redis *redis.RedisClient, kafka *kafka.Producer, log logger.Logger) *Service {
	return &Service{
		dao:    groupDAO,
		redis:  redis,
		kafka:  kafka,
		logger: log,
	}
}

// CreateGroup 创建群组
func (s *Service) CreateGroup(ctx context.Context, name, description, avatar string, ownerID int64, isPublic bool, maxMembers int32, memberIDs []int64) (*model.Group, error) {
	if name == "" {
		return nil, fmt.Errorf("群组名称不能为空")
	}
	if ownerID <= 0 {
		return nil, fmt.Errorf("群主ID无效")
	}
	if maxMembers <= 0 {
		maxMembers = model.DefaultMaxMembers
	}

	// 创建群组
	group := &model.Group{
		Name:         name,
		Description:  description,
		Avatar:       avatar,
		OwnerID:      ownerID,
		MemberCount:  1, // 群主自己
		MaxMembers:   maxMembers,
		IsPublic:     isPublic,
		Announcement: "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.dao.CreateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("创建群组失败: %v", err)
	}

	// 添加群主为成员
	ownerMember := &model.GroupMember{
		GroupID:  group.ID,
		UserID:   ownerID,
		Role:     model.RoleOwner,
		Nickname: "",
		JoinedAt: time.Now(),
	}
	if err := s.dao.AddMember(ctx, ownerMember); err != nil {
		return nil, fmt.Errorf("添加群主失败: %v", err)
	}

	// 添加初始成员
	memberCount := int32(1) // 群主
	for _, memberID := range memberIDs {
		if memberID == ownerID {
			continue
		}
		member := &model.GroupMember{
			GroupID:  group.ID,
			UserID:   memberID,
			Role:     model.RoleMember,
			Nickname: "",
			JoinedAt: time.Now(),
		}
		if err := s.dao.AddMember(ctx, member); err != nil {
			// 记录错误但不中断流程
			s.logger.Error(ctx, "Failed to add initial member to group",
				logger.F("groupID", group.ID),
				logger.F("memberID", memberID),
				logger.F("error", err.Error()))
			continue
		}
		memberCount++
	}

	// 更新成员数量
	group.MemberCount = memberCount
	if err := s.dao.UpdateMemberCount(ctx, group.ID, memberCount); err != nil {
		// 记录错误但不影响返回结果
		s.logger.Error(ctx, "Failed to update member count after group creation",
			logger.F("groupID", group.ID),
			logger.F("memberCount", memberCount),
			logger.F("error", err.Error()))
	}

	return group, nil
}

// GetGroupInfo 获取群组信息
func (s *Service) GetGroupInfo(ctx context.Context, groupID, userID int64) (*model.Group, []*model.GroupMember, error) {
	isMember, err := s.dao.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("检查成员身份失败: %v", err)
	}

	group, err := s.dao.GetGroup(ctx, groupID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取群组信息失败: %v", err)
	}

	// 如果是私有群且用户不是成员，只返回基本信息
	if !group.IsPublic && !isMember {
		return &model.Group{
			ID:          group.ID,
			Name:        group.Name,
			Avatar:      group.Avatar,
			MemberCount: group.MemberCount,
			IsPublic:    group.IsPublic,
		}, nil, nil
	}

	// 获取群成员列表
	members, err := s.dao.GetGroupMembers(ctx, groupID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取群成员失败: %v", err)
	}

	return group, members, nil
}

// SearchGroups 搜索群组
func (s *Service) SearchGroups(ctx context.Context, keyword string, page, pageSize int32) ([]*model.Group, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}

	return s.dao.SearchGroups(ctx, keyword, page, pageSize)
}

// GetUserGroups 获取用户群组列表
func (s *Service) GetUserGroups(ctx context.Context, userID int64, page, pageSize int32) ([]*model.Group, int64, error) {
	if userID <= 0 {
		return nil, 0, fmt.Errorf("用户ID无效")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}

	return s.dao.GetUserGroups(ctx, userID, page, pageSize)
}

// DisbandGroup 解散群组
func (s *Service) DisbandGroup(ctx context.Context, groupID, userID int64) error {
	member, err := s.dao.GetMember(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("获取成员信息失败: %v", err)
	}
	if member == nil || member.Role != model.RoleOwner {
		return fmt.Errorf("只有群主可以解散群组")
	}

	// 删除群组（级联删除成员、邀请、申请等）
	return s.dao.DeleteGroup(ctx, groupID)
}

// JoinGroup 加入群组
func (s *Service) JoinGroup(ctx context.Context, groupID, userID int64, reason string) error {
	group, err := s.dao.GetGroup(ctx, groupID)
	if err != nil {
		return fmt.Errorf("群组不存在: %v", err)
	}

	isMember, err := s.dao.IsMember(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("检查成员身份失败: %v", err)
	}
	if isMember {
		return fmt.Errorf("已是群成员")
	}

	if group.MemberCount >= group.MaxMembers {
		return fmt.Errorf("群组已满")
	}

	// 如果是公开群，直接加入
	if group.IsPublic {
		member := &model.GroupMember{
			GroupID:  groupID,
			UserID:   userID,
			Role:     model.RoleMember,
			Nickname: "",
			JoinedAt: time.Now(),
		}
		if err := s.dao.AddMember(ctx, member); err != nil {
			return fmt.Errorf("加入群组失败: %v", err)
		}

		// 更新成员数量
		newCount := group.MemberCount + 1
		if err := s.dao.UpdateMemberCount(ctx, groupID, newCount); err != nil {
			// 日志记录错误，但不影响结果
			s.logger.Error(ctx, "Failed to update member count after joining group",
				logger.F("groupID", groupID),
				logger.F("newCount", newCount),
				logger.F("error", err.Error()))
		}

		return nil
	}

	// 私有群需要申请
	return s.createJoinRequest(ctx, groupID, userID, reason)
}

// createJoinRequest 创建加群申请
func (s *Service) createJoinRequest(ctx context.Context, groupID, userID int64, reason string) error {
	existingRequest, err := s.dao.GetJoinRequest(ctx, groupID, userID)
	if err == nil && existingRequest != nil && existingRequest.Status == model.JoinRequestStatusPending {
		return fmt.Errorf("已有待处理的申请")
	}

	request := &model.GroupJoinRequest{
		GroupID:   groupID,
		UserID:    userID,
		Reason:    reason,
		Status:    model.JoinRequestStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.dao.CreateJoinRequest(ctx, request)
}

// LeaveGroup 退出群组
func (s *Service) LeaveGroup(ctx context.Context, groupID, userID int64) error {
	member, err := s.dao.GetMember(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("获取成员信息失败: %v", err)
	}
	if member == nil {
		return fmt.Errorf("不是群成员")
	}

	if member.Role == model.RoleOwner {
		return fmt.Errorf("群主不能退出群组，请解散群组")
	}

	if err := s.dao.RemoveMember(ctx, groupID, userID); err != nil {
		return fmt.Errorf("退出群组失败: %v", err)
	}

	// 更新成员数量
	count, err := s.dao.GetMemberCount(ctx, groupID)
	if err == nil {
		if updateErr := s.dao.UpdateMemberCount(ctx, groupID, count); updateErr != nil {
			s.logger.Error(ctx, "Failed to update member count after leaving group",
				logger.F("groupID", groupID),
				logger.F("count", count),
				logger.F("error", updateErr.Error()))
		}
	} else {
		s.logger.Error(ctx, "Failed to get member count after leaving group",
			logger.F("groupID", groupID),
			logger.F("error", err.Error()))
	}

	return nil
}

// KickMember 踢出成员
func (s *Service) KickMember(ctx context.Context, groupID, operatorID, targetUserID int64) error {
	operator, err := s.dao.GetMember(ctx, groupID, operatorID)
	if err != nil {
		return fmt.Errorf("获取操作者信息失败: %v", err)
	}
	if operator == nil || (operator.Role != model.RoleOwner && operator.Role != model.RoleAdmin) {
		return fmt.Errorf("权限不足")
	}

	target, err := s.dao.GetMember(ctx, groupID, targetUserID)
	if err != nil {
		return fmt.Errorf("获取目标用户信息失败: %v", err)
	}
	if target == nil {
		return fmt.Errorf("目标用户不是群成员")
	}

	if target.Role == model.RoleOwner {
		return fmt.Errorf("不能踢出群主")
	}

	if operator.Role == model.RoleAdmin && target.Role == model.RoleAdmin {
		return fmt.Errorf("只有群主可以踢出管理员")
	}

	if err := s.dao.RemoveMember(ctx, groupID, targetUserID); err != nil {
		return fmt.Errorf("踢出成员失败: %v", err)
	}

	// 更新成员数量
	count, err := s.dao.GetMemberCount(ctx, groupID)
	if err == nil {
		if updateErr := s.dao.UpdateMemberCount(ctx, groupID, count); updateErr != nil {
			s.logger.Error(ctx, "Failed to update member count after kicking member",
				logger.F("groupID", groupID),
				logger.F("targetUserID", targetUserID),
				logger.F("count", count),
				logger.F("error", updateErr.Error()))
		}
	} else {
		s.logger.Error(ctx, "Failed to get member count after kicking member",
			logger.F("groupID", groupID),
			logger.F("targetUserID", targetUserID),
			logger.F("error", err.Error()))
	}

	return nil
}

// InviteToGroup 邀请加入群组
func (s *Service) InviteToGroup(ctx context.Context, groupID, inviterID, inviteeID int64, message string) error {
	inviter, err := s.dao.GetMember(ctx, groupID, inviterID)
	if err != nil {
		return fmt.Errorf("获取邀请者信息失败: %v", err)
	}
	if inviter == nil {
		return fmt.Errorf("邀请者不是群成员")
	}

	isMember, err := s.dao.IsMember(ctx, groupID, inviteeID)
	if err != nil {
		return fmt.Errorf("检查被邀请者身份失败: %v", err)
	}
	if isMember {
		return fmt.Errorf("被邀请者已是群成员")
	}

	existingInvitation, err := s.dao.GetInvitation(ctx, groupID, inviteeID)
	if err == nil && existingInvitation != nil && existingInvitation.Status == model.InvitationStatusPending {
		return fmt.Errorf("已有待处理的邀请")
	}

	// 创建邀请
	invitation := &model.GroupInvitation{
		GroupID:   groupID,
		InviterID: inviterID,
		InviteeID: inviteeID,
		Message:   message,
		Status:    model.InvitationStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.dao.CreateInvitation(ctx, invitation)
}

// PublishAnnouncement 发布群公告
func (s *Service) PublishAnnouncement(ctx context.Context, groupID, userID int64, content string) error {
	member, err := s.dao.GetMember(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("获取成员信息失败: %v", err)
	}
	if member == nil || (member.Role != model.RoleOwner && member.Role != model.RoleAdmin) {
		return fmt.Errorf("权限不足")
	}

	// 更新群公告
	group := &model.Group{
		ID:           groupID,
		Announcement: content,
		UpdatedAt:    time.Now(),
	}

	return s.dao.UpdateGroup(ctx, group)
}

// GetGroupMemberIDs 获取群组成员ID列表（用于群消息推送）
func (s *Service) GetGroupMemberIDs(ctx context.Context, groupID int64) ([]int64, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("群组ID无效")
	}

	// 获取群成员列表
	members, err := s.dao.GetGroupMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("获取群成员失败: %v", err)
	}

	// 提取成员ID列表
	var memberIDs []int64
	for _, member := range members {
		memberIDs = append(memberIDs, member.UserID)
	}

	s.logger.Info(ctx, "获取群成员ID列表成功",
		logger.F("groupID", groupID),
		logger.F("memberCount", len(memberIDs)),
		logger.F("memberIDs", memberIDs))

	return memberIDs, nil
}

// ValidateGroupMember 验证用户是否为群成员（用于群消息发送权限验证）
func (s *Service) ValidateGroupMember(ctx context.Context, groupID, userID int64) error {
	if groupID <= 0 {
		return fmt.Errorf("群组ID无效")
	}
	if userID <= 0 {
		return fmt.Errorf("用户ID无效")
	}

	// 检查群组是否存在
	_, err := s.dao.GetGroup(ctx, groupID)
	if err != nil {
		return fmt.Errorf("群组不存在: %v", err)
	}

	// 检查用户是否是群成员
	isMember, err := s.dao.IsMember(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("检查成员身份失败: %v", err)
	}
	if !isMember {
		return fmt.Errorf("用户不是群成员")
	}

	return nil
}
