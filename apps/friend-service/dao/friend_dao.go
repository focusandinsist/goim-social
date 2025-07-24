package dao

import (
	"context"
	"fmt"

	"websocket-server/apps/friend-service/model"
	"websocket-server/pkg/database"
)

// friendDAO 好友数据访问对象
type friendDAO struct {
	db *database.PostgreSQL
}

// NewFriendDAO 创建好友DAO实例
func NewFriendDAO(db *database.PostgreSQL) FriendDAO {
	return &friendDAO{db: db}
}

// CreateFriend 创建好友关系
func (d *friendDAO) CreateFriend(ctx context.Context, friend *model.Friend) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(friend).Error; err != nil {
		return fmt.Errorf("failed to create friend: %v", err)
	}
	return nil
}

// DeleteFriend 删除好友关系
func (d *friendDAO) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ? AND friend_id = ?", userID, friendID).
		Delete(&model.Friend{}).Error; err != nil {
		return fmt.Errorf("failed to delete friend: %v", err)
	}
	return nil
}

// GetFriend 获取好友信息
func (d *friendDAO) GetFriend(ctx context.Context, userID, friendID int64) (*model.Friend, error) {
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
func (d *friendDAO) ListFriends(ctx context.Context, userID int64) ([]*model.Friend, error) {
	var friends []*model.Friend
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ?", userID).Find(&friends).Error; err != nil {
		return nil, fmt.Errorf("failed to list friends: %v", err)
	}
	return friends, nil
}

// IsFriend 检查是否为好友
func (d *friendDAO) IsFriend(ctx context.Context, userID, friendID int64) (bool, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.Friend{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check friend: %v", err)
	}
	return count > 0, nil
}

// UpdateFriendRemark 更新好友备注
func (d *friendDAO) UpdateFriendRemark(ctx context.Context, userID, friendID int64, remark string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.Friend{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("remark", remark).Error; err != nil {
		return fmt.Errorf("failed to update friend remark: %v", err)
	}
	return nil
}

// CreateFriendApply 创建好友申请
func (d *friendDAO) CreateFriendApply(ctx context.Context, apply *model.FriendApply) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(apply).Error; err != nil {
		return fmt.Errorf("failed to create friend apply: %v", err)
	}
	return nil
}

// UpdateFriendApplyStatus 更新好友申请状态
func (d *friendDAO) UpdateFriendApplyStatus(ctx context.Context, userID, applicantID int64, status string) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.FriendApply{}).
		Where("user_id = ? AND applicant_id = ?", userID, applicantID).
		Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update friend apply status: %v", err)
	}
	return nil
}

// GetFriendApply 获取好友申请
func (d *friendDAO) GetFriendApply(ctx context.Context, userID, applicantID int64) (*model.FriendApply, error) {
	var apply model.FriendApply
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ? AND applicant_id = ?", userID, applicantID).
		First(&apply).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("friend apply not found")
		}
		return nil, fmt.Errorf("failed to get friend apply: %v", err)
	}
	return &apply, nil
}

// ListFriendApply 获取好友申请列表
func (d *friendDAO) ListFriendApply(ctx context.Context, userID int64) ([]*model.FriendApply, error) {
	var applies []*model.FriendApply
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ?", userID).Find(&applies).Error; err != nil {
		return nil, fmt.Errorf("failed to list friend applies: %v", err)
	}
	return applies, nil
}

// DeleteFriendApply 删除好友申请
func (d *friendDAO) DeleteFriendApply(ctx context.Context, userID, applicantID int64) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("user_id = ? AND applicant_id = ?", userID, applicantID).
		Delete(&model.FriendApply{}).Error; err != nil {
		return fmt.Errorf("failed to delete friend apply: %v", err)
	}
	return nil
}
