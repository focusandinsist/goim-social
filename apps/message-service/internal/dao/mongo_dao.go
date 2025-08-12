package dao

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goim-social/apps/message-service/internal/model"
)

type mongoDAO struct {
	db *mongo.Database
}

// NewMongoDAO 创建MongoDB DAO实例
func NewMongoDAO(db *mongo.Database) MessageDAO {
	return &mongoDAO{
		db: db,
	}
}

// ==================== 消息相关方法 ====================

// SaveMessage 保存消息
func (d *mongoDAO) SaveMessage(ctx context.Context, message *model.Message) error {
	collection := d.db.Collection("messages")
	_, err := collection.InsertOne(ctx, message)
	return err
}

// GetMessage 获取消息
func (d *mongoDAO) GetMessage(ctx context.Context, messageID int64) (*model.Message, error) {
	collection := d.db.Collection("messages")
	var message model.Message
	err := collection.FindOne(ctx, bson.M{"message_id": messageID}).Decode(&message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetMessageHistory 获取消息历史
func (d *mongoDAO) GetMessageHistory(ctx context.Context, userID, targetID int64, isGroup bool, limit int32, offset int32) ([]*model.HistoryMessage, error) {
	collection := d.db.Collection("messages")
	
	var filter bson.M
	if isGroup {
		filter = bson.M{"group_id": targetID}
	} else {
		filter = bson.M{
			"$or": []bson.M{
				{"from": userID, "to": targetID},
				{"from": targetID, "to": userID},
			},
		}
	}
	
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))
	
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var messages []*model.HistoryMessage
	for cursor.Next(ctx) {
		var msg model.Message
		if err := cursor.Decode(&msg); err != nil {
			continue
		}
		
		historyMsg := &model.HistoryMessage{
			ID:        msg.MessageID,
			From:      msg.From,
			To:        msg.To,
			GroupID:   msg.GroupID,
			Content:   msg.Content,
			MsgType:   int32(msg.MessageType),
			AckID:     msg.AckID,
			CreatedAt: msg.CreatedAt,
			Status:    0, // 默认状态
		}
		messages = append(messages, historyMsg)
	}
	
	return messages, nil
}

// UpdateMessageStatus 更新消息状态
func (d *mongoDAO) UpdateMessageStatus(ctx context.Context, messageID int64, status string) error {
	collection := d.db.Collection("messages")
	filter := bson.M{"message_id": messageID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// DeleteMessage 删除消息
func (d *mongoDAO) DeleteMessage(ctx context.Context, messageID int64) error {
	collection := d.db.Collection("messages")
	_, err := collection.DeleteOne(ctx, bson.M{"message_id": messageID})
	return err
}

// ==================== 历史记录相关方法 ====================

// RecordUserAction 记录用户行为
func (d *mongoDAO) RecordUserAction(ctx context.Context, record *model.HistoryRecord) error {
	collection := d.db.Collection("history_records")
	record.CreatedAt = time.Now()
	_, err := collection.InsertOne(ctx, record)
	return err
}

// BatchRecordUserAction 批量记录用户行为
func (d *mongoDAO) BatchRecordUserAction(ctx context.Context, records []*model.HistoryRecord) error {
	if len(records) == 0 {
		return nil
	}
	
	collection := d.db.Collection("history_records")
	docs := make([]interface{}, len(records))
	for i, record := range records {
		record.CreatedAt = time.Now()
		docs[i] = record
	}
	
	_, err := collection.InsertMany(ctx, docs)
	return err
}

// GetUserHistory 获取用户历史记录
func (d *mongoDAO) GetUserHistory(ctx context.Context, userID int64, actionType, objectType string, startTime, endTime time.Time, page, pageSize int32) ([]*model.HistoryRecord, int64, error) {
	collection := d.db.Collection("history_records")
	
	// 构建查询条件
	filter := bson.M{"user_id": userID}
	if actionType != "" {
		filter["action_type"] = actionType
	}
	if objectType != "" {
		filter["object_type"] = objectType
	}
	if !startTime.IsZero() || !endTime.IsZero() {
		timeFilter := bson.M{}
		if !startTime.IsZero() {
			timeFilter["$gte"] = startTime
		}
		if !endTime.IsZero() {
			timeFilter["$lte"] = endTime
		}
		filter["created_at"] = timeFilter
	}
	
	// 获取总数
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(pageSize)).
		SetSkip(int64((page - 1) * pageSize))
	
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	
	var records []*model.HistoryRecord
	for cursor.Next(ctx) {
		var record model.HistoryRecord
		if err := cursor.Decode(&record); err != nil {
			continue
		}
		records = append(records, &record)
	}
	
	return records, total, nil
}

// DeleteUserHistory 删除用户历史记录
func (d *mongoDAO) DeleteUserHistory(ctx context.Context, userID int64, recordIDs []string) (int64, error) {
	collection := d.db.Collection("history_records")
	
	objectIDs := make([]primitive.ObjectID, 0, len(recordIDs))
	for _, idStr := range recordIDs {
		if objectID, err := primitive.ObjectIDFromHex(idStr); err == nil {
			objectIDs = append(objectIDs, objectID)
		}
	}
	
	if len(objectIDs) == 0 {
		return 0, fmt.Errorf("no valid record IDs provided")
	}
	
	filter := bson.M{
		"user_id": userID,
		"_id":     bson.M{"$in": objectIDs},
	}
	
	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	
	return result.DeletedCount, nil
}

// ==================== 统计相关方法 ====================

// GetUserActionStats 获取用户行为统计
func (d *mongoDAO) GetUserActionStats(ctx context.Context, userID int64, actionType string, startTime, endTime time.Time, groupBy string) ([]*model.ActionStatItem, error) {
	collection := d.db.Collection("history_records")
	
	// 构建聚合管道
	pipeline := []bson.M{
		{"$match": bson.M{
			"user_id":    userID,
			"created_at": bson.M{"$gte": startTime, "$lte": endTime},
		}},
	}
	
	if actionType != "" {
		pipeline[0]["$match"].(bson.M)["action_type"] = actionType
	}
	
	// 根据groupBy添加分组逻辑
	var dateFormat string
	switch groupBy {
	case model.GroupByDay:
		dateFormat = "%Y-%m-%d"
	case model.GroupByWeek:
		dateFormat = "%Y-%U"
	case model.GroupByMonth:
		dateFormat = "%Y-%m"
	default:
		dateFormat = "%Y-%m-%d"
	}
	
	pipeline = append(pipeline, bson.M{
		"$group": bson.M{
			"_id": bson.M{
				"date":        bson.M{"$dateToString": bson.M{"format": dateFormat, "date": "$created_at"}},
				"action_type": "$action_type",
			},
			"count": bson.M{"$sum": 1},
		},
	})
	
	pipeline = append(pipeline, bson.M{
		"$sort": bson.M{"_id.date": 1},
	})
	
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var stats []*model.ActionStatItem
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date       string `bson:"date"`
				ActionType string `bson:"action_type"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}
		
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		
		stats = append(stats, &model.ActionStatItem{
			Date:       result.ID.Date,
			ActionType: result.ID.ActionType,
			Count:      result.Count,
		})
	}
	
	return stats, nil
}

// UpdateUserActionStats 更新用户行为统计
func (d *mongoDAO) UpdateUserActionStats(ctx context.Context, userID int64, actionType string) error {
	collection := d.db.Collection("user_action_stats")
	
	filter := bson.M{
		"user_id":     userID,
		"action_type": actionType,
	}
	
	update := bson.M{
		"$inc": bson.M{
			"total_count": 1,
			"today_count": 1,
			"week_count":  1,
			"month_count": 1,
		},
		"$set": bson.M{
			"last_action_time": time.Now(),
			"updated_at":       time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetObjectHotStats 获取对象热度统计
func (d *mongoDAO) GetObjectHotStats(ctx context.Context, objectType string, objectID int64) (*model.ObjectHotStats, error) {
	collection := d.db.Collection("object_hot_stats")
	
	var stats model.ObjectHotStats
	err := collection.FindOne(ctx, bson.M{
		"object_type": objectType,
		"object_id":   objectID,
	}).Decode(&stats)
	
	if err != nil {
		return nil, err
	}
	
	return &stats, nil
}

// UpdateObjectHotStats 更新对象热度统计
func (d *mongoDAO) UpdateObjectHotStats(ctx context.Context, objectType string, objectID int64, actionType string, delta int64) error {
	collection := d.db.Collection("object_hot_stats")
	
	filter := bson.M{
		"object_type": objectType,
		"object_id":   objectID,
	}
	
	updateField := ""
	switch actionType {
	case model.ActionTypeView:
		updateField = "view_count"
	case model.ActionTypeLike:
		updateField = "like_count"
	case model.ActionTypeFavorite:
		updateField = "favorite_count"
	case model.ActionTypeShare:
		updateField = "share_count"
	case model.ActionTypeComment:
		updateField = "comment_count"
	default:
		return fmt.Errorf("unsupported action type: %s", actionType)
	}
	
	update := bson.M{
		"$inc": bson.M{updateField: delta},
		"$set": bson.M{
			"last_update_time": time.Now(),
		},
		"$setOnInsert": bson.M{
			"object_type": objectType,
			"object_id":   objectID,
			"created_at":  time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}
