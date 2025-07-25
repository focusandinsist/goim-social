package service

import (
	"context"
	"fmt"
	"time"

	friendpb "websocket-server/api/rest"
	"websocket-server/apps/friend-service/dao"
	"websocket-server/apps/friend-service/model"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"
)

type Service struct {
	dao           dao.FriendDAO
	redis         *redis.RedisClient
	kafka         *kafka.Producer
	messageClient friendpb.FriendEventServiceClient
}

func NewService(friendDAO dao.FriendDAO, redis *redis.RedisClient, kafka *kafka.Producer, messageClient friendpb.FriendEventServiceClient) *Service {
	return &Service{
		dao:           friendDAO,
		redis:         redis,
		kafka:         kafka,
		messageClient: messageClient,
	}
}

// AddFriend 添加好友
func (s *Service) AddFriend(ctx context.Context, userID, friendID int64, remark string) error {
	isFriend, err := s.dao.IsFriend(ctx, userID, friendID)
	if err != nil {
		return fmt.Errorf("failed to check friend relation: %v", err)
	}
	if isFriend {
		return fmt.Errorf("already friends")
	}

	// 不是好友，先通过 gRPC 通知 message 微服务
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

	// 双向添加好友关系
	// TODO: 事务
	friend := &model.Friend{
		UserID:   userID,
		FriendID: friendID,
		Remark:   remark,
	}
	if err := s.dao.CreateFriend(ctx, friend); err != nil {
		return fmt.Errorf("failed to create friend relation: %v", err)
	}

	friendReverse := &model.Friend{
		UserID:   friendID,
		FriendID: userID,
		Remark:   "", // 反向关系默认无备注
	}
	if err := s.dao.CreateFriend(ctx, friendReverse); err != nil {
		return fmt.Errorf("failed to create reverse friend relation: %v", err)
	}

	return nil
}

// DeleteFriend 删除好友
func (s *Service) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	// 先通知Message服务
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

	// TODO:事务 删除好友关系（双向删除）
	if err := s.dao.DeleteFriend(ctx, userID, friendID); err != nil {
		return fmt.Errorf("failed to delete friend relation: %v", err)
	}

	if err := s.dao.DeleteFriend(ctx, friendID, userID); err != nil {
		return fmt.Errorf("failed to delete reverse friend relation: %v", err)
	}

	return nil
}

// UpdateFriendRemark 更新好友备注
func (s *Service) UpdateFriendRemark(ctx context.Context, userID, friendID int64, newRemark string) error {
	return s.dao.UpdateFriendRemark(ctx, userID, friendID, newRemark)
}

// GetFriend 获取单个好友信息
func (s *Service) GetFriend(ctx context.Context, userID, friendID int64) (*model.Friend, error) {
	return s.dao.GetFriend(ctx, userID, friendID)
}

// ListFriends 查询好友列表
func (s *Service) ListFriends(ctx context.Context, userID int64) ([]*model.Friend, error) {
	return s.dao.ListFriends(ctx, userID)
}

// ApplyFriend 申请加好友
func (s *Service) ApplyFriend(ctx context.Context, userID, friendID int64, remark string) error {
	if userID == friendID {
		return fmt.Errorf("不能添加自己为好友")
	}

	// 检查是否已是好友
	isFriend, err := s.dao.IsFriend(ctx, userID, friendID)
	if err != nil {
		return fmt.Errorf("failed to check friend relation: %v", err)
	}
	if isFriend {
		return fmt.Errorf("已是好友")
	}

	// 检查是否已存在未处理的申请
	existingApply, err := s.dao.GetFriendApply(ctx, friendID, userID)
	if err == nil && existingApply != nil && existingApply.Status == "pending" {
		return fmt.Errorf("已申请，等待对方处理")
	}

	apply := &model.FriendApply{
		UserID:      friendID, // 被申请人
		ApplicantID: userID,   // 申请人
		Remark:      remark,
		Status:      "pending",
	}
	return s.dao.CreateFriendApply(ctx, apply)
}

// RespondFriendApply 同意/拒绝好友申请
func (s *Service) RespondFriendApply(ctx context.Context, userID, applicantID int64, agree bool) error {
	status := "rejected"
	if agree {
		status = "accepted"
	}

	if err := s.dao.UpdateFriendApplyStatus(ctx, userID, applicantID, status); err != nil {
		return fmt.Errorf("failed to update friend apply status: %v", err)
	}

	// 同意则创建双向好友关系
	if agree {
		friend1 := &model.Friend{
			UserID:   userID,
			FriendID: applicantID,
			Remark:   "",
		}
		friend2 := &model.Friend{
			UserID:   applicantID,
			FriendID: userID,
			Remark:   "",
		}

		if err := s.dao.CreateFriend(ctx, friend1); err != nil {
			return fmt.Errorf("failed to create friend relation 1: %v", err)
		}
		if err := s.dao.CreateFriend(ctx, friend2); err != nil {
			return fmt.Errorf("failed to create friend relation 2: %v", err)
		}
	}

	return nil
}

// ListFriendApply 查询好友申请列表
func (s *Service) ListFriendApply(ctx context.Context, userID int64) ([]*model.FriendApply, error) {
	return s.dao.ListFriendApply(ctx, userID)
}
