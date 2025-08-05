package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"goim-social/api/rest"
	"goim-social/apps/history-service/dao"
	"goim-social/apps/history-service/model"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/telemetry"
)

// Service 历史记录服务
type Service struct {
	dao    dao.HistoryDAO
	redis  *redis.RedisClient
	kafka  *kafka.Producer
	logger logger.Logger
}

// NewService 创建历史记录服务实例
func NewService(historyDAO dao.HistoryDAO, redis *redis.RedisClient, kafka *kafka.Producer, log logger.Logger) *Service {
	return &Service{
		dao:    historyDAO,
		redis:  redis,
		kafka:  kafka,
		logger: log,
	}
}

// CreateHistory 创建历史记录
func (s *Service) CreateHistory(ctx context.Context, params *model.CreateHistoryParams) (*model.HistoryRecord, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "history.service.CreateHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("user.id", params.UserID),
		attribute.String("action.type", params.ActionType),
		attribute.String("object.type", params.ObjectType),
		attribute.Int64("object.id", params.ObjectID),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, params.UserID)

	// 参数验证
	if err := s.validateCreateHistoryParams(params); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parameter validation failed")
		return nil, err
	}

	// 构建历史记录对象
	record := &model.HistoryRecord{
		UserID:      params.UserID,
		ActionType:  params.ActionType,
		ObjectType:  params.ObjectType,
		ObjectID:    params.ObjectID,
		ObjectTitle: strings.TrimSpace(params.ObjectTitle),
		ObjectURL:   params.ObjectURL,
		Metadata:    params.Metadata,
		IPAddress:   params.IPAddress,
		UserAgent:   params.UserAgent,
		DeviceInfo:  params.DeviceInfo,
		Location:    params.Location,
		Duration:    params.Duration,
		CreatedAt:   time.Now(),
	}

	// 创建历史记录
	_, dbSpan := telemetry.StartSpan(ctx, "history.service.CreateHistoryDB")
	if err := s.dao.CreateHistory(ctx, record); err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to create history record")
		dbSpan.End()
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create history record")
		return nil, fmt.Errorf("创建历史记录失败: %v", err)
	}
	dbSpan.SetStatus(codes.Ok, "history record created in database")
	dbSpan.End()

	// 清除相关缓存
	_, cacheSpan := telemetry.StartSpan(ctx, "history.service.ClearCache")
	s.clearHistoryCache(ctx, params.UserID, params.ObjectType, params.ObjectID)
	cacheSpan.SetStatus(codes.Ok, "cache cleared")
	cacheSpan.End()

	// 发送事件
	_, eventSpan := telemetry.StartSpan(ctx, "history.service.PublishEvent")
	s.publishEvent(ctx, model.EventHistoryCreated, record)
	eventSpan.SetStatus(codes.Ok, "event published")
	eventSpan.End()

	s.logger.Info(ctx, "History record created successfully",
		logger.F("recordID", record.ID),
		logger.F("userID", record.UserID),
		logger.F("actionType", record.ActionType))

	span.SetStatus(codes.Ok, "history record created successfully")
	return record, nil
}

// BatchCreateHistory 批量创建历史记录
func (s *Service) BatchCreateHistory(ctx context.Context, paramsList []*model.CreateHistoryParams) ([]*model.HistoryRecord, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "history.service.BatchCreateHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int("batch.size", len(paramsList)),
	)

	if len(paramsList) == 0 {
		span.SetStatus(codes.Error, "empty batch list")
		return nil, fmt.Errorf("批量创建列表不能为空")
	}

	if len(paramsList) > model.MaxBatchCreateSize {
		span.SetStatus(codes.Error, "batch size exceeds limit")
		return nil, fmt.Errorf("批量创建数量超过限制: %d", model.MaxBatchCreateSize)
	}

	// 验证所有参数
	var records []*model.HistoryRecord
	for _, params := range paramsList {
		if err := s.validateCreateHistoryParams(params); err != nil {
			return nil, fmt.Errorf("参数验证失败: %v", err)
		}

		record := &model.HistoryRecord{
			UserID:      params.UserID,
			ActionType:  params.ActionType,
			ObjectType:  params.ObjectType,
			ObjectID:    params.ObjectID,
			ObjectTitle: strings.TrimSpace(params.ObjectTitle),
			ObjectURL:   params.ObjectURL,
			Metadata:    params.Metadata,
			IPAddress:   params.IPAddress,
			UserAgent:   params.UserAgent,
			DeviceInfo:  params.DeviceInfo,
			Location:    params.Location,
			Duration:    params.Duration,
			CreatedAt:   time.Now(),
		}
		records = append(records, record)
	}

	// 批量创建历史记录
	if err := s.dao.BatchCreateHistory(ctx, records); err != nil {
		return nil, fmt.Errorf("批量创建历史记录失败: %v", err)
	}

	// 清除相关缓存
	for _, record := range records {
		s.clearHistoryCache(ctx, record.UserID, record.ObjectType, record.ObjectID)
	}

	// 发送事件
	for _, record := range records {
		s.publishEvent(ctx, model.EventHistoryCreated, record)
	}

	s.logger.Info(ctx, "Batch history records created successfully",
		logger.F("count", len(records)))

	return records, nil
}

// GetUserHistory 获取用户历史记录
func (s *Service) GetUserHistory(ctx context.Context, params *model.GetUserHistoryParams) ([]*model.HistoryRecord, int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "history.service.GetUserHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("user.id", params.UserID),
		attribute.Int("page", int(params.Page)),
		attribute.Int("page_size", int(params.PageSize)),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, params.UserID)

	// 参数验证
	if params.UserID <= 0 {
		span.SetStatus(codes.Error, "invalid user ID")
		return nil, 0, fmt.Errorf("用户ID无效")
	}

	// 设置默认值
	if params.Page <= 0 {
		params.Page = model.DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}

	// 验证行为类型和对象类型
	if params.ActionType != "" && !model.IsValidActionType(params.ActionType) {
		span.SetStatus(codes.Error, "invalid action type")
		return nil, 0, fmt.Errorf("无效的行为类型: %s", params.ActionType)
	}
	if params.ObjectType != "" && !model.IsValidObjectType(params.ObjectType) {
		span.SetStatus(codes.Error, "invalid object type")
		return nil, 0, fmt.Errorf("无效的对象类型: %s", params.ObjectType)
	}

	// 查询用户历史记录
	records, total, err := s.dao.GetUserHistory(ctx, params)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get user history")
		return nil, 0, err
	}

	span.SetAttributes(attribute.Int("result.count", len(records)))
	span.SetStatus(codes.Ok, "user history retrieved successfully")
	return records, total, nil
}

// GetObjectHistory 获取对象历史记录
func (s *Service) GetObjectHistory(ctx context.Context, params *model.GetObjectHistoryParams) ([]*model.HistoryRecord, int64, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "history.service.GetObjectHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("object.id", params.ObjectID),
		attribute.String("object.type", params.ObjectType),
		attribute.Int("page", int(params.Page)),
		attribute.Int("page_size", int(params.PageSize)),
	)

	// 参数验证
	if params.ObjectID <= 0 {
		span.SetStatus(codes.Error, "invalid object ID")
		return nil, 0, fmt.Errorf("对象ID无效")
	}
	if !model.IsValidObjectType(params.ObjectType) {
		span.SetStatus(codes.Error, "invalid object type")
		return nil, 0, fmt.Errorf("无效的对象类型: %s", params.ObjectType)
	}

	// 设置默认值
	if params.Page <= 0 {
		params.Page = model.DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}

	// 验证行为类型
	if params.ActionType != "" && !model.IsValidActionType(params.ActionType) {
		span.SetStatus(codes.Error, "invalid action type")
		return nil, 0, fmt.Errorf("无效的行为类型: %s", params.ActionType)
	}

	// 查询对象历史记录
	records, total, err := s.dao.GetObjectHistory(ctx, params)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get object history")
		return nil, 0, err
	}

	span.SetAttributes(attribute.Int("result.count", len(records)))
	span.SetStatus(codes.Ok, "object history retrieved successfully")
	return records, total, nil
}

// DeleteHistory 删除历史记录
func (s *Service) DeleteHistory(ctx context.Context, params *model.DeleteHistoryParams) (int32, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "history.service.DeleteHistory")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int64("user.id", params.UserID),
		attribute.Int("record_ids.count", len(params.RecordIDs)),
	)

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, params.UserID)

	// 参数验证
	if params.UserID <= 0 {
		span.SetStatus(codes.Error, "invalid user ID")
		return 0, fmt.Errorf("用户ID无效")
	}
	if len(params.RecordIDs) == 0 {
		span.SetStatus(codes.Error, "empty record IDs list")
		return 0, fmt.Errorf("记录ID列表不能为空")
	}
	if len(params.RecordIDs) > model.MaxBatchDeleteSize {
		span.SetStatus(codes.Error, "batch delete size exceeds limit")
		return 0, fmt.Errorf("批量删除数量超过限制: %d", model.MaxBatchDeleteSize)
	}

	// 执行删除操作
	deletedCount, err := s.dao.DeleteHistory(ctx, params.UserID, params.RecordIDs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete history records")
		return 0, fmt.Errorf("删除历史记录失败: %v", err)
	}

	// 清除缓存
	s.clearUserHistoryCache(ctx, params.UserID)

	s.logger.Info(ctx, "History records deleted successfully",
		logger.F("userID", params.UserID),
		logger.F("deletedCount", deletedCount))

	span.SetAttributes(attribute.Int("deleted.count", int(deletedCount)))
	span.SetStatus(codes.Ok, "history records deleted successfully")
	return deletedCount, nil
}

// ClearUserHistory 清空用户历史记录
func (s *Service) ClearUserHistory(ctx context.Context, params *model.ClearUserHistoryParams) (int32, error) {
	// 参数验证
	if params.UserID <= 0 {
		return 0, fmt.Errorf("用户ID无效")
	}

	// 验证行为类型和对象类型
	if params.ActionType != "" && !model.IsValidActionType(params.ActionType) {
		return 0, fmt.Errorf("无效的行为类型: %s", params.ActionType)
	}
	if params.ObjectType != "" && !model.IsValidObjectType(params.ObjectType) {
		return 0, fmt.Errorf("无效的对象类型: %s", params.ObjectType)
	}

	deletedCount, err := s.dao.ClearUserHistory(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("清空用户历史记录失败: %v", err)
	}

	// 清除缓存
	s.clearUserHistoryCache(ctx, params.UserID)

	s.logger.Info(ctx, "User history cleared successfully",
		logger.F("userID", params.UserID),
		logger.F("deletedCount", deletedCount))

	return deletedCount, nil
}

// GetUserActionStats 获取用户行为统计
func (s *Service) GetUserActionStats(ctx context.Context, userID int64, actionType string) ([]*model.UserActionStats, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}

	if actionType != "" {
		// 获取特定行为类型的统计
		if !model.IsValidActionType(actionType) {
			return nil, fmt.Errorf("无效的行为类型: %s", actionType)
		}
		stats, err := s.dao.GetUserActionStats(ctx, userID, actionType)
		if err != nil {
			return nil, err
		}
		return []*model.UserActionStats{stats}, nil
	}

	// 获取所有行为类型的统计
	return s.dao.GetAllUserActionStats(ctx, userID)
}

// GetHotObjects 获取热门对象
func (s *Service) GetHotObjects(ctx context.Context, params *model.GetHotObjectsParams) ([]*model.ObjectHotStats, error) {
	// 参数验证
	if params.ObjectType != "" && !model.IsValidObjectType(params.ObjectType) {
		return nil, fmt.Errorf("无效的对象类型: %s", params.ObjectType)
	}
	if params.TimeRange != "" && !model.IsValidTimeRange(params.TimeRange) {
		return nil, fmt.Errorf("无效的时间范围: %s", params.TimeRange)
	}

	// 设置默认值
	if params.TimeRange == "" {
		params.TimeRange = model.TimeRangeAll
	}
	if params.Limit <= 0 {
		params.Limit = model.DefaultHotObjectLimit
	}
	if params.Limit > model.MaxHotObjectLimit {
		params.Limit = model.MaxHotObjectLimit
	}

	return s.dao.GetHotObjects(ctx, params)
}

// GetUserActivityStats 获取用户活跃度统计
func (s *Service) GetUserActivityStats(ctx context.Context, params *model.GetUserActivityStatsParams) ([]*model.UserActivityStats, error) {
	if params.UserID <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}

	return s.dao.GetUserActivityStats(ctx, params)
}

// 辅助方法

// validateCreateHistoryParams 验证创建历史记录参数
func (s *Service) validateCreateHistoryParams(params *model.CreateHistoryParams) error {
	if params.UserID <= 0 {
		return fmt.Errorf("用户ID无效")
	}
	if !model.IsValidActionType(params.ActionType) {
		return fmt.Errorf("无效的行为类型: %s", params.ActionType)
	}
	if !model.IsValidObjectType(params.ObjectType) {
		return fmt.Errorf("无效的对象类型: %s", params.ObjectType)
	}
	if params.ObjectID <= 0 {
		return fmt.Errorf("对象ID无效")
	}

	return nil
}

// clearHistoryCache 清除历史记录相关缓存
func (s *Service) clearHistoryCache(ctx context.Context, userID int64, objectType string, objectID int64) {
	// 清除用户历史缓存
	s.clearUserHistoryCache(ctx, userID)

	// 清除对象统计缓存
	objectStatsKey := fmt.Sprintf("%s%s:%d", model.ObjectStatsCachePrefix, objectType, objectID)
	s.redis.Del(ctx, objectStatsKey)

	// 清除热门对象缓存
	hotObjectsPattern := fmt.Sprintf("%s%s:*", model.HotObjectsCachePrefix, objectType)
	keys, err := s.redis.Keys(ctx, hotObjectsPattern)
	if err == nil && len(keys) > 0 {
		s.redis.Del(ctx, keys...)
	}
}

// clearUserHistoryCache 清除用户历史缓存
func (s *Service) clearUserHistoryCache(ctx context.Context, userID int64) {
	// 清除用户历史缓存
	pattern := fmt.Sprintf("%s%d:*", model.HistoryCachePrefix, userID)
	keys, err := s.redis.Keys(ctx, pattern)
	if err == nil && len(keys) > 0 {
		s.redis.Del(ctx, keys...)
	}

	// 清除用户统计缓存
	userStatsKey := fmt.Sprintf("%s%d", model.UserStatsCachePrefix, userID)
	s.redis.Del(ctx, userStatsKey)

	// 清除用户活跃度缓存
	activityStatsKey := fmt.Sprintf("%s%d", model.ActivityStatsCachePrefix, userID)
	s.redis.Del(ctx, activityStatsKey)
}

// publishEvent 发布事件
func (s *Service) publishEvent(ctx context.Context, eventType string, record *model.HistoryRecord) {
	if s.kafka == nil {
		return
	}

	// 构建protobuf事件消息并发送到Kafka
	go func() {
		event := &rest.HistoryEvent{
			Type:              eventType,
			RecordId:          record.ID,
			UserId:            record.UserID,
			ActionType:        convertActionTypeToProto(record.ActionType),
			HistoryObjectType: convertObjectTypeToProto(record.ObjectType),
			ObjectId:          record.ObjectID,
			Timestamp:         time.Now().Unix(),
		}

		if err := s.kafka.PublishMessage("history-events", event); err != nil {
			s.logger.Error(context.Background(), "Failed to publish event",
				logger.F("eventType", eventType),
				logger.F("recordID", record.ID),
				logger.F("error", err.Error()))
		}
	}()
}

// convertActionTypeToProto 将行为类型转换为protobuf枚举
func convertActionTypeToProto(actionType string) rest.ActionType {
	switch actionType {
	case model.ActionTypeView:
		return rest.ActionType_ACTION_TYPE_VIEW
	case model.ActionTypeLike:
		return rest.ActionType_ACTION_TYPE_LIKE
	case model.ActionTypeFavorite:
		return rest.ActionType_ACTION_TYPE_FAVORITE
	case model.ActionTypeShare:
		return rest.ActionType_ACTION_TYPE_SHARE
	case model.ActionTypeComment:
		return rest.ActionType_ACTION_TYPE_COMMENT
	case model.ActionTypeFollow:
		return rest.ActionType_ACTION_TYPE_FOLLOW
	case model.ActionTypeLogin:
		return rest.ActionType_ACTION_TYPE_LOGIN
	case model.ActionTypeSearch:
		return rest.ActionType_ACTION_TYPE_SEARCH
	case model.ActionTypeDownload:
		return rest.ActionType_ACTION_TYPE_DOWNLOAD
	case model.ActionTypePurchase:
		return rest.ActionType_ACTION_TYPE_PURCHASE
	default:
		return rest.ActionType_ACTION_TYPE_UNSPECIFIED
	}
}

// convertObjectTypeToProto 将对象类型转换为protobuf枚举
func convertObjectTypeToProto(objectType string) rest.HistoryObjectType {
	switch objectType {
	case model.ObjectTypePost:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_POST
	case model.ObjectTypeArticle:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_ARTICLE
	case model.ObjectTypeVideo:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_VIDEO
	case model.ObjectTypeUser:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_USER
	case model.ObjectTypeProduct:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_PRODUCT
	case model.ObjectTypeGroup:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_GROUP
	default:
		return rest.HistoryObjectType_HISTORY_OBJECT_TYPE_UNSPECIFIED
	}
}
