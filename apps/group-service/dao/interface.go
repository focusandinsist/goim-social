package dao

import (
	"context"

	"websocket-server/apps/group-service/model"
)

// GroupDAO 群组数据访问接口
type GroupDAO interface {
	// 群组管理
	CreateGroup(ctx context.Context, group *model.Group) error
	GetGroup(ctx context.Context, groupID int64) (*model.Group, error)
	UpdateGroup(ctx context.Context, group *model.Group) error
	DeleteGroup(ctx context.Context, groupID int64) error
	SearchGroups(ctx context.Context, keyword string, page, pageSize int32) ([]*model.Group, int64, error)
	GetUserGroups(ctx context.Context, userID int64, page, pageSize int32) ([]*model.Group, int64, error)

	// 群成员管理
	AddMember(ctx context.Context, member *model.GroupMember) error
	RemoveMember(ctx context.Context, groupID, userID int64) error
	GetGroupMembers(ctx context.Context, groupID int64) ([]*model.GroupMember, error)
	GetMember(ctx context.Context, groupID, userID int64) (*model.GroupMember, error)
	UpdateMemberRole(ctx context.Context, groupID, userID int64, role string) error
	IsMember(ctx context.Context, groupID, userID int64) (bool, error)
	GetMemberCount(ctx context.Context, groupID int64) (int32, error)
	UpdateMemberCount(ctx context.Context, groupID int64, count int32) error

	// 群邀请管理
	CreateInvitation(ctx context.Context, invitation *model.GroupInvitation) error
	GetInvitation(ctx context.Context, groupID, inviteeID int64) (*model.GroupInvitation, error)
	UpdateInvitationStatus(ctx context.Context, invitationID int64, status string) error
	GetUserInvitations(ctx context.Context, userID int64) ([]*model.GroupInvitation, error)

	// 加群申请管理
	CreateJoinRequest(ctx context.Context, request *model.GroupJoinRequest) error
	GetJoinRequest(ctx context.Context, groupID, userID int64) (*model.GroupJoinRequest, error)
	UpdateJoinRequestStatus(ctx context.Context, requestID int64, status string) error
	GetGroupJoinRequests(ctx context.Context, groupID int64) ([]*model.GroupJoinRequest, error)

	// 群公告管理
	CreateAnnouncement(ctx context.Context, announcement *model.GroupAnnouncement) error
	GetLatestAnnouncement(ctx context.Context, groupID int64) (*model.GroupAnnouncement, error)
	UpdateGroupAnnouncement(ctx context.Context, groupID int64, content string) error
}
