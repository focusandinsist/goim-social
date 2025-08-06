package service

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/api/rest"
	"goim-social/apps/user-service/dao"
	"goim-social/apps/user-service/model"
	"goim-social/pkg/auth"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/telemetry"
)

// Service 用户服务
type Service struct {
	dao    dao.UserDAO
	redis  *redis.RedisClient
	kafka  *kafka.Producer
	logger logger.Logger
}

// NewService 创建用户服务
func NewService(userDAO dao.UserDAO, redis *redis.RedisClient, kafka *kafka.Producer, log logger.Logger) *Service {
	return &Service{
		dao:    userDAO,
		redis:  redis,
		kafka:  kafka,
		logger: log,
	}
}

// Register 用户注册
func (s *Service) Register(ctx context.Context, req *rest.RegisterRequest) (*rest.RegisterResponse, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "user.service.Register")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.String("user.username", req.Username),
		attribute.String("user.email", req.Email),
		attribute.String("user.nickname", req.Nickname),
	)

	// 检查用户名是否已存在
	exists, err := s.dao.CheckUsernameExists(ctx, req.Username)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check username exists")
		return nil, err
	}
	if exists {
		span.SetStatus(codes.Error, "username already exists")
		return nil, errors.New("username already exists")
	}

	// 检查邮箱是否已存在
	emailExists, err := s.dao.CheckEmailExists(ctx, req.Email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check email exists")
		return nil, err
	}
	if emailExists {
		span.SetStatus(codes.Error, "email already exists")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create user")
		return nil, err
	}

	// 将用户ID添加到context和span
	ctx = tracecontext.WithUserID(ctx, user.ID)
	span.SetAttributes(attribute.Int64("user.id", user.ID))

	s.logger.Info(ctx, "User registered successfully",
		logger.F("userID", user.ID),
		logger.F("username", req.Username),
		logger.F("email", req.Email))

	span.SetStatus(codes.Ok, "user registered successfully")
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
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "user.service.Login")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.String("user.username", req.Username),
		attribute.String("user.device_id", req.DeviceId),
	)

	user, err := s.dao.GetUserByUsername(ctx, req.Username)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user not found")
		return nil, err
	}

	// 将用户ID添加到context和span
	ctx = tracecontext.WithUserID(ctx, user.ID)
	span.SetAttributes(attribute.Int64("user.id", user.ID))

	// 验证密码 (简化处理，实际应该加密比较)
	if user.Password != req.Password {
		span.SetStatus(codes.Error, "invalid password")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to generate JWT token")
		return nil, err
	}

	s.logger.Info(ctx, "User login successful",
		logger.F("userID", user.ID),
		logger.F("username", req.Username),
		logger.F("deviceID", req.DeviceId))

	span.SetStatus(codes.Ok, "user login successful")
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
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "user.service.GetUserByID")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("user.id", userID))

	// 将用户ID添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	user, err := s.dao.GetUser(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user not found")
		return nil, err
	}

	span.SetAttributes(
		attribute.String("user.username", user.Username),
		attribute.String("user.email", user.Email),
	)

	s.logger.Info(ctx, "Get user by ID successful",
		logger.F("userID", userID),
		logger.F("username", user.Username))

	span.SetStatus(codes.Ok, "user retrieved successfully")
	return user, nil
}
