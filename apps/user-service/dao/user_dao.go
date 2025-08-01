package dao

import (
	"context"
	"fmt"

	"goim-social/apps/user-service/model"
	"goim-social/pkg/database"
)

// userDAO 用户数据访问对象
type userDAO struct {
	db *database.PostgreSQL
}

// NewUserDAO 创建用户DAO实例
func NewUserDAO(db *database.PostgreSQL) UserDAO {
	return &userDAO{db: db}
}

// CreateUser 创建用户
func (d *userDAO) CreateUser(ctx context.Context, user *model.User) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}
	return nil
}

// GetUser 根据ID获取用户
func (d *userDAO) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (d *userDAO) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %v", err)
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (d *userDAO) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %v", err)
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func (d *userDAO) UpdateUser(ctx context.Context, user *model.User) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}
	return nil
}

// DeleteUser 删除用户
func (d *userDAO) DeleteUser(ctx context.Context, userID int64) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("id = ?", userID).Delete(&model.User{}).Error; err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}
	return nil
}

// SearchUsers 搜索用户
func (d *userDAO) SearchUsers(ctx context.Context, keyword string, page, pageSize int32) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	db := d.db.GetDB()
	query := db.WithContext(ctx).Model(&model.User{}).
		Where("username ILIKE ? OR nickname ILIKE ? OR email ILIKE ?", 
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %v", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %v", err)
	}

	return users, total, nil
}

// ListUsers 获取用户列表
func (d *userDAO) ListUsers(ctx context.Context, page, pageSize int32) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	db := d.db.GetDB()
	query := db.WithContext(ctx).Model(&model.User{})

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %v", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %v", err)
	}

	return users, total, nil
}

// GetUsersByIDs 根据ID列表获取用户
func (d *userDAO) GetUsersByIDs(ctx context.Context, userIDs []int64) ([]*model.User, error) {
	var users []*model.User
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to get users by IDs: %v", err)
	}
	return users, nil
}

// UpdateUserStatus 更新用户状态
func (d *userDAO) UpdateUserStatus(ctx context.Context, userID int64, status int) error {
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update user status: %v", err)
	}
	return nil
}

// GetActiveUsers 获取活跃用户列表
func (d *userDAO) GetActiveUsers(ctx context.Context, page, pageSize int32) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	db := d.db.GetDB()
	query := db.WithContext(ctx).Model(&model.User{}).Where("status = ?", 0) // 0表示正常状态

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count active users: %v", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get active users: %v", err)
	}

	return users, total, nil
}

// GetUserCount 获取用户总数
func (d *userDAO) GetUserCount(ctx context.Context) (int64, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get user count: %v", err)
	}
	return count, nil
}

// GetUserCountByStatus 根据状态获取用户数量
func (d *userDAO) GetUserCountByStatus(ctx context.Context, status int) (int64, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.User{}).
		Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get user count by status: %v", err)
	}
	return count, nil
}

// CheckUsernameExists 检查用户名是否存在
func (d *userDAO) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.User{}).
		Where("username = ?", username).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check username exists: %v", err)
	}
	return count > 0, nil
}

// CheckEmailExists 检查邮箱是否存在
func (d *userDAO) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	db := d.db.GetDB()
	if err := db.WithContext(ctx).Model(&model.User{}).
		Where("email = ?", email).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check email exists: %v", err)
	}
	return count > 0, nil
}
