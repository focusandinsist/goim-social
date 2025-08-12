package dao

import (
	"context"

	"goim-social/apps/social-service/internal/model"
)

// SocialDAO 社交数据访问接口
type SocialDAO interface {
	// 好友关系管理
	CreateFriend(ctx context.Context, friend *model.Friend) error
	DeleteFriend(ctx context.Context, userID, friendID int64) error
	GetFriend(ctx context.Context, userID, friendID int64) (*model.Friend, error)
	ListFriends(ctx context.Context, userID int64) ([]*model.Friend, error)
	IsFriend(ctx context.Context, userID, friendID int64) (bool, error)
	UpdateFriendRemark(ctx context.Context, userID, friendID int64, remark string) error

	// 好友申请管理
	CreateFriendApply(ctx context.Context, apply *model.FriendApply) error
	GetFriendApply(ctx context.Context, userID, applicantID int64) (*model.FriendApply, error)
	ListFriendApply(ctx context.Context, userID int64) ([]*model.FriendApply, error)
	UpdateFriendApplyStatus(ctx context.Context, userID, applicantID int64, status string) error

	// 群组管理
	CreateGroup(ctx context.Context, group *model.Group) error
	GetGroup(ctx context.Context, groupID int64) (*model.Group, error)
	UpdateGroup(ctx context.Context, group *model.Group) error
	DeleteGroup(ctx context.Context, groupID int64) error
	SearchGroups(ctx context.Context, keyword string, isPublic bool, limit, offset int) ([]*model.Group, int64, error)
	UpdateMemberCount(ctx context.Context, groupID int64, count int32) error

	// 群成员管理
	AddMember(ctx context.Context, member *model.GroupMember) error
	RemoveMember(ctx context.Context, groupID, userID int64) error
	GetMember(ctx context.Context, groupID, userID int64) (*model.GroupMember, error)
	GetGroupMembers(ctx context.Context, groupID int64) ([]*model.GroupMember, error)
	GetMemberIDs(ctx context.Context, groupID int64) ([]int64, error)
	IsMember(ctx context.Context, groupID, userID int64) (bool, error)
	UpdateMemberRole(ctx context.Context, groupID, userID int64, role string) error
	UpdateMemberNickname(ctx context.Context, groupID, userID int64, nickname string) error
	GetUserGroups(ctx context.Context, userID int64) ([]*model.Group, error)

	// 群邀请管理
	CreateInvitation(ctx context.Context, invitation *model.GroupInvitation) error
	GetInvitation(ctx context.Context, invitationID int64) (*model.GroupInvitation, error)
	ListInvitations(ctx context.Context, userID int64, status string) ([]*model.GroupInvitation, error)
	UpdateInvitationStatus(ctx context.Context, invitationID int64, status string) error

	// 加群申请管理
	CreateJoinRequest(ctx context.Context, request *model.GroupJoinRequest) error
	GetJoinRequest(ctx context.Context, groupID, userID int64) (*model.GroupJoinRequest, error)
	ListJoinRequests(ctx context.Context, groupID int64, status string) ([]*model.GroupJoinRequest, error)
	UpdateJoinRequestStatus(ctx context.Context, requestID int64, status string) error

	// 统一社交关系查询接口
	ValidateFriendship(ctx context.Context, userID, friendID int64) (bool, error)
	ValidateGroupMembership(ctx context.Context, userID, groupID int64) (bool, error)
	GetUserSocialInfo(ctx context.Context, userID int64) (*SocialInfo, error)
}

// SocialInfo 用户社交信息汇总
type SocialInfo struct {
	UserID      int64   `json:"user_id"`
	FriendCount int     `json:"friend_count"`
	GroupCount  int     `json:"group_count"`
	FriendIDs   []int64 `json:"friend_ids"`
	GroupIDs    []int64 `json:"group_ids"`
}
