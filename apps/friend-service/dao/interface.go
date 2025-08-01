package dao

import (
	"context"

	"goim-social/apps/friend-service/model"
)

// FriendDAO 好友数据访问接口
type FriendDAO interface {
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
	DeleteFriendApply(ctx context.Context, userID, applicantID int64) error
}
