package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"websocket-server/apps/history-service/dao"
	"websocket-server/apps/history-service/model"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/redis"
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
	// 参数验证
	if err := s.validateCreateHistoryParams(params); err != nil {
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
	if err := s.dao.CreateHistory(ctx, record); err != nil {
		return nil, fmt.Errorf("创建历史记录失败: %v", err)
	}

	// 清除相关缓存
	s.clearHistoryCache(ctx, params.UserID, params.ObjectType, params.ObjectID)

	// 发送事件
	s.publishEvent(ctx, model.EventHistoryCreated, record)

	s.logger.Info(ctx, "History record created successfully",
		logger.F("recordID", record.ID),
		logger.F("userID", record.UserID),
		logger.F("actionType", record.ActionType))

	return record, nil
}

// BatchCreateHistory 批量创建历史记录
func (s *Service) BatchCreateHistory(ctx context.Context, paramsList []*model.CreateHistoryParams) ([]*model.HistoryRecord, error) {
	if len(paramsList) == 0 {
		return nil, fmt.Errorf("批量创建列表不能为空")
	}

	if len(paramsList) > model.MaxBatchCreateSize {
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
	// 参数验证
	if params.UserID <= 0 {
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
		return nil, 0, fmt.Errorf("无效的行为类型: %s", params.ActionType)
	}
	if params.ObjectType != "" && !model.IsValidObjectType(params.ObjectType) {
		return nil, 0, fmt.Errorf("无效的对象类型: %s", params.ObjectType)
	}

	return s.dao.GetUserHistory(ctx, params)
}

// GetObjectHistory 获取对象历史记录
func (s *Service) GetObjectHistory(ctx context.Context, params *model.GetObjectHistoryParams) ([]*model.HistoryRecord, int64, error) {
	// 参数验证
	if params.ObjectID <= 0 {
		return nil, 0, fmt.Errorf("对象ID无效")
	}
	if !model.IsValidObjectType(params.ObjectType) {
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
		return nil, 0, fmt.Errorf("无效的行为类型: %s", params.ActionType)
	}

	return s.dao.GetObjectHistory(ctx, params)
}

// DeleteHistory 删除历史记录
func (s *Service) DeleteHistory(ctx context.Context, params *model.DeleteHistoryParams) (int32, error) {
	// 参数验证
	if params.UserID <= 0 {
		return 0, fmt.Errorf("用户ID无效")
	}
	if len(params.RecordIDs) == 0 {
		return 0, fmt.Errorf("记录ID列表不能为空")
	}
	if len(params.RecordIDs) > model.MaxBatchDeleteSize {
		return 0, fmt.Errorf("批量删除数量超过限制: %d", model.MaxBatchDeleteSize)
	}

	deletedCount, err := s.dao.DeleteHistory(ctx, params.UserID, params.RecordIDs)
	if err != nil {
		return 0, fmt.Errorf("删除历史记录失败: %v", err)
	}

	// 清除缓存
	s.clearUserHistoryCache(ctx, params.UserID)

	s.logger.Info(ctx, "History records deleted successfully",
		logger.F("userID", params.UserID),
		logger.F("deletedCount", deletedCount))

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

	// 构建事件消息并发送到Kafka
	go func() {
		eventData := map[string]interface{}{
			"type":         eventType,
			"record_id":    record.ID,
			"user_id":      record.UserID,
			"action_type":  record.ActionType,
			"object_type":  record.ObjectType,
			"object_id":    record.ObjectID,
			"timestamp":    time.Now().Unix(),
		}

		if err := s.kafka.PublishMessage("history-events", eventData); err != nil {
			s.logger.Error(context.Background(), "Failed to publish event",
				logger.F("eventType", eventType),
				logger.F("recordID", record.ID),
				logger.F("error", err.Error()))
		}
	}()
}
