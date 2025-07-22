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

// ApplyFriend 申请加好友
func (s *Service) ApplyFriend(ctx context.Context, userID, friendID int64, remark string) error {
	if userID == friendID {
		return fmt.Errorf("不能添加自己为好友")
	}
	// 检查是否已是好友
	filter := bson.M{"user_id": userID, "friend_id": friendID}
	count, err := s.db.GetCollection("friends").CountDocuments(ctx, filter)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("已是好友")
	}
	// 检查是否已存在未处理的申请
	applyFilter := bson.M{"user_id": friendID, "applicant_id": userID, "status": "pending"}
	applyCount, err := s.db.GetCollection("friend_applies").CountDocuments(ctx, applyFilter)
	if err != nil {
		return err
	}
	if applyCount > 0 {
		return fmt.Errorf("已申请，等待对方处理")
	}
	// 插入申请
	apply := &model.FriendApply{
		UserID:      friendID, // 被申请人
		ApplicantID: userID,   // 申请人
		Remark:      remark,
		Status:      "pending",
		Timestamp:   time.Now().Unix(),
	}
	_, err = s.db.GetCollection("friend_applies").InsertOne(ctx, apply)
	return err
}

// RespondFriendApply 同意/拒绝好友申请
func (s *Service) RespondFriendApply(ctx context.Context, userID, applicantID int64, agree bool) error {
	filter := bson.M{"user_id": userID, "applicant_id": applicantID, "status": "pending"}
	update := bson.M{}
	if agree {
		update = bson.M{"$set": bson.M{"status": "accepted", "agree_time": time.Now().Unix()}}
	} else {
		update = bson.M{"$set": bson.M{"status": "rejected", "reject_time": time.Now().Unix()}}
	}
	result, err := s.db.GetCollection("friend_applies").UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("未找到待处理的好友申请")
	}
	if agree {
		// 双向加好友
		err1 := s.AddFriend(ctx, applicantID, userID, "")
		err2 := s.AddFriend(ctx, userID, applicantID, "")
		if err1 != nil || err2 != nil {
			return fmt.Errorf("添加好友失败: %v %v", err1, err2)
		}
	}
	return nil
}

// ListFriendApply 查询好友申请列表
func (s *Service) ListFriendApply(ctx context.Context, userID int64) ([]*model.FriendApply, error) {
	filter := bson.M{"user_id": userID}
	var applies []*model.FriendApply
	cursor, err := s.db.GetCollection("friend_applies").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err = cursor.All(ctx, &applies); err != nil {
		return nil, err
	}
	return applies, nil
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
