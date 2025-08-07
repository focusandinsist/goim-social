package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/api/rest"
	"goim-social/apps/message-service/dao"
	"goim-social/apps/message-service/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/database"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/telemetry"
)

// Service Message服务（合并了历史记录功能）
type Service struct {
	db     *database.MongoDB
	redis  *redis.RedisClient
	kafka  *kafka.Producer
	dao    dao.MessageDAO
	logger logger.Logger
}

// NewService 创建Message服务实例
func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer, logger logger.Logger) *Service {
	messageDAO := dao.NewMongoDAO(db.GetDatabase())
	return &Service{
		db:     db,
		redis:  redis,
		kafka:  kafka,
		dao:    messageDAO,
		logger: logger,
	}
}

// SaveMessage 保存消息到数据库
func (s *Service) SaveMessage(ctx context.Context, msg *model.Message) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "message.service.SaveMessage")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("message.id", msg.MessageID),
		attribute.Int64("message.from", msg.From),
		attribute.Int64("message.to", msg.To),
		attribute.Int64("message.group_id", msg.GroupID),
		attribute.String("message.content", msg.Content),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, msg.From)
	if msg.GroupID > 0 {
		ctx = tracecontext.WithGroupID(ctx, msg.GroupID)
	}

	collection := s.db.GetCollection("messages")

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	msg.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to save message")
		return fmt.Errorf("保存消息失败: %v", err)
	}

	span.SetStatus(codes.Ok, "message saved successfully")
	return nil
}

// GetMessageHistory 获取消息历史
func (s *Service) GetMessageHistory(ctx context.Context, userID, groupID int64, page, size int) ([]*model.Message, int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "message.service.GetMessageHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("user.id", userID),
		attribute.Int64("group.id", groupID),
		attribute.Int("page", page),
		attribute.Int("size", size),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)
	if groupID > 0 {
		ctx = tracecontext.WithGroupID(ctx, groupID)
	}

	collection := s.db.GetCollection("messages")

	// 构建查询条件
	var filter bson.M
	if groupID > 0 {
		// 群聊消息：查询该群组的所有消息
		filter = bson.M{"group_id": groupID}
		span.SetAttributes(attribute.String("query.type", "group"))
	} else {
		// 私聊消息：查询用户参与的对话
		// 这里需要另一个参数来指定对话的另一方
		// 暂时查询用户相关的所有私聊消息
		filter = bson.M{
			"$and": []bson.M{
				{"group_id": bson.M{"$eq": 0}}, // 不是群聊
				{
					"$or": []bson.M{
						{"from": userID},
						{"to": userID},
					},
				},
			},
		}
		span.SetAttributes(attribute.String("query.type", "private"))
	}

	// 获取总数
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to count messages")
		return nil, 0, fmt.Errorf("统计消息数量失败: %v", err)
	}

	// 分页查询
	skip := int64((page - 1) * size)
	limit := int64(size)

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}). // 按时间倒序
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query messages")
		return nil, 0, fmt.Errorf("查询消息失败: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []*model.Message
	for cursor.Next(ctx) {
		var msg model.Message
		if err := cursor.Decode(&msg); err != nil {
			log.Printf("解码消息失败: %v", err)
			continue
		}
		messages = append(messages, &msg)
	}

	span.SetAttributes(
		attribute.Int64("result.total", total),
		attribute.Int("result.count", len(messages)),
	)
	span.SetStatus(codes.Ok, "message history retrieved successfully")
	return messages, total, nil
}

// UpdateMessageStatus 更新消息状态
func (s *Service) UpdateMessageStatus(ctx context.Context, messageID int64, status string) error {
	collection := s.db.GetCollection("messages")

	filter := bson.M{"message_id": messageID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("更新消息状态失败: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("消息不存在: MessageID=%d", messageID)
	}

	return nil
}

// MarkMessageAsRead 标记消息为已读
func (s *Service) MarkMessageAsRead(ctx context.Context, userID, messageID int64) error {
	// 验证用户是否有权限标记该消息为已读
	collection := s.db.GetCollection("messages")

	filter := bson.M{
		"message_id": messageID,
		"$or": []bson.M{
			{"to": userID},                 // 私聊中的接收者
			{"group_id": bson.M{"$gt": 0}}, // 群聊消息（需要进一步验证群成员身份）
		},
	}

	var message model.Message
	err := collection.FindOne(ctx, filter).Decode(&message)
	if err != nil {
		return fmt.Errorf("消息不存在或无权限: %v", err)
	}

	// 更新消息状态为已读
	return s.UpdateMessageStatus(ctx, messageID, model.MessageStatusRead)
}

// DeleteMessage 删除消息
func (s *Service) DeleteMessage(ctx context.Context, messageID int64, userID int64) error {
	collection := s.db.GetCollection("messages")

	// 只允许发送者删除自己的消息
	filter := bson.M{
		"message_id": messageID,
		"from":       userID,
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("删除消息失败: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("消息不存在或无权限删除")
	}

	return nil
}

// SaveWSMessage 保存WebSocket消息
func (s *Service) SaveWSMessage(ctx context.Context, msg *rest.WSMessage) error {
	// 构造消息对象
	message := &model.Message{
		ID:          primitive.NewObjectID(),
		MessageID:   msg.MessageId,
		From:        msg.From,
		To:          msg.To,
		GroupID:     msg.GroupId,
		Content:     msg.Content,
		MessageType: int(msg.MessageType),
		Timestamp:   msg.Timestamp,
		AckID:       msg.AckId,
		Status:      model.MessageStatusSent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.SaveMessage(ctx, message)
}

// SendMessage 发送消息（HTTP接口用）
func (s *Service) SendMessage(ctx context.Context, req *rest.SendMessageRequest) (int64, string, error) {
	// 生成消息ID和AckID
	messageID := time.Now().UnixNano()
	ackID := fmt.Sprintf("ack_%d", messageID)

	// 构造消息对象
	message := &model.Message{
		ID:          primitive.NewObjectID(),
		MessageID:   messageID,
		From:        0, // TODO: 从上下文获取用户ID
		To:          req.To,
		GroupID:     req.GroupId,
		Content:     req.Content,
		MessageType: int(req.MessageType),
		Timestamp:   time.Now().Unix(),
		AckID:       ackID,
		Status:      model.MessageStatusSent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := s.SaveMessage(ctx, message)
	if err != nil {
		return 0, "", err
	}

	return messageID, ackID, nil
}

// GetUnreadMessages 获取未读消息
func (s *Service) GetUnreadMessages(ctx context.Context, userID int64) ([]*model.Message, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "message.service.GetUnreadMessages")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int64("user.id", userID))

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	collection := s.db.GetCollection("messages")

	// 查询未读消息
	filter := bson.M{
		"$or": []bson.M{
			{"to": userID, "status": bson.M{"$ne": model.MessageStatusRead}},
			{"group_id": bson.M{"$gt": 0}, "status": bson.M{"$ne": model.MessageStatusRead}},
		},
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query unread messages")
		return nil, fmt.Errorf("查询未读消息失败: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []*model.Message
	for cursor.Next(ctx) {
		var msg model.Message
		if err := cursor.Decode(&msg); err != nil {
			log.Printf("解码消息失败: %v", err)
			continue
		}
		messages = append(messages, &msg)
	}

	span.SetAttributes(attribute.Int("result.count", len(messages)))
	span.SetStatus(codes.Ok, "unread messages retrieved successfully")
	return messages, nil
}

// MarkMessagesAsRead 批量标记消息已读
func (s *Service) MarkMessagesAsRead(ctx context.Context, userID int64, messageIDs []int64) ([]int64, error) {
	var failedIDs []int64

	for _, messageID := range messageIDs {
		if err := s.MarkMessageAsRead(ctx, userID, messageID); err != nil {
			failedIDs = append(failedIDs, messageID)
		}
	}

	return failedIDs, nil
}

// ==================== 历史记录相关服务方法 ====================

// RecordUserAction 记录用户行为
func (s *Service) RecordUserAction(ctx context.Context, userID int64, actionType, objectType string, objectID int64, objectTitle, objectURL, metadata, ipAddress, userAgent, deviceInfo, location string, duration int64) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "message.service.RecordUserAction")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("history.user_id", userID),
		attribute.String("history.action_type", actionType),
		attribute.String("history.object_type", objectType),
		attribute.Int64("history.object_id", objectID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 参数验证
	if userID <= 0 {
		span.SetStatus(codes.Error, "invalid user ID")
		return fmt.Errorf("invalid user ID")
	}
	if actionType == "" {
		span.SetStatus(codes.Error, "action type is required")
		return fmt.Errorf("action type is required")
	}
	if objectType == "" {
		span.SetStatus(codes.Error, "object type is required")
		return fmt.Errorf("object type is required")
	}
	if objectID <= 0 {
		span.SetStatus(codes.Error, "invalid object ID")
		return fmt.Errorf("invalid object ID")
	}

	// 构建历史记录
	record := &model.HistoryRecord{
		UserID:      userID,
		ActionType:  actionType,
		ObjectType:  objectType,
		ObjectID:    objectID,
		ObjectTitle: objectTitle,
		ObjectURL:   objectURL,
		Metadata:    metadata,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		DeviceInfo:  deviceInfo,
		Location:    location,
		Duration:    duration,
	}

	// 记录用户行为
	if err := s.dao.RecordUserAction(ctx, record); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to record user action")
		s.logger.Error(ctx, "Failed to record user action",
			logger.F("userID", userID),
			logger.F("actionType", actionType),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to record user action: %v", err)
	}

	// 异步更新统计数据
	go func() {
		if err := s.dao.UpdateUserActionStats(context.Background(), userID, actionType); err != nil {
			s.logger.Error(context.Background(), "Failed to update user action stats",
				logger.F("userID", userID),
				logger.F("actionType", actionType),
				logger.F("error", err.Error()))
		}

		if err := s.dao.UpdateObjectHotStats(context.Background(), objectType, objectID, actionType, 1); err != nil {
			s.logger.Error(context.Background(), "Failed to update object hot stats",
				logger.F("objectType", objectType),
				logger.F("objectID", objectID),
				logger.F("actionType", actionType),
				logger.F("error", err.Error()))
		}
	}()

	s.logger.Info(ctx, "User action recorded successfully",
		logger.F("userID", userID),
		logger.F("actionType", actionType),
		logger.F("objectType", objectType),
		logger.F("objectID", objectID))

	span.SetStatus(codes.Ok, "user action recorded successfully")
	return nil
}

// BatchRecordUserAction 批量记录用户行为
func (s *Service) BatchRecordUserAction(ctx context.Context, records []*model.HistoryRecord) (int32, int32, []string) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "message.service.BatchRecordUserAction")
	defer span.End()

	// 设置span属性
	span.SetAttributes(attribute.Int("history.record_count", len(records)))

	if len(records) == 0 {
		span.SetStatus(codes.Error, "no records provided")
		return 0, 0, []string{"no records provided"}
	}

	// 验证记录
	var validRecords []*model.HistoryRecord
	var errors []string

	for i, record := range records {
		if record.UserID <= 0 {
			errors = append(errors, fmt.Sprintf("record %d: invalid user ID", i))
			continue
		}
		if record.ActionType == "" {
			errors = append(errors, fmt.Sprintf("record %d: action type is required", i))
			continue
		}
		if record.ObjectType == "" {
			errors = append(errors, fmt.Sprintf("record %d: object type is required", i))
			continue
		}
		if record.ObjectID <= 0 {
			errors = append(errors, fmt.Sprintf("record %d: invalid object ID", i))
			continue
		}
		validRecords = append(validRecords, record)
	}

	successCount := int32(0)
	failedCount := int32(len(records) - len(validRecords))

	if len(validRecords) > 0 {
		if err := s.dao.BatchRecordUserAction(ctx, validRecords); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to batch record user actions")
			s.logger.Error(ctx, "Failed to batch record user actions",
				logger.F("recordCount", len(validRecords)),
				logger.F("error", err.Error()))
			errors = append(errors, fmt.Sprintf("database error: %v", err))
			failedCount += int32(len(validRecords))
		} else {
			successCount = int32(len(validRecords))
		}
	}

	s.logger.Info(ctx, "Batch record user actions completed",
		logger.F("totalRecords", len(records)),
		logger.F("successCount", successCount),
		logger.F("failedCount", failedCount))

	span.SetAttributes(
		attribute.Int("history.success_count", int(successCount)),
		attribute.Int("history.failed_count", int(failedCount)),
	)
	span.SetStatus(codes.Ok, "batch record completed")

	return successCount, failedCount, errors
}

// GetUserHistory 获取用户历史记录
func (s *Service) GetUserHistory(ctx context.Context, userID int64, actionType, objectType string, startTime, endTime time.Time, page, pageSize int32) ([]*model.HistoryRecord, int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "message.service.GetUserHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("history.user_id", userID),
		attribute.String("history.action_type", actionType),
		attribute.String("history.object_type", objectType),
		attribute.Int("history.page", int(page)),
		attribute.Int("history.page_size", int(pageSize)),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 参数验证
	if userID <= 0 {
		span.SetStatus(codes.Error, "invalid user ID")
		return nil, 0, fmt.Errorf("invalid user ID")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > model.MaxPageSize {
		pageSize = model.DefaultPageSize
	}

	// 获取历史记录
	records, total, err := s.dao.GetUserHistory(ctx, userID, actionType, objectType, startTime, endTime, page, pageSize)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user history")
		s.logger.Error(ctx, "Failed to get user history",
			logger.F("userID", userID),
			logger.F("error", err.Error()))
		return nil, 0, fmt.Errorf("failed to get user history: %v", err)
	}

	span.SetAttributes(
		attribute.Int64("history.total", total),
		attribute.Int("history.count", len(records)),
	)

	s.logger.Info(ctx, "User history retrieved successfully",
		logger.F("userID", userID),
		logger.F("total", total),
		logger.F("count", len(records)))

	span.SetStatus(codes.Ok, "user history retrieved successfully")
	return records, total, nil
}

// DeleteUserHistory 删除用户历史记录
func (s *Service) DeleteUserHistory(ctx context.Context, userID int64, recordIDs []string) (int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "message.service.DeleteUserHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("history.user_id", userID),
		attribute.Int("history.record_count", len(recordIDs)),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 参数验证
	if userID <= 0 {
		span.SetStatus(codes.Error, "invalid user ID")
		return 0, fmt.Errorf("invalid user ID")
	}
	if len(recordIDs) == 0 {
		span.SetStatus(codes.Error, "no record IDs provided")
		return 0, fmt.Errorf("no record IDs provided")
	}

	// 删除历史记录
	deletedCount, err := s.dao.DeleteUserHistory(ctx, userID, recordIDs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete user history")
		s.logger.Error(ctx, "Failed to delete user history",
			logger.F("userID", userID),
			logger.F("recordIDs", strings.Join(recordIDs, ",")),
			logger.F("error", err.Error()))
		return 0, fmt.Errorf("failed to delete user history: %v", err)
	}

	span.SetAttributes(attribute.Int64("history.deleted_count", deletedCount))

	s.logger.Info(ctx, "User history deleted successfully",
		logger.F("userID", userID),
		logger.F("deletedCount", deletedCount))

	span.SetStatus(codes.Ok, "user history deleted successfully")
	return deletedCount, nil
}

// GetUserActionStats 获取用户行为统计
func (s *Service) GetUserActionStats(ctx context.Context, userID int64, actionType string, startTime, endTime time.Time, groupBy string) ([]*model.ActionStatItem, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "message.service.GetUserActionStats")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("history.user_id", userID),
		attribute.String("history.action_type", actionType),
		attribute.String("history.group_by", groupBy),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, userID)

	// 参数验证
	if userID <= 0 {
		span.SetStatus(codes.Error, "invalid user ID")
		return nil, fmt.Errorf("invalid user ID")
	}
	if groupBy == "" {
		groupBy = model.GroupByDay
	}

	// 获取统计数据
	stats, err := s.dao.GetUserActionStats(ctx, userID, actionType, startTime, endTime, groupBy)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user action stats")
		s.logger.Error(ctx, "Failed to get user action stats",
			logger.F("userID", userID),
			logger.F("actionType", actionType),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get user action stats: %v", err)
	}

	span.SetAttributes(attribute.Int("history.stats_count", len(stats)))

	s.logger.Info(ctx, "User action stats retrieved successfully",
		logger.F("userID", userID),
		logger.F("actionType", actionType),
		logger.F("statsCount", len(stats)))

	span.SetStatus(codes.Ok, "user action stats retrieved successfully")
	return stats, nil
}
