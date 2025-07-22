package service

import (
	"context"
	"websocket-server/api/rest"
	"websocket-server/apps/group/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"
)

type Service struct {
	db    *database.MongoDB
	redis *redis.RedisClient
	kafka *kafka.Producer
}

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:    db,
		redis: redis,
		kafka: kafka,
	}
}

// Message 群消息结构体
// 如有需要可迁移到 model 层

type Message struct {
	ID      int64
	Content string
}

// CreateGroup 创建群组
func (s *Service) CreateGroup(ctx context.Context, name, description string, ownerID int64, memberIDs []int64) (*model.Group, error) {
	// TODO: 实现数据库写入
	return &model.Group{
		ID:          1,
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		MemberIDs:   memberIDs,
	}, nil
}

// AddMembers 添加成员到群组
func (s *Service) AddMembers(ctx context.Context, groupID int64, userIDs []int64) error {
	// TODO: 实现添加成员逻辑
	return nil
}

// SendMessage 发送群消息
func (s *Service) SendMessage(ctx context.Context, groupID, senderID int64, content string, messageType int) (*Message, error) {
	// TODO: 实现消息发送逻辑
	return &Message{
		ID:      1,
		Content: content,
	}, nil
}

// GetGroup 获取群组信息
func (s *Service) GetGroup(ctx context.Context, groupID int64) (*model.Group, error) {
	// TODO: 实现获取群组信息逻辑
	return &model.Group{
		ID:          groupID,
		Name:        "示例群",
		Description: "desc",
		OwnerID:     1,
		MemberIDs:   []int64{1, 2, 3},
	}, nil
}

// DeleteGroup 删除群组
func (s *Service) DeleteGroup(ctx context.Context, groupID, userID int64) error {
	// TODO: 实现删除群组逻辑
	return nil
}

// GetGroupList 获取群组列表
func (s *Service) GetGroupList(ctx context.Context, userID int64, page, size int) ([]*model.Group, int, error) {
	// TODO: 实现获取群组列表逻辑
	return []*model.Group{}, 0, nil
}

// GetGroupInfo 获取群组详细信息
func (s *Service) GetGroupInfo(ctx context.Context, groupID, userID int64) (*model.Group, error) {
	// TODO: 实现获取群组详细信息逻辑
	return &model.Group{
		ID:          groupID,
		Name:        "示例群",
		Description: "desc",
		OwnerID:     1,
		MemberIDs:   []int64{1, 2, 3},
	}, nil
}

// 业务方法中可直接用 s.db, s.redis, s.kafka

// GRPCService gRPC服务实现
type GRPCService struct {
	rest.UnimplementedGroupServiceServer
	svc *Service
}

// NewGRPCService 构造函数
func (s *Service) NewGRPCService(svc *Service) *GRPCService {
	return &GRPCService{svc: svc}
}

// CreateGroup gRPC接口实现
func (g *GRPCService) CreateGroup(ctx context.Context, req *rest.CreateGroupRequest) (*rest.CreateGroupResponse, error) {
	group, err := g.svc.CreateGroup(ctx, req.Name, req.Description, req.OwnerId, req.MemberIds)
	if err != nil {
		return nil, err
	}

	return &rest.CreateGroupResponse{
		Group: &rest.GroupInfo{
			Id:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			OwnerId:     group.OwnerID,
			MemberIds:   group.MemberIDs,
		},
	}, nil
}

// AddMembers gRPC接口实现
func (g *GRPCService) AddMembers(ctx context.Context, req *rest.AddMembersRequest) (*rest.AddMembersResponse, error) {
	err := g.svc.AddMembers(ctx, req.GroupId, req.UserIds)
	return &rest.AddMembersResponse{Success: err == nil}, err
}

// GetGroup gRPC接口实现
func (g *GRPCService) GetGroup(ctx context.Context, req *rest.GetGroupRequest) (*rest.GetGroupResponse, error) {
	group, err := g.svc.GetGroup(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	return &rest.GetGroupResponse{
		Group: &rest.GroupInfo{
			Id:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			OwnerId:     group.OwnerID,
			MemberIds:   group.MemberIDs,
		},
	}, nil
}

// DeleteGroup gRPC接口实现
func (g *GRPCService) DeleteGroup(ctx context.Context, req *rest.DeleteGroupRequest) (*rest.DeleteGroupResponse, error) {
	err := g.svc.DeleteGroup(ctx, req.GroupId, req.UserId)
	return &rest.DeleteGroupResponse{Success: err == nil}, err
}

// GetGroupList gRPC接口实现
func (g *GRPCService) GetGroupList(ctx context.Context, req *rest.GetGroupListRequest) (*rest.GetGroupListResponse, error) {
	groups, total, err := g.svc.GetGroupList(ctx, req.UserId, int(req.Page), int(req.Size))
	if err != nil {
		return nil, err
	}

	var groupInfos []*rest.GroupInfo
	for _, group := range groups {
		groupInfos = append(groupInfos, &rest.GroupInfo{
			Id:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			OwnerId:     group.OwnerID,
			MemberIds:   group.MemberIDs,
		})
	}

	return &rest.GetGroupListResponse{
		Groups: groupInfos,
		Total:  int32(total),
	}, nil
}

// GetGroupInfo gRPC接口实现
func (g *GRPCService) GetGroupInfo(ctx context.Context, req *rest.GetGroupInfoRequest) (*rest.GetGroupInfoResponse, error) {
	group, err := g.svc.GetGroupInfo(ctx, req.GroupId, req.UserId)
	if err != nil {
		return nil, err
	}

	return &rest.GetGroupInfoResponse{
		Group: &rest.GroupInfo{
			Id:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			OwnerId:     group.OwnerID,
			MemberIds:   group.MemberIDs,
		},
	}, nil
}
