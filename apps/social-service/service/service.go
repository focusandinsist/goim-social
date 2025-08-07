package service

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/apps/social-service/dao"
	"goim-social/apps/social-service/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/telemetry"
)

// Service 社交服务
type Service struct {
	dao    dao.SocialDAO
	redis  *redis.RedisClient
	kafka  *kafka.Producer
	logger logger.Logger
}

// NewService 创建社交服务实例
func NewService(socialDAO dao.SocialDAO, redis *redis.RedisClient, kafka *kafka.Producer, log logger.Logger) *Service {
	return &Service{
		dao:    socialDAO,
		redis:  redis,
		kafka:  kafka,
		logger: log,
	}
}

// ============ 好友关系管理 ============

// SendFriendRequest 发送好友申请
func (s *Service) SendFriendRequest(ctx context.Context, applicantID, userID int64, remark string) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.SendFriendRequest")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("friend.applicant_id", applicantID),
		attribute.Int64("friend.user_id", userID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, applicantID)

	// 检查是否已经是好友
	isFriend, err := s.dao.IsFriend(ctx, applicantID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check friendship")
		return fmt.Errorf("检查好友关系失败: %v", err)
	}
	if isFriend {
		span.SetStatus(codes.Error, "already friends")
		return fmt.Errorf("已经是好友关系")
	}

	// 检查是否已有申请
	existingApply, err := s.dao.GetFriendApply(ctx, userID, applicantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check existing apply")
		return fmt.Errorf("检查申请记录失败: %v", err)
	}
	if existingApply != nil && existingApply.Status == model.FriendApplyStatusPending {
		span.SetStatus(codes.Error, "apply already exists")
		return fmt.Errorf("已有待处理的好友申请")
	}

	// 创建好友申请
	apply := &model.FriendApply{
		UserID:      userID,
		ApplicantID: applicantID,
		Remark:      remark,
		Status:      model.FriendApplyStatusPending,
	}

	if err := s.dao.CreateFriendApply(ctx, apply); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create friend apply")
		return fmt.Errorf("创建好友申请失败: %v", err)
	}

	s.logger.Info(ctx, "Friend request sent successfully",
		logger.F("applicantID", applicantID),
		logger.F("userID", userID),
		logger.F("applyID", apply.ID))

	span.SetStatus(codes.Ok, "friend request sent successfully")
	return nil
}

// AcceptFriendRequest 接受好友申请
func (s *Service) AcceptFriendRequest(ctx context.Context, userID, applicantID int64, remark string) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.AcceptFriendRequest")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("friend.user_id", userID),
		attribute.Int64("friend.applicant_id", applicantID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 获取申请记录
	apply, err := s.dao.GetFriendApply(ctx, userID, applicantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get friend apply")
		return fmt.Errorf("获取申请记录失败: %v", err)
	}
	if apply == nil {
		span.SetStatus(codes.Error, "apply not found")
		return fmt.Errorf("申请记录不存在")
	}
	if apply.Status != model.FriendApplyStatusPending {
		span.SetStatus(codes.Error, "apply not pending")
		return fmt.Errorf("申请状态不正确")
	}

	// 创建双向好友关系
	friend1 := &model.Friend{
		UserID:   userID,
		FriendID: applicantID,
		Remark:   remark,
	}
	friend2 := &model.Friend{
		UserID:   applicantID,
		FriendID: userID,
		Remark:   "",
	}

	if err := s.dao.CreateFriend(ctx, friend1); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create friend1")
		return fmt.Errorf("创建好友关系失败: %v", err)
	}

	if err := s.dao.CreateFriend(ctx, friend2); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create friend2")
		return fmt.Errorf("创建好友关系失败: %v", err)
	}

	// 更新申请状态
	if err := s.dao.UpdateFriendApplyStatus(ctx, userID, applicantID, model.FriendApplyStatusAccepted); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update apply status")
		return fmt.Errorf("更新申请状态失败: %v", err)
	}

	s.logger.Info(ctx, "Friend request accepted successfully",
		logger.F("userID", userID),
		logger.F("applicantID", applicantID))

	span.SetStatus(codes.Ok, "friend request accepted successfully")
	return nil
}

// RejectFriendRequest 拒绝好友申请
func (s *Service) RejectFriendRequest(ctx context.Context, userID, applicantID int64, reason string) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.RejectFriendRequest")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("friend.user_id", userID),
		attribute.Int64("friend.applicant_id", applicantID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 更新申请状态
	if err := s.dao.UpdateFriendApplyStatus(ctx, userID, applicantID, model.FriendApplyStatusRejected); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update apply status")
		return fmt.Errorf("更新申请状态失败: %v", err)
	}

	s.logger.Info(ctx, "Friend request rejected successfully",
		logger.F("userID", userID),
		logger.F("applicantID", applicantID),
		logger.F("reason", reason))

	span.SetStatus(codes.Ok, "friend request rejected successfully")
	return nil
}

// DeleteFriend 删除好友
func (s *Service) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.DeleteFriend")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("friend.user_id", userID),
		attribute.Int64("friend.friend_id", friendID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 删除双向好友关系
	if err := s.dao.DeleteFriend(ctx, userID, friendID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete friend")
		return fmt.Errorf("删除好友失败: %v", err)
	}

	s.logger.Info(ctx, "Friend deleted successfully",
		logger.F("userID", userID),
		logger.F("friendID", friendID))

	span.SetStatus(codes.Ok, "friend deleted successfully")
	return nil
}

// GetFriendList 获取好友列表
func (s *Service) GetFriendList(ctx context.Context, userID int64) ([]*model.Friend, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.GetFriendList")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("friend.user_id", userID))

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	friends, err := s.dao.ListFriends(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get friend list")
		return nil, fmt.Errorf("获取好友列表失败: %v", err)
	}

	span.SetAttributes(attribute.Int("friend.count", len(friends)))
	span.SetStatus(codes.Ok, "friend list retrieved successfully")
	return friends, nil
}

// GetFriendApplyList 获取好友申请列表
func (s *Service) GetFriendApplyList(ctx context.Context, userID int64) ([]*model.FriendApply, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.GetFriendApplyList")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("friend.user_id", userID))

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	applies, err := s.dao.ListFriendApply(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get friend apply list")
		return nil, fmt.Errorf("获取好友申请列表失败: %v", err)
	}

	span.SetAttributes(attribute.Int("friend.apply_count", len(applies)))
	span.SetStatus(codes.Ok, "friend apply list retrieved successfully")
	return applies, nil
}

// ============ 群组管理 ============

// CreateGroup 创建群组
func (s *Service) CreateGroup(ctx context.Context, ownerID int64, name, description, avatar string, isPublic bool, maxMembers int32, memberIDs []int64) (*model.Group, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.CreateGroup")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("group.owner_id", ownerID),
		attribute.String("group.name", name),
		attribute.Bool("group.is_public", isPublic),
		attribute.Int("group.initial_members", len(memberIDs)),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, ownerID)

	// 设置默认值
	if maxMembers <= 0 {
		maxMembers = model.DefaultMaxMembers
	}

	// 创建群组
	group := &model.Group{
		Name:         name,
		Description:  description,
		Avatar:       avatar,
		OwnerID:      ownerID,
		MemberCount:  1, // 群主
		MaxMembers:   maxMembers,
		IsPublic:     isPublic,
		Announcement: "",
	}

	if err := s.dao.CreateGroup(ctx, group); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create group")
		return nil, fmt.Errorf("创建群组失败: %v", err)
	}

	// 设置群组ID到span
	span.SetAttributes(attribute.Int64("group.id", group.ID))
	ctx = tracecontext.WithGroupID(ctx, group.ID)

	// 添加群主为成员
	owner := &model.GroupMember{
		UserID:   ownerID,
		GroupID:  group.ID,
		Role:     model.RoleOwner,
		Nickname: "",
	}

	if err := s.dao.AddMember(ctx, owner); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add owner as member")
		return nil, fmt.Errorf("添加群主失败: %v", err)
	}

	// 添加初始成员
	memberCount := int32(1) // 群主
	for _, memberID := range memberIDs {
		if memberID == ownerID {
			continue // 跳过群主
		}

		member := &model.GroupMember{
			UserID:   memberID,
			GroupID:  group.ID,
			Role:     model.RoleMember,
			Nickname: "",
		}

		if err := s.dao.AddMember(ctx, member); err != nil {
			s.logger.Error(ctx, "Failed to add initial member",
				logger.F("groupID", group.ID),
				logger.F("memberID", memberID),
				logger.F("error", err.Error()))
			continue
		}
		memberCount++
	}

	// 更新成员数量
	if err := s.dao.UpdateMemberCount(ctx, group.ID, memberCount); err != nil {
		s.logger.Error(ctx, "Failed to update member count",
			logger.F("groupID", group.ID),
			logger.F("count", memberCount),
			logger.F("error", err.Error()))
	}
	group.MemberCount = memberCount

	s.logger.Info(ctx, "Group created successfully",
		logger.F("groupID", group.ID),
		logger.F("ownerID", ownerID),
		logger.F("name", name),
		logger.F("memberCount", memberCount))

	span.SetStatus(codes.Ok, "group created successfully")
	return group, nil
}

// GetGroup 获取群组信息
func (s *Service) GetGroup(ctx context.Context, groupID int64) (*model.Group, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.GetGroup")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("group.id", groupID))

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, groupID)

	group, err := s.dao.GetGroup(ctx, groupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get group")
		return nil, fmt.Errorf("获取群组信息失败: %v", err)
	}

	span.SetAttributes(
		attribute.String("group.name", group.Name),
		attribute.Int64("group.owner_id", group.OwnerID),
		attribute.Int("group.member_count", int(group.MemberCount)),
	)

	span.SetStatus(codes.Ok, "group retrieved successfully")
	return group, nil
}

// UpdateGroup 更新群组信息
func (s *Service) UpdateGroup(ctx context.Context, groupID, operatorID int64, name, description, avatar, announcement string) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.UpdateGroup")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("group.id", groupID),
		attribute.Int64("group.operator_id", operatorID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, groupID)
	ctx = tracecontext.WithUserID(ctx, operatorID)

	// 检查权限
	member, err := s.dao.GetMember(ctx, groupID, operatorID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get member")
		return fmt.Errorf("获取成员信息失败: %v", err)
	}
	if member.Role != model.RoleOwner && member.Role != model.RoleAdmin {
		span.SetStatus(codes.Error, "insufficient permissions")
		return fmt.Errorf("权限不足")
	}

	// 获取群组信息
	group, err := s.dao.GetGroup(ctx, groupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get group")
		return fmt.Errorf("获取群组信息失败: %v", err)
	}

	// 更新字段
	if name != "" {
		group.Name = name
	}
	if description != "" {
		group.Description = description
	}
	if avatar != "" {
		group.Avatar = avatar
	}
	if announcement != "" {
		group.Announcement = announcement
	}
	group.UpdatedAt = time.Now()

	if err := s.dao.UpdateGroup(ctx, group); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update group")
		return fmt.Errorf("更新群组信息失败: %v", err)
	}

	s.logger.Info(ctx, "Group updated successfully",
		logger.F("groupID", groupID),
		logger.F("operatorID", operatorID))

	span.SetStatus(codes.Ok, "group updated successfully")
	return nil
}

// JoinGroup 加入群组
func (s *Service) JoinGroup(ctx context.Context, groupID, userID int64) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.JoinGroup")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("group.id", groupID),
		attribute.Int64("group.user_id", userID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, groupID)
	ctx = tracecontext.WithUserID(ctx, userID)

	// 检查是否已是成员
	isMember, err := s.dao.IsMember(ctx, groupID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check membership")
		return fmt.Errorf("检查成员关系失败: %v", err)
	}
	if isMember {
		span.SetStatus(codes.Error, "already member")
		return fmt.Errorf("已经是群成员")
	}

	// 检查群组是否存在
	group, err := s.dao.GetGroup(ctx, groupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get group")
		return fmt.Errorf("获取群组信息失败: %v", err)
	}

	// 检查群组是否已满
	if group.MemberCount >= group.MaxMembers {
		span.SetStatus(codes.Error, "group is full")
		return fmt.Errorf("群组已满")
	}

	// 添加成员
	member := &model.GroupMember{
		UserID:   userID,
		GroupID:  groupID,
		Role:     model.RoleMember,
		Nickname: "",
	}

	if err := s.dao.AddMember(ctx, member); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add member")
		return fmt.Errorf("添加成员失败: %v", err)
	}

	// 更新成员数量
	newCount := group.MemberCount + 1
	if err := s.dao.UpdateMemberCount(ctx, groupID, newCount); err != nil {
		s.logger.Error(ctx, "Failed to update member count",
			logger.F("groupID", groupID),
			logger.F("count", newCount),
			logger.F("error", err.Error()))
	}

	s.logger.Info(ctx, "User joined group successfully",
		logger.F("groupID", groupID),
		logger.F("userID", userID))

	span.SetStatus(codes.Ok, "user joined group successfully")
	return nil
}

// LeaveGroup 离开群组
func (s *Service) LeaveGroup(ctx context.Context, groupID, userID int64) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.LeaveGroup")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("group.id", groupID),
		attribute.Int64("group.user_id", userID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, groupID)
	ctx = tracecontext.WithUserID(ctx, userID)

	// 检查是否为群主
	member, err := s.dao.GetMember(ctx, groupID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get member")
		return fmt.Errorf("获取成员信息失败: %v", err)
	}
	if member.Role == model.RoleOwner {
		span.SetStatus(codes.Error, "owner cannot leave")
		return fmt.Errorf("群主不能离开群组")
	}

	// 移除成员
	if err := s.dao.RemoveMember(ctx, groupID, userID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to remove member")
		return fmt.Errorf("移除成员失败: %v", err)
	}

	// 更新成员数量
	group, err := s.dao.GetGroup(ctx, groupID)
	if err == nil {
		newCount := group.MemberCount - 1
		if err := s.dao.UpdateMemberCount(ctx, groupID, newCount); err != nil {
			s.logger.Error(ctx, "Failed to update member count",
				logger.F("groupID", groupID),
				logger.F("count", newCount),
				logger.F("error", err.Error()))
		}
	}

	s.logger.Info(ctx, "User left group successfully",
		logger.F("groupID", groupID),
		logger.F("userID", userID))

	span.SetStatus(codes.Ok, "user left group successfully")
	return nil
}

// GetGroupMembers 获取群成员列表
func (s *Service) GetGroupMembers(ctx context.Context, groupID int64) ([]*model.GroupMember, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.GetGroupMembers")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("group.id", groupID))

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, groupID)

	members, err := s.dao.GetGroupMembers(ctx, groupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get group members")
		return nil, fmt.Errorf("获取群成员列表失败: %v", err)
	}

	span.SetAttributes(attribute.Int("group.member_count", len(members)))
	span.SetStatus(codes.Ok, "group members retrieved successfully")
	return members, nil
}

// GetMemberIDs 获取群成员ID列表
func (s *Service) GetMemberIDs(ctx context.Context, groupID int64) ([]int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.GetMemberIDs")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("group.id", groupID))

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, groupID)

	memberIDs, err := s.dao.GetMemberIDs(ctx, groupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get member IDs")
		return nil, fmt.Errorf("获取群成员ID列表失败: %v", err)
	}

	span.SetAttributes(attribute.Int("group.member_count", len(memberIDs)))
	span.SetStatus(codes.Ok, "member IDs retrieved successfully")
	return memberIDs, nil
}

// ============ 统一社交关系查询接口 ============

// ValidateFriendship 验证好友关系
func (s *Service) ValidateFriendship(ctx context.Context, userID, friendID int64) (bool, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.ValidateFriendship")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("friend.user_id", userID),
		attribute.Int64("friend.friend_id", friendID),
	)

	isFriend, err := s.dao.ValidateFriendship(ctx, userID, friendID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate friendship")
		return false, fmt.Errorf("验证好友关系失败: %v", err)
	}

	span.SetAttributes(attribute.Bool("friend.is_friend", isFriend))
	span.SetStatus(codes.Ok, "friendship validated successfully")
	return isFriend, nil
}

// ValidateGroupMembership 验证群成员关系
func (s *Service) ValidateGroupMembership(ctx context.Context, userID, groupID int64) (bool, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.ValidateGroupMembership")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("group.user_id", userID),
		attribute.Int64("group.id", groupID),
	)

	isMember, err := s.dao.ValidateGroupMembership(ctx, userID, groupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate group membership")
		return false, fmt.Errorf("验证群成员关系失败: %v", err)
	}

	span.SetAttributes(attribute.Bool("group.is_member", isMember))
	span.SetStatus(codes.Ok, "group membership validated successfully")
	return isMember, nil
}

// GetUserSocialInfo 获取用户社交信息汇总
func (s *Service) GetUserSocialInfo(ctx context.Context, userID int64) (*dao.SocialInfo, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "social.service.GetUserSocialInfo")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("social.user_id", userID))

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	socialInfo, err := s.dao.GetUserSocialInfo(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user social info")
		return nil, fmt.Errorf("获取用户社交信息失败: %v", err)
	}

	span.SetAttributes(
		attribute.Int("social.friend_count", socialInfo.FriendCount),
		attribute.Int("social.group_count", socialInfo.GroupCount),
	)

	span.SetStatus(codes.Ok, "user social info retrieved successfully")
	return socialInfo, nil
}
