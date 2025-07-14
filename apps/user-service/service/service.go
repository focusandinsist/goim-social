package service

import (
	"context"
	"errors"
	"time"
	"websocket-server/api/rest"
	"websocket-server/apps/user-service/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
)

// Service 用户服务
type Service struct {
	db    *database.MongoDB
	redis *redis.RedisClient
	kafka *kafka.Producer
}

// NewService 创建用户服务
func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:    db,
		redis: redis,
		kafka: kafka,
	}
}

// Register 用户注册
func (s *Service) Register(ctx context.Context, req *rest.RegisterRequest) (*rest.RegisterResponse, error) {
	// 检查用户名是否已存在
	exists, err := s.db.GetCollection("users").CountDocuments(ctx, bson.M{"username": req.Username})
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("username already exists")
	}

	// 创建用户
	user := &model.User{
		Username: req.Username,
		Password: req.Password, // 实际应该加密
		Email:    req.Email,
		Nickname: req.Nickname,
		Status:   0, // 正常状态
	}

	_, err = s.db.GetCollection("users").InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	return &rest.RegisterResponse{
		User: &rest.UserResponse{
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
	var user model.User
	err := s.db.GetCollection("users").FindOne(ctx, bson.M{"username": req.Username}).Decode(&user)
	if err != nil {
		return nil, err
	}

	// 验证密码 (简化处理，实际应该加密比较)
	if user.Password != req.Password {
		return nil, errors.New("invalid password")
	}

	// 生成 JWT token，带 device_id
	claims := map[string]any{
		"user_id":   user.ID,
		"username":  user.Username,
		"device_id": req.DeviceId,
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	token, err := GenerateJWT(claims) // 需实现 GenerateJWT
	if err != nil {
		return nil, err
	}

	return &rest.LoginResponse{
		Token: token,
		User: &rest.UserResponse{
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

// GenerateJWT 生成带 device_id 的 JWT token
func GenerateJWT(claims map[string]any) (string, error) {
	jwtClaims := jwt.MapClaims{}
	for k, v := range claims {
		jwtClaims[k] = v
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	secret := "your-secret" // 建议放到配置文件
	return token.SignedString([]byte(secret))
}

// GetUserByID 根据ID获取用户
func (s *Service) GetUserByID(ctx context.Context, userID int64) (*rest.UserResponse, error) {
	var user model.User
	err := s.db.GetCollection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &rest.UserResponse{
		Id:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}, nil
}

// GRPCService gRPC服务实现
type GRPCService struct {
	rest.UnimplementedUserServiceServer
	svc *Service
}

// NewGRPCService 构造函数
func (s *Service) NewGRPCService(svc *Service) *GRPCService {
	return &GRPCService{svc: svc}
}

// Login gRPC接口实现
func (g *GRPCService) Login(ctx context.Context, req *rest.LoginRequest) (*rest.LoginResponse, error) {
	return g.svc.Login(ctx, req)
}

// Register gRPC接口实现
func (g *GRPCService) Register(ctx context.Context, req *rest.RegisterRequest) (*rest.RegisterResponse, error) {
	return g.svc.Register(ctx, req)
}

// GetUser gRPC接口实现
func (g *GRPCService) GetUser(ctx context.Context, req *rest.GetUserRequest) (*rest.GetUserResponse, error) {
	// 这里需要将string类型的user_id转换为int64
	// 简化处理，实际应该做更严格的转换
	userID := int64(1) // 临时处理

	user, err := g.svc.GetUserByID(ctx, userID)
	if err != nil {
		return &rest.GetUserResponse{
			Success: false,
			Message: err.Error(),
			User:    nil,
		}, nil
	}

	return &rest.GetUserResponse{
		Success: true,
		Message: "获取用户信息成功",
		User:    user,
	}, nil
}
