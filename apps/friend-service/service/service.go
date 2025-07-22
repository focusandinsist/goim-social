package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	friendpb "websocket-server/api/rest"
	"websocket-server/apps/friend-service/model"
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
	// 1. 先通知Message服务
	if s.messageClient != nil {
		event := &friendpb.FriendEvent{
			Type:      friendpb.FriendEventType_DELETE_FRIEND,
			UserId:    userID,
			FriendId:  friendID,
			Timestamp: time.Now().Unix(),
		}
		req := &friendpb.NotifyFriendEventRequest{Event: event}
		resp, err := s.messageClient.NotifyFriendEvent(ctx, req)
		if err != nil || !resp.GetSuccess() {
			return fmt.Errorf("notify message service failed: %v", err)
		}
	}

	// 2. 删除好友关系（双向删除）
	// 用map和bson都可以，bson省略一步map=>bson的转化
	filter1 := bson.M{"user_id": userID, "friend_id": friendID}
	filter2 := bson.M{"user_id": friendID, "friend_id": userID}

	collection := s.db.GetCollection("friends")
	_, err1 := collection.DeleteOne(ctx, filter1)
	_, err2 := collection.DeleteOne(ctx, filter2)

	if err1 != nil {
		return fmt.Errorf("failed to delete friend relation 1: %v", err1)
	}
	if err2 != nil {
		return fmt.Errorf("failed to delete friend relation 2: %v", err2)
	}

	return nil
}

// UpdateFriendRemark 更新好友备注
func (s *Service) UpdateFriendRemark(ctx context.Context, userID, friendID int64, newRemark string) error {
	filter := bson.M{"user_id": userID, "friend_id": friendID}
	update := bson.M{"$set": bson.M{"remark": newRemark}}

	collection := s.db.GetCollection("friends")
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update friend remark: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("friend relation not found")
	}

	return nil
}

// GetFriend 获取单个好友信息
func (s *Service) GetFriend(ctx context.Context, userID, friendID int64) (*model.Friend, error) {
	var friend model.Friend
	filter := bson.M{"user_id": userID, "friend_id": friendID}

	collection := s.db.GetCollection("friends")
	err := collection.FindOne(ctx, filter).Decode(&friend)
	if err != nil {
		return nil, fmt.Errorf("friend not found: %v", err)
	}

	return &friend, nil
}

// ListFriends 查询好友列表
func (s *Service) ListFriends(ctx context.Context, userID int64) ([]*model.Friend, error) {
	var friends []*model.Friend
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetLimit(100) // 上限100个好友

	collection := s.db.GetCollection("friends")
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query friends: %v", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &friends); err != nil {
		return nil, fmt.Errorf("failed to decode friends: %v", err)
	}

	return friends, nil
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
