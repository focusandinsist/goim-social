package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"goim-social/apps/interaction-service/dao"
	"goim-social/apps/interaction-service/model"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
)

// Service 互动服务
type Service struct {
	dao    dao.InteractionDAO
	redis  *redis.RedisClient
	kafka  *kafka.Producer
	logger logger.Logger
}

// NewService 创建互动服务实例
func NewService(interactionDAO dao.InteractionDAO, redis *redis.RedisClient, kafka *kafka.Producer, log logger.Logger) *Service {
	return &Service{
		dao:    interactionDAO,
		redis:  redis,
		kafka:  kafka,
		logger: log,
	}
}

// DoInteraction 执行互动操作
func (s *Service) DoInteraction(ctx context.Context, userID, objectID int64, objectType, interactionType, metadata string) (*model.Interaction, error) {
	// 参数验证
	if userID <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}
	if objectID <= 0 {
		return nil, fmt.Errorf("对象ID无效")
	}
	if !model.ValidateObjectType(objectType) {
		return nil, fmt.Errorf("对象类型无效: %s", objectType)
	}
	if !model.ValidateInteractionType(interactionType) {
		return nil, fmt.Errorf("互动类型无效: %s", interactionType)
	}

	// 创建互动记录
	interaction := &model.Interaction{
		UserID:          userID,
		ObjectID:        objectID,
		ObjectType:      objectType,
		InteractionType: interactionType,
		Metadata:        metadata,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.dao.CreateInteraction(ctx, interaction); err != nil {
		return nil, fmt.Errorf("创建互动失败: %v", err)
	}

	// 清除相关缓存
	s.clearInteractionCache(ctx, userID, objectID, objectType, interactionType)

	// 发送事件到消息队列
	s.publishInteractionEvent(ctx, "create", interaction)

	return interaction, nil
}

// UndoInteraction 取消互动操作
func (s *Service) UndoInteraction(ctx context.Context, userID, objectID int64, objectType, interactionType string) error {
	// 参数验证
	if userID <= 0 {
		return fmt.Errorf("用户ID无效")
	}
	if objectID <= 0 {
		return fmt.Errorf("对象ID无效")
	}
	if !model.ValidateObjectType(objectType) {
		return fmt.Errorf("对象类型无效: %s", objectType)
	}
	if !model.ValidateInteractionType(interactionType) {
		return fmt.Errorf("互动类型无效: %s", interactionType)
	}

	// 删除互动记录
	if err := s.dao.DeleteInteraction(ctx, userID, objectID, objectType, interactionType); err != nil {
		return fmt.Errorf("取消互动失败: %v", err)
	}

	// 清除相关缓存
	s.clearInteractionCache(ctx, userID, objectID, objectType, interactionType)

	// 发送事件到消息队列
	interaction := &model.Interaction{
		UserID:          userID,
		ObjectID:        objectID,
		ObjectType:      objectType,
		InteractionType: interactionType,
	}
	s.publishInteractionEvent(ctx, "delete", interaction)

	return nil
}

// CheckInteraction 检查互动状态
func (s *Service) CheckInteraction(ctx context.Context, userID, objectID int64, objectType, interactionType string) (bool, *model.Interaction, error) {
	// 参数验证
	if userID <= 0 || objectID <= 0 {
		return false, nil, fmt.Errorf("参数无效")
	}
	if !model.ValidateObjectType(objectType) {
		return false, nil, fmt.Errorf("对象类型无效: %s", objectType)
	}
	if !model.ValidateInteractionType(interactionType) {
		return false, nil, fmt.Errorf("互动类型无效: %s", interactionType)
	}

	// 先尝试从缓存获取
	cacheKey := fmt.Sprintf("%s:%d", model.GetUserInteractionKey(userID, objectType, interactionType), objectID)
	if s.redis != nil {
		if cached, err := s.redis.Get(ctx, cacheKey); err == nil && cached != "" {
			if cached == "1" {
				// 从数据库获取详细信息
				interaction, err := s.dao.GetInteraction(ctx, userID, objectID, objectType, interactionType)
				if err == nil {
					return true, interaction, nil
				}
			} else {
				return false, nil, nil
			}
		}
	}

	// 从数据库查询
	interaction, err := s.dao.GetInteraction(ctx, userID, objectID, objectType, interactionType)
	if err != nil {
		if err.Error() == "record not found" {
			// 缓存结果
			if s.redis != nil {
				s.redis.Set(ctx, cacheKey, "0", time.Duration(model.CacheExpireUserAction)*time.Second)
			}
			return false, nil, nil
		}
		return false, nil, err
	}

	// 缓存结果
	if s.redis != nil {
		s.redis.Set(ctx, cacheKey, "1", time.Duration(model.CacheExpireUserAction)*time.Second)
	}

	return true, interaction, nil
}

// BatchCheckInteraction 批量检查互动状态
func (s *Service) BatchCheckInteraction(ctx context.Context, userID int64, objectIDs []int64, objectType, interactionType string) (map[int64]bool, error) {
	// 参数验证
	if userID <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}
	if len(objectIDs) == 0 {
		return make(map[int64]bool), nil
	}
	if len(objectIDs) > model.MaxBatchSize {
		return nil, fmt.Errorf("批量查询数量超过限制: %d", model.MaxBatchSize)
	}
	if !model.ValidateObjectType(objectType) {
		return nil, fmt.Errorf("对象类型无效: %s", objectType)
	}
	if !model.ValidateInteractionType(interactionType) {
		return nil, fmt.Errorf("互动类型无效: %s", interactionType)
	}

	query := &model.BatchInteractionQuery{
		UserID:          userID,
		ObjectIDs:       objectIDs,
		ObjectType:      objectType,
		InteractionType: interactionType,
	}

	return s.dao.BatchCheckInteractions(ctx, query)
}

// GetObjectStats 获取对象统计
func (s *Service) GetObjectStats(ctx context.Context, objectID int64, objectType string) (*model.InteractionStats, error) {
	// 参数验证
	if objectID <= 0 {
		return nil, fmt.Errorf("对象ID无效")
	}
	if !model.ValidateObjectType(objectType) {
		return nil, fmt.Errorf("对象类型无效: %s", objectType)
	}

	// 先尝试从缓存获取
	cacheKey := model.GetStatsKey(objectID, objectType)
	if s.redis != nil {
		if cached, err := s.redis.Get(ctx, cacheKey); err == nil && cached != "" {
			var stats model.InteractionStats
			if json.Unmarshal([]byte(cached), &stats) == nil {
				return &stats, nil
			}
		}
	}

	// 从数据库获取
	stats, err := s.dao.GetInteractionStats(ctx, objectID, objectType)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	if s.redis != nil {
		if data, err := json.Marshal(stats); err == nil {
			s.redis.Set(ctx, cacheKey, string(data), time.Duration(model.CacheExpireStats)*time.Second)
		}
	}

	return stats, nil
}

// GetBatchObjectStats 批量获取对象统计
func (s *Service) GetBatchObjectStats(ctx context.Context, objectIDs []int64, objectType string) ([]*model.InteractionStats, error) {
	// 参数验证
	if len(objectIDs) == 0 {
		return []*model.InteractionStats{}, nil
	}
	if len(objectIDs) > model.MaxBatchSize {
		return nil, fmt.Errorf("批量查询数量超过限制: %d", model.MaxBatchSize)
	}
	if !model.ValidateObjectType(objectType) {
		return nil, fmt.Errorf("对象类型无效: %s", objectType)
	}

	query := &model.InteractionStatsQuery{
		ObjectIDs:  objectIDs,
		ObjectType: objectType,
	}

	return s.dao.BatchGetInteractionStats(ctx, query)
}

// GetUserInteractions 获取用户互动列表
func (s *Service) GetUserInteractions(ctx context.Context, userID int64, objectType, interactionType string, page, pageSize int32) ([]*model.Interaction, int64, error) {
	// 参数验证
	if userID <= 0 {
		return nil, 0, fmt.Errorf("用户ID无效")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	query := &model.InteractionQuery{
		UserID:          userID,
		ObjectType:      objectType,
		InteractionType: interactionType,
		Page:            page,
		PageSize:        pageSize,
		SortBy:          model.SortByCreatedAt,
		SortOrder:       model.SortOrderDesc,
	}

	return s.dao.GetUserInteractions(ctx, query)
}

// GetObjectInteractions 获取对象互动列表
func (s *Service) GetObjectInteractions(ctx context.Context, objectID int64, objectType, interactionType string, page, pageSize int32) ([]*model.Interaction, int64, error) {
	// 参数验证
	if objectID <= 0 {
		return nil, 0, fmt.Errorf("对象ID无效")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	query := &model.InteractionQuery{
		ObjectID:        objectID,
		ObjectType:      objectType,
		InteractionType: interactionType,
		Page:            page,
		PageSize:        pageSize,
		SortBy:          model.SortByCreatedAt,
		SortOrder:       model.SortOrderDesc,
	}

	return s.dao.GetObjectInteractions(ctx, query)
}

// GetHotObjects 获取热门对象
func (s *Service) GetHotObjects(ctx context.Context, objectType, interactionType string, limit int32) ([]*model.HotObject, error) {
	// 参数验证
	if !model.ValidateObjectType(objectType) {
		return nil, fmt.Errorf("对象类型无效: %s", objectType)
	}
	if interactionType != "" && !model.ValidateInteractionType(interactionType) {
		return nil, fmt.Errorf("互动类型无效: %s", interactionType)
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.dao.GetHotObjects(ctx, objectType, interactionType, limit)
}

// clearInteractionCache 清除互动相关缓存
func (s *Service) clearInteractionCache(ctx context.Context, userID, objectID int64, objectType, interactionType string) {
	if s.redis == nil {
		return
	}

	// 清除用户互动缓存
	userCacheKey := model.GetUserInteractionKey(userID, objectType, interactionType)
	s.redis.Del(ctx, userCacheKey)

	// 清除统计缓存
	statsCacheKey := model.GetStatsKey(objectID, objectType)
	s.redis.Del(ctx, statsCacheKey)

	// 清除热门列表缓存
	hotCacheKey := model.GetHotListKey(objectType, interactionType)
	s.redis.Del(ctx, hotCacheKey)
}

// publishInteractionEvent 发布互动事件到消息队列
func (s *Service) publishInteractionEvent(ctx context.Context, eventType string, interaction *model.Interaction) {
	if s.kafka == nil {
		return
	}

	event := &model.InteractionEvent{
		EventType:       eventType,
		UserID:          interaction.UserID,
		ObjectID:        interaction.ObjectID,
		ObjectType:      interaction.ObjectType,
		InteractionType: interaction.InteractionType,
		Metadata:        interaction.Metadata,
		Timestamp:       time.Now(),
	}

	// TODO：后续改为protobuf序列化
	eventData, err := json.Marshal(event)
	if err != nil {
		s.logger.Error(ctx, "Failed to marshal interaction event",
			logger.F("error", err.Error()),
			logger.F("event", event))
		return
	}

	// 异步发送事件
	go func() {
		topic := "interaction-events"
		key := fmt.Sprintf("%d:%d:%s", event.UserID, event.ObjectID, event.InteractionType)

		if err := s.kafka.SendMessage(topic, []byte(key), eventData); err != nil {
			s.logger.Error(context.Background(), "Failed to send interaction event",
				logger.F("error", err.Error()),
				logger.F("topic", topic),
				logger.F("key", key))
		}
	}()
}

// GetInteractionSummary 获取对象的互动汇总（包含用户状态）
func (s *Service) GetInteractionSummary(ctx context.Context, objectID, userID int64, objectType string) (*model.InteractionSummary, error) {
	// 获取统计数据
	stats, err := s.GetObjectStats(ctx, objectID, objectType)
	if err != nil {
		return nil, err
	}

	summary := &model.InteractionSummary{
		ObjectID:      objectID,
		ObjectType:    objectType,
		LikeCount:     stats.LikeCount,
		FavoriteCount: stats.FavoriteCount,
		ShareCount:    stats.ShareCount,
		RepostCount:   stats.RepostCount,
	}

	// 如果提供了用户ID，获取用户的互动状态
	if userID > 0 {
		userInteractions := make(map[string]bool)

		for _, interactionType := range model.ValidInteractionTypes {
			hasInteraction, _, err := s.CheckInteraction(ctx, userID, objectID, objectType, interactionType)
			if err != nil {
				s.logger.Error(ctx, "Failed to check user interaction",
					logger.F("error", err.Error()),
					logger.F("userID", userID),
					logger.F("objectID", objectID),
					logger.F("interactionType", interactionType))
				continue
			}
			userInteractions[interactionType] = hasInteraction
		}

		summary.UserInteractions = userInteractions
	}

	return summary, nil
}

// BatchGetInteractionSummary 批量获取互动汇总
func (s *Service) BatchGetInteractionSummary(ctx context.Context, objectIDs []int64, userID int64, objectType string) ([]*model.InteractionSummary, error) {
	if len(objectIDs) == 0 {
		return []*model.InteractionSummary{}, nil
	}
	if len(objectIDs) > model.MaxBatchSize {
		return nil, fmt.Errorf("批量查询数量超过限制: %d", model.MaxBatchSize)
	}

	// 批量获取统计数据
	statsList, err := s.GetBatchObjectStats(ctx, objectIDs, objectType)
	if err != nil {
		return nil, err
	}

	// 创建统计数据映射
	statsMap := make(map[int64]*model.InteractionStats)
	for _, stats := range statsList {
		statsMap[stats.ObjectID] = stats
	}

	var summaries []*model.InteractionSummary
	for _, objectID := range objectIDs {
		stats, exists := statsMap[objectID]
		if !exists {
			// 如果没有统计数据，创建空的
			stats = &model.InteractionStats{
				ObjectID:   objectID,
				ObjectType: objectType,
			}
		}

		summary := &model.InteractionSummary{
			ObjectID:      objectID,
			ObjectType:    objectType,
			LikeCount:     stats.LikeCount,
			FavoriteCount: stats.FavoriteCount,
			ShareCount:    stats.ShareCount,
			RepostCount:   stats.RepostCount,
		}

		// 如果提供了用户ID，获取用户的互动状态
		if userID > 0 {
			userInteractions := make(map[string]bool)

			for _, interactionType := range model.ValidInteractionTypes {
				hasInteraction, _, err := s.CheckInteraction(ctx, userID, objectID, objectType, interactionType)
				if err != nil {
					s.logger.Error(ctx, "Failed to check user interaction in batch",
						logger.F("error", err.Error()),
						logger.F("userID", userID),
						logger.F("objectID", objectID),
						logger.F("interactionType", interactionType))
					continue
				}
				userInteractions[interactionType] = hasInteraction
			}

			summary.UserInteractions = userInteractions
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}
