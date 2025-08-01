package dao

import (
	"context"

	"goim-social/apps/user-service/model"
)

// UserDAO 用户数据访问接口
type UserDAO interface {
	// 用户管理
	CreateUser(ctx context.Context, user *model.User) error
	GetUser(ctx context.Context, userID int64) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, userID int64) error
	
	// 用户查询
	SearchUsers(ctx context.Context, keyword string, page, pageSize int32) ([]*model.User, int64, error)
	ListUsers(ctx context.Context, page, pageSize int32) ([]*model.User, int64, error)
	GetUsersByIDs(ctx context.Context, userIDs []int64) ([]*model.User, error)
	
	// 用户状态管理
	UpdateUserStatus(ctx context.Context, userID int64, status int) error
	GetActiveUsers(ctx context.Context, page, pageSize int32) ([]*model.User, int64, error)
	
	// 用户统计
	GetUserCount(ctx context.Context) (int64, error)
	GetUserCountByStatus(ctx context.Context, status int) (int64, error)
	
	// 用户验证
	CheckUsernameExists(ctx context.Context, username string) (bool, error)
	CheckEmailExists(ctx context.Context, email string) (bool, error)
}
