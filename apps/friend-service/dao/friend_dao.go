package dao

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"websocket-server/apps/friend-service/model"
	"websocket-server/pkg/database"
)

// friendDAO 好友数据访问对象
type friendDAO struct {
	db *database.MongoDB
}

// NewFriendDAO 创建好友DAO实例
func NewFriendDAO(db *database.MongoDB) FriendDAO {
	return &friendDAO{db: db}
}

// CreateFriend 创建好友关系
func (d *friendDAO) CreateFriend(ctx context.Context, friend *model.Friend) error {
	collection := d.db.GetCollection("friends")
	_, err := collection.InsertOne(ctx, friend)
	if err != nil {
		return fmt.Errorf("failed to create friend: %v", err)
	}
	return nil
}

// DeleteFriend 删除好友关系
func (d *friendDAO) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	collection := d.db.GetCollection("friends")
	filter := bson.M{"user_id": userID, "friend_id": friendID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete friend: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("friend relation not found")
	}

	return nil
}

// GetFriend 获取好友信息
func (d *friendDAO) GetFriend(ctx context.Context, userID, friendID int64) (*model.Friend, error) {
	var friend model.Friend
	collection := d.db.GetCollection("friends")
	filter := bson.M{"user_id": userID, "friend_id": friendID}

	err := collection.FindOne(ctx, filter).Decode(&friend)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("friend not found")
		}
		return nil, fmt.Errorf("failed to get friend: %v", err)
	}

	return &friend, nil
}

// ListFriends 查询好友列表
func (d *friendDAO) ListFriends(ctx context.Context, userID int64) ([]*model.Friend, error) {
	var friends []*model.Friend
	collection := d.db.GetCollection("friends")
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetLimit(100) // 限制最多100个好友

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

// IsFriend 是否为好友关系
func (d *friendDAO) IsFriend(ctx context.Context, userID, friendID int64) (bool, error) {
	collection := d.db.GetCollection("friends")
	filter := bson.M{"user_id": userID, "friend_id": friendID}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check friend relation: %v", err)
	}

	return count > 0, nil
}

// UpdateFriendRemark 更新好友备注
func (d *friendDAO) UpdateFriendRemark(ctx context.Context, userID, friendID int64, remark string) error {
	collection := d.db.GetCollection("friends")
	filter := bson.M{"user_id": userID, "friend_id": friendID}
	update := bson.M{"$set": bson.M{"remark": remark}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update friend remark: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("friend relation not found")
	}

	return nil
}

// CreateFriendApply 创建好友申请
func (d *friendDAO) CreateFriendApply(ctx context.Context, apply *model.FriendApply) error {
	collection := d.db.GetCollection("friend_applies")
	_, err := collection.InsertOne(ctx, apply)
	if err != nil {
		return fmt.Errorf("failed to create friend apply: %v", err)
	}
	return nil
}

// GetFriendApply 获取好友申请
func (d *friendDAO) GetFriendApply(ctx context.Context, userID, applicantID int64) (*model.FriendApply, error) {
	var apply model.FriendApply
	collection := d.db.GetCollection("friend_applies")
	filter := bson.M{"user_id": userID, "applicant_id": applicantID}

	err := collection.FindOne(ctx, filter).Decode(&apply)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("friend apply not found")
		}
		return nil, fmt.Errorf("failed to get friend apply: %v", err)
	}

	return &apply, nil
}

// ListFriendApply 查询好友申请列表
func (d *friendDAO) ListFriendApply(ctx context.Context, userID int64) ([]*model.FriendApply, error) {
	var applies []*model.FriendApply
	collection := d.db.GetCollection("friend_applies")
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}) // 按时间倒序

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query friend applies: %v", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &applies); err != nil {
		return nil, fmt.Errorf("failed to decode friend applies: %v", err)
	}

	return applies, nil
}

// UpdateFriendApplyStatus 更新好友申请状态
func (d *friendDAO) UpdateFriendApplyStatus(ctx context.Context, userID, applicantID int64, status string) error {
	collection := d.db.GetCollection("friend_applies")
	filter := bson.M{"user_id": userID, "applicant_id": applicantID}
	update := bson.M{"$set": bson.M{"status": status}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update friend apply status: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("friend apply not found")
	}

	return nil
}

// DeleteFriendApply 删除好友申请
func (d *friendDAO) DeleteFriendApply(ctx context.Context, userID, applicantID int64) error {
	collection := d.db.GetCollection("friend_applies")
	filter := bson.M{"user_id": userID, "applicant_id": applicantID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete friend apply: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("friend apply not found")
	}

	return nil
}
