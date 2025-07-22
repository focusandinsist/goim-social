package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	friendpb "websocket-server/api/rest"
	"websocket-server/apps/friend/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"
)

type Service struct {
	db            *database.MongoDB
	redis         *redis.RedisClient
	kafka         *kafka.Producer
	messageClient friendpb.FriendEventServiceClient // 新增：gRPC客户端
}

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer, messageClient friendpb.FriendEventServiceClient) *Service {
	return &Service{
		db:            db,
		redis:         redis,
		kafka:         kafka,
		messageClient: messageClient,
	}
}

// AddFriend 添加好友
func (s *Service) AddFriend(ctx context.Context, userID, friendID int64, remark string) error {
	// 1. 先查是否已是好友
	filter := bson.M{"user_id": userID, "friend_id": friendID}
	count, err := s.db.GetCollection("friends").CountDocuments(ctx, filter)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // 已是好友，直接返回
	}

	// 2. 不是好友，先通过 gRPC 通知 message 微服务
	event := &friendpb.FriendEvent{
		Type:      friendpb.FriendEventType_ADD_FRIEND,
		UserId:    userID,
		FriendId:  friendID,
		Remark:    remark,
		Timestamp: time.Now().Unix(),
	}
	req := &friendpb.NotifyFriendEventRequest{Event: event}
	resp, err := s.messageClient.NotifyFriendEvent(ctx, req)
	if err != nil || !resp.GetSuccess() {
		return fmt.Errorf("notify message service failed: %v", err)
	}

	// 3. 正常加好友逻辑
	friend := &model.Friend{
		UserID:    userID,
		FriendID:  friendID,
		Remark:    remark,
		CreatedAt: time.Now().Unix(),
	}
	_, err = s.db.GetCollection("friends").InsertOne(ctx, friend)
	return err
}

// DeleteFriend 删除好友
func (s *Service) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	// 用map也行，用bson也行，bson省略异步map=>bson的转化
	// _, err := s.db.GetCollection("friends").DeleteOne(ctx, map[string]interface{}{"user_id": userID, "friend_id": friendID})
	filter := bson.M{"user_id": userID, "friend_id": friendID}
	_, err := s.db.GetCollection("friends").DeleteOne(ctx, filter)
	return err
}

// ListFriends 查询好友列表
func (s *Service) ListFriends(ctx context.Context, userID int64) ([]*model.Friend, error) {
	var friends []*model.Friend
	filter := bson.M{"user_id": userID}
	opts := options.FindOptions{}
	opts.SetLimit(50) // 上限50个好友
	cursor, err := s.db.GetCollection("friends").Find(ctx, filter)
	if err != nil {
		err = cursor.All(ctx, &friends)
	}
	return friends, err
}

// gRPC服务端实现
type GRPCService struct {
	friendpb.UnimplementedFriendEventServiceServer
	svc *Service
}

// NewGRPCService 创建gRPC服务
func (s *Service) NewGRPCService(svc *Service) *GRPCService {
	return &GRPCService{svc: svc}
}

// NotifyFriendEvent 处理好友事件通知
func (g *GRPCService) NotifyFriendEvent(ctx context.Context, req *friendpb.NotifyFriendEventRequest) (*friendpb.NotifyFriendEventResponse, error) {
	event := req.GetEvent()
	if event == nil {
		return &friendpb.NotifyFriendEventResponse{
			Success: false,
			Message: "event is nil",
		}, nil
	}

	// 根据事件类型处理
	switch event.Type {
	case friendpb.FriendEventType_ADD_FRIEND:
		err := g.svc.AddFriend(ctx, event.UserId, event.FriendId, event.Remark)
		if err != nil {
			return &friendpb.NotifyFriendEventResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}
	case friendpb.FriendEventType_DELETE_FRIEND:
		err := g.svc.DeleteFriend(ctx, event.UserId, event.FriendId)
		if err != nil {
			return &friendpb.NotifyFriendEventResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}
	default:
		return &friendpb.NotifyFriendEventResponse{
			Success: false,
			Message: "unknown event type",
		}, nil
	}

	return &friendpb.NotifyFriendEventResponse{
		Success: true,
		Message: "ok",
	}, nil
}
