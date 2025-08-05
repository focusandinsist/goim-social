package service

import (
	"context"
	"errors"
	"time"

	"goim-social/api/rest"
	"goim-social/apps/user-service/dao"
	"goim-social/apps/user-service/model"
	"goim-social/pkg/auth"
	"goim-social/pkg/kafka"
	"goim-social/pkg/redis"
)

// Service 用户服务
type Service struct {
	dao   dao.UserDAO
	redis *redis.RedisClient
	kafka *kafka.Producer
}

// NewService 创建用户服务
func NewService(userDAO dao.UserDAO, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		dao:   userDAO,
		redis: redis,
		kafka: kafka,
	}
}

// Register 用户注册
func (s *Service) Register(ctx context.Context, req *rest.RegisterRequest) (*rest.RegisterResponse, error) {
	// 检查用户名是否已存在
	exists, err := s.dao.CheckUsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("username already exists")
	}

	// 检查邮箱是否已存在
	emailExists, err := s.dao.CheckEmailExists(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, errors.New("email already exists")
	}

	// 创建用户
	user := &model.User{
		Username: req.Username,
		Password: req.Password, // TODO: 加密
		Email:    req.Email,
		Nickname: req.Nickname,
		Status:   0, // 正常状态
	}

	err = s.dao.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return &rest.RegisterResponse{
		User: &rest.UserInfo{
			Id:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		},
	}, nil
}

// Login 用户登录
func (s *Service) Login(ctx context.Context, req *rest.LoginRequest) (*rest.LoginResponse, error) {
	user, err := s.dao.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	// 验证密码 (简化处理，实际应该加密比较)
	if user.Password != req.Password {
		return nil, errors.New("invalid password")
	}

	// 生成 JWT token，带 device_id
	claims := map[string]interface{}{
		"user_id":   user.ID,
		"username":  user.Username,
		"device_id": req.DeviceId,
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	token, err := auth.GenerateJWT(claims)
	if err != nil {
		return nil, err
	}

	return &rest.LoginResponse{
		Token: token,
		User: &rest.UserInfo{
			Id:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		},
		ExpireAt: claims["exp"].(int64),
		DeviceId: req.DeviceId,
	}, nil
}

// GetUserByID 根据ID获取用户
func (s *Service) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	user, err := s.dao.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
