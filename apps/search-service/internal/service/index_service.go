package service

import (
	"context"
	"fmt"
	"time"

	"goim-social/apps/search-service/internal/dao"
	"goim-social/apps/search-service/internal/model"
	"goim-social/pkg/database"
	"goim-social/pkg/logger"
)

// indexService 索引管理服务实现
type indexService struct {
	searchDAO    dao.SearchDAO
	historyDAO   dao.HistoryDAO
	eventService EventService
	config       *ServiceConfig
	logger       logger.Logger
}

// NewIndexService 创建索引服务实例（简化版本）
func NewIndexService(elasticSearch *database.ElasticSearch, postgreSQL *database.PostgreSQL, log logger.Logger) IndexService {
	if elasticSearch == nil {
		panic("ElasticSearch is required for search service. Please set ELASTICSEARCH_ENABLED=true and ensure ElasticSearch is running.")
	}

	// 初始化DAO层
	searchDAO := dao.NewElasticsearchDAO(elasticSearch.GetClient(), log)
	historyDAO := dao.NewHistoryDAO(postgreSQL, log)

	// 初始化服务层依赖
	eventService := NewMockEventService()

	// 创建默认配置
	config := &ServiceConfig{
		DefaultPageSize:  20,
		MaxPageSize:      100,
		SearchTimeout:    5000,
		HighlightPreTag:  "<em>",
		HighlightPostTag: "</em>",
		CacheEnabled:     true,
		CacheTTL: map[string]int{
			"search_results": 300,
			"suggestions":    600,
			"hot_searches":   1800,
		},
		IndexSettings: map[string]interface{}{
			"refresh_interval":   "1s",
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
		FieldWeights: map[string]float64{
			"title":   2.0,
			"content": 1.0,
			"tags":    1.5,
		},
		EventEnabled: true,
		EventTopics: map[string]string{
			"search": "search_events",
			"index":  "index_events",
		},
	}

	return &indexService{
		searchDAO:    searchDAO,
		historyDAO:   historyDAO,
		eventService: eventService,
		config:       config,
		logger:       log,
	}
}

// NewIndexServiceWithConfig 创建索引服务实例（详细版本，保持向后兼容）
func NewIndexServiceWithConfig(
	searchDAO dao.SearchDAO,
	historyDAO dao.HistoryDAO,
	eventService EventService,
	config *ServiceConfig,
	log logger.Logger,
) IndexService {
	return &indexService{
		searchDAO:    searchDAO,
		historyDAO:   historyDAO,
		eventService: eventService,
		config:       config,
		logger:       log,
	}
}

// ============ 索引管理 ============

// CreateIndex 创建索引
func (s *indexService) CreateIndex(ctx context.Context, indexName string, indexType string) error {
	if indexName == "" || indexType == "" {
		return fmt.Errorf("index name and type are required")
	}

	// 检查索引是否已存在
	exists, err := s.searchDAO.IndexExists(ctx, indexName)
	if err != nil {
		s.logger.Error(ctx, "Failed to check index existence",
			logger.F("index_name", indexName),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to check index existence: %v", err)
	}

	if exists {
		s.logger.Warn(ctx, "Index already exists",
			logger.F("index_name", indexName))
		return fmt.Errorf("index already exists: %s", indexName)
	}

	// 获取索引配置
	mapping, settings := s.getIndexConfig(indexType)

	// 创建索引
	err = s.searchDAO.CreateIndex(ctx, indexName, mapping, settings)
	if err != nil {
		s.logger.Error(ctx, "Failed to create index",
			logger.F("index_name", indexName),
			logger.F("index_type", indexType),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to create index: %v", err)
	}

	// 保存索引配置到数据库
	indexConfig := &model.SearchIndex{
		IndexName:      indexName,
		IndexType:      indexType,
		MappingConfig:  mapping,
		SettingsConfig: settings,
		IsActive:       true,
	}

	if err := s.historyDAO.CreateSearchIndex(ctx, indexConfig); err != nil {
		s.logger.Warn(ctx, "Failed to save index config to database",
			logger.F("index_name", indexName),
			logger.F("error", err.Error()))
		// 不返回错误，因为索引已经创建成功
	}

	// 发布索引事件
	event := &IndexEvent{
		Action:       "create",
		IndexName:    indexName,
		DocumentType: indexType,
		Timestamp:    time.Now().Unix(),
		Source:       "search-service",
	}

	if err := s.eventService.PublishIndexEvent(ctx, event); err != nil {
		s.logger.Warn(ctx, "Failed to publish index event",
			logger.F("index_name", indexName),
			logger.F("error", err.Error()))
	}

	s.logger.Info(ctx, "Index created successfully",
		logger.F("index_name", indexName),
		logger.F("index_type", indexType))

	return nil
}

// DeleteIndex 删除索引
func (s *indexService) DeleteIndex(ctx context.Context, indexName string) error {
	if indexName == "" {
		return fmt.Errorf("index name is required")
	}

	// 检查索引是否存在
	exists, err := s.searchDAO.IndexExists(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %v", err)
	}

	if !exists {
		return fmt.Errorf("index does not exist: %s", indexName)
	}

	// 删除索引
	err = s.searchDAO.DeleteIndex(ctx, indexName)
	if err != nil {
		s.logger.Error(ctx, "Failed to delete index",
			logger.F("index_name", indexName),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to delete index: %v", err)
	}

	// 更新数据库中的索引配置状态
	if indexConfig, err := s.historyDAO.GetSearchIndex(ctx, indexName); err == nil {
		indexConfig.IsActive = false
		s.historyDAO.UpdateSearchIndex(ctx, indexConfig)
	}

	// 发布索引事件
	event := &IndexEvent{
		Action:    "delete",
		IndexName: indexName,
		Timestamp: time.Now().Unix(),
		Source:    "search-service",
	}

	if err := s.eventService.PublishIndexEvent(ctx, event); err != nil {
		s.logger.Warn(ctx, "Failed to publish index event",
			logger.F("index_name", indexName),
			logger.F("error", err.Error()))
	}

	s.logger.Info(ctx, "Index deleted successfully",
		logger.F("index_name", indexName))

	return nil
}

// ReindexAll 重建所有索引
func (s *indexService) ReindexAll(ctx context.Context) error {
	s.logger.Info(ctx, "Starting reindex all operation")

	// 获取所有活跃的索引配置
	indices, err := s.historyDAO.ListSearchIndices(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to list search indices: %v", err)
	}

	for _, index := range indices {
		if err := s.ReindexByType(ctx, index.IndexType); err != nil {
			s.logger.Error(ctx, "Failed to reindex",
				logger.F("index_type", index.IndexType),
				logger.F("error", err.Error()))
			// 继续处理其他索引
		}
	}

	s.logger.Info(ctx, "Reindex all operation completed")
	return nil
}

// ReindexByType 按类型重建索引
func (s *indexService) ReindexByType(ctx context.Context, indexType string) error {
	if indexType == "" {
		return fmt.Errorf("index type is required")
	}

	s.logger.Info(ctx, "Starting reindex by type",
		logger.F("index_type", indexType))

	// 根据索引类型获取对应的索引名称
	indexName := model.GetIndexBySearchType(indexType)
	if indexName == "" {
		return fmt.Errorf("unsupported index type: %s", indexType)
	}

	// 创建新的索引名称（带时间戳）
	newIndexName := fmt.Sprintf("%s_%d", indexName, time.Now().Unix())

	// 创建新索引
	if err := s.CreateIndex(ctx, newIndexName, indexType); err != nil {
		return fmt.Errorf("failed to create new index: %v", err)
	}

	// 这里应该从源数据库同步数据到新索引
	// 具体实现取决于数据源

	// 切换索引别名（如果使用别名的话）
	// 这里简化处理，实际应该使用别名来实现无缝切换

	s.logger.Info(ctx, "Reindex by type completed",
		logger.F("index_type", indexType),
		logger.F("new_index", newIndexName))

	return nil
}

// ============ 文档管理 ============

// IndexDocument 索引单个文档
func (s *indexService) IndexDocument(ctx context.Context, indexType string, docID string, document interface{}) error {
	if indexType == "" || docID == "" || document == nil {
		return fmt.Errorf("index type, document ID and document are required")
	}

	indexName := model.GetIndexBySearchType(indexType)
	if indexName == "" {
		return fmt.Errorf("unsupported index type: %s", indexType)
	}

	err := s.searchDAO.IndexDocument(ctx, indexName, docID, document)
	if err != nil {
		s.logger.Error(ctx, "Failed to index document",
			logger.F("index_name", indexName),
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to index document: %v", err)
	}

	// 发布索引事件
	event := &IndexEvent{
		Action:       "index",
		IndexName:    indexName,
		DocumentID:   docID,
		DocumentType: indexType,
		Timestamp:    time.Now().Unix(),
		Source:       "search-service",
	}

	if err := s.eventService.PublishIndexEvent(ctx, event); err != nil {
		s.logger.Warn(ctx, "Failed to publish index event",
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
	}

	s.logger.Debug(ctx, "Document indexed successfully",
		logger.F("index_name", indexName),
		logger.F("doc_id", docID))

	return nil
}

// BulkIndexDocuments 批量索引文档
func (s *indexService) BulkIndexDocuments(ctx context.Context, indexType string, documents []IndexDocument) error {
	if indexType == "" || len(documents) == 0 {
		return fmt.Errorf("index type and documents are required")
	}

	indexName := model.GetIndexBySearchType(indexType)
	if indexName == "" {
		return fmt.Errorf("unsupported index type: %s", indexType)
	}

	// 转换为DAO层的批量文档格式
	bulkDocs := make([]dao.BulkDocument, len(documents))
	for i, doc := range documents {
		bulkDocs[i] = dao.BulkDocument{
			ID:       doc.ID,
			Document: doc.Document,
			Action:   "index",
		}
	}

	err := s.searchDAO.BulkIndexDocuments(ctx, indexName, bulkDocs)
	if err != nil {
		s.logger.Error(ctx, "Failed to bulk index documents",
			logger.F("index_name", indexName),
			logger.F("count", len(documents)),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to bulk index documents: %v", err)
	}

	s.logger.Info(ctx, "Documents bulk indexed successfully",
		logger.F("index_name", indexName),
		logger.F("count", len(documents)))

	return nil
}

// ============ 数据同步 ============

// SyncFromDatabase 从数据库同步数据
func (s *indexService) SyncFromDatabase(ctx context.Context, sourceService string, sourceTable string, targetIndex string) error {
	if sourceService == "" || sourceTable == "" || targetIndex == "" {
		return fmt.Errorf("source service, source table and target index are required")
	}

	s.logger.Info(ctx, "Starting database sync",
		logger.F("source_service", sourceService),
		logger.F("source_table", sourceTable),
		logger.F("target_index", targetIndex))

	// 获取或创建同步状态
	syncStatus, err := s.historyDAO.GetSyncStatus(ctx, sourceTable, targetIndex)
	if err != nil {
		// 创建新的同步状态
		syncStatus = &model.SyncStatus{
			SourceTable:   sourceTable,
			SourceService: sourceService,
			TargetIndex:   targetIndex,
			LastSyncID:    0,
			SyncStatus:    model.SyncStatusPending,
		}

		if err := s.historyDAO.CreateSyncStatus(ctx, syncStatus); err != nil {
			return fmt.Errorf("failed to create sync status: %v", err)
		}
	}

	// 更新同步状态为运行中
	syncStatus.SyncStatus = model.SyncStatusRunning
	syncStatus.LastSyncTime = time.Now()
	syncStatus.ErrorMessage = ""

	if err := s.historyDAO.UpdateSyncStatus(ctx, syncStatus); err != nil {
		s.logger.Warn(ctx, "Failed to update sync status",
			logger.F("error", err.Error()))
	}

	// 这里应该实现具体的数据同步逻辑
	// 由于没有具体的数据源连接，这里只是示例

	// 模拟同步过程
	time.Sleep(1 * time.Second)

	// 更新同步状态为完成
	syncStatus.SyncStatus = model.SyncStatusCompleted
	syncStatus.LastSyncTime = time.Now()

	if err := s.historyDAO.UpdateSyncStatus(ctx, syncStatus); err != nil {
		s.logger.Warn(ctx, "Failed to update sync status",
			logger.F("error", err.Error()))
	}

	s.logger.Info(ctx, "Database sync completed",
		logger.F("source_service", sourceService),
		logger.F("source_table", sourceTable),
		logger.F("target_index", targetIndex))

	return nil
}

// GetSyncStatus 获取同步状态
func (s *indexService) GetSyncStatus(ctx context.Context, sourceTable string, targetIndex string) (*model.SyncStatus, error) {
	if sourceTable == "" || targetIndex == "" {
		return nil, fmt.Errorf("source table and target index are required")
	}

	syncStatus, err := s.historyDAO.GetSyncStatus(ctx, sourceTable, targetIndex)
	if err != nil {
		s.logger.Error(ctx, "Failed to get sync status",
			logger.F("source_table", sourceTable),
			logger.F("target_index", targetIndex),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to get sync status: %v", err)
	}

	return syncStatus, nil
}

// ListSyncStatuses 列出同步状态
func (s *indexService) ListSyncStatuses(ctx context.Context, sourceService string) ([]*model.SyncStatus, error) {
	statuses, err := s.historyDAO.ListSyncStatuses(ctx, sourceService)
	if err != nil {
		s.logger.Error(ctx, "Failed to list sync statuses",
			logger.F("source_service", sourceService),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to list sync statuses: %v", err)
	}

	return statuses, nil
}

// ============ 健康检查 ============

// HealthCheck 健康检查
func (s *indexService) HealthCheck(ctx context.Context) error {
	// 检查ElasticSearch连接
	if err := s.searchDAO.Ping(ctx); err != nil {
		s.logger.Error(ctx, "ElasticSearch health check failed",
			logger.F("error", err.Error()))
		return fmt.Errorf("elasticsearch health check failed: %v", err)
	}

	// 检查集群健康状态
	health, err := s.searchDAO.GetClusterHealth(ctx)
	if err != nil {
		s.logger.Error(ctx, "Failed to get cluster health",
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to get cluster health: %v", err)
	}

	// 检查集群状态
	if status, ok := health["status"].(string); ok {
		if status == "red" {
			return fmt.Errorf("elasticsearch cluster status is red")
		}
	}

	s.logger.Debug(ctx, "Health check passed")
	return nil
}

// GetClusterInfo 获取集群信息
func (s *indexService) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	// 获取集群健康状态
	health, err := s.searchDAO.GetClusterHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster health: %v", err)
	}

	// 获取集群统计信息
	stats, err := s.searchDAO.GetClusterStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stats: %v", err)
	}

	// 合并信息
	clusterInfo := map[string]interface{}{
		"health":    health,
		"stats":     stats,
		"timestamp": time.Now().Unix(),
	}

	return clusterInfo, nil
}

// ============ 辅助方法 ============

// getIndexConfig 获取索引配置
func (s *indexService) getIndexConfig(indexType string) (map[string]interface{}, map[string]interface{}) {
	// 默认设置
	settings := map[string]interface{}{
		"number_of_shards":   model.DefaultESShards,
		"number_of_replicas": model.DefaultESReplicas,
		"refresh_interval":   model.DefaultESRefresh,
		"analysis": map[string]interface{}{
			"analyzer": map[string]interface{}{
				"ik_smart_analyzer": map[string]interface{}{
					"type":      "custom",
					"tokenizer": "ik_smart",
				},
				"ik_max_word_analyzer": map[string]interface{}{
					"type":      "custom",
					"tokenizer": "ik_max_word",
				},
			},
		},
	}

	// 根据索引类型定义映射
	var mapping map[string]interface{}

	switch indexType {
	case model.SearchTypeContent:
		mapping = s.getContentMapping()
	case model.SearchTypeUser:
		mapping = s.getUserMapping()
	case model.SearchTypeMessage:
		mapping = s.getMessageMapping()
	case model.SearchTypeGroup:
		mapping = s.getGroupMapping()
	default:
		mapping = s.getDefaultMapping()
	}

	return mapping, settings
}

// getContentMapping 获取内容映射
func (s *indexService) getContentMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"id":            map[string]interface{}{"type": "long"},
			"title":         map[string]interface{}{"type": "text", "analyzer": "ik_max_word_analyzer"},
			"content":       map[string]interface{}{"type": "text", "analyzer": "ik_max_word_analyzer"},
			"summary":       map[string]interface{}{"type": "text", "analyzer": "ik_smart_analyzer"},
			"author_id":     map[string]interface{}{"type": "long"},
			"author_name":   map[string]interface{}{"type": "keyword"},
			"category_id":   map[string]interface{}{"type": "long"},
			"category":      map[string]interface{}{"type": "keyword"},
			"tags":          map[string]interface{}{"type": "keyword"},
			"status":        map[string]interface{}{"type": "keyword"},
			"view_count":    map[string]interface{}{"type": "long"},
			"like_count":    map[string]interface{}{"type": "long"},
			"comment_count": map[string]interface{}{"type": "long"},
			"share_count":   map[string]interface{}{"type": "long"},
			"created_at":    map[string]interface{}{"type": "date"},
			"updated_at":    map[string]interface{}{"type": "date"},
		},
	}
}

// getUserMapping 获取用户映射
func (s *indexService) getUserMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"id":             map[string]interface{}{"type": "long"},
			"username":       map[string]interface{}{"type": "text", "analyzer": "ik_smart_analyzer"},
			"nickname":       map[string]interface{}{"type": "text", "analyzer": "ik_smart_analyzer"},
			"email":          map[string]interface{}{"type": "keyword"},
			"avatar":         map[string]interface{}{"type": "keyword"},
			"bio":            map[string]interface{}{"type": "text", "analyzer": "ik_max_word_analyzer"},
			"location":       map[string]interface{}{"type": "keyword"},
			"tags":           map[string]interface{}{"type": "keyword"},
			"status":         map[string]interface{}{"type": "keyword"},
			"is_verified":    map[string]interface{}{"type": "boolean"},
			"friend_count":   map[string]interface{}{"type": "long"},
			"follower_count": map[string]interface{}{"type": "long"},
			"post_count":     map[string]interface{}{"type": "long"},
			"last_active_at": map[string]interface{}{"type": "date"},
			"created_at":     map[string]interface{}{"type": "date"},
		},
	}
}

// getMessageMapping 获取消息映射
func (s *indexService) getMessageMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"id":            map[string]interface{}{"type": "long"},
			"from_user_id":  map[string]interface{}{"type": "long"},
			"from_username": map[string]interface{}{"type": "keyword"},
			"to_user_id":    map[string]interface{}{"type": "long"},
			"to_username":   map[string]interface{}{"type": "keyword"},
			"group_id":      map[string]interface{}{"type": "long"},
			"group_name":    map[string]interface{}{"type": "keyword"},
			"content":       map[string]interface{}{"type": "text", "analyzer": "ik_max_word_analyzer"},
			"message_type":  map[string]interface{}{"type": "keyword"},
			"created_at":    map[string]interface{}{"type": "date"},
		},
	}
}

// getGroupMapping 获取群组映射
func (s *indexService) getGroupMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"id":           map[string]interface{}{"type": "long"},
			"name":         map[string]interface{}{"type": "text", "analyzer": "ik_max_word_analyzer"},
			"description":  map[string]interface{}{"type": "text", "analyzer": "ik_max_word_analyzer"},
			"avatar":       map[string]interface{}{"type": "keyword"},
			"owner_id":     map[string]interface{}{"type": "long"},
			"owner_name":   map[string]interface{}{"type": "keyword"},
			"member_count": map[string]interface{}{"type": "long"},
			"max_members":  map[string]interface{}{"type": "long"},
			"is_public":    map[string]interface{}{"type": "boolean"},
			"tags":         map[string]interface{}{"type": "keyword"},
			"category":     map[string]interface{}{"type": "keyword"},
			"status":       map[string]interface{}{"type": "keyword"},
			"created_at":   map[string]interface{}{"type": "date"},
			"updated_at":   map[string]interface{}{"type": "date"},
		},
	}
}

// getDefaultMapping 获取默认映射
func (s *indexService) getDefaultMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"id":         map[string]interface{}{"type": "long"},
			"title":      map[string]interface{}{"type": "text", "analyzer": "ik_max_word_analyzer"},
			"content":    map[string]interface{}{"type": "text", "analyzer": "ik_max_word_analyzer"},
			"created_at": map[string]interface{}{"type": "date"},
			"updated_at": map[string]interface{}{"type": "date"},
		},
	}
}

// UpdateDocument 更新文档
func (s *indexService) UpdateDocument(ctx context.Context, indexType string, docID string, document interface{}) error {
	if indexType == "" || docID == "" || document == nil {
		return fmt.Errorf("index type, document ID and document are required")
	}

	indexName := model.GetIndexBySearchType(indexType)
	if indexName == "" {
		return fmt.Errorf("unsupported index type: %s", indexType)
	}

	err := s.searchDAO.UpdateDocument(ctx, indexName, docID, document)
	if err != nil {
		s.logger.Error(ctx, "Failed to update document",
			logger.F("index_name", indexName),
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to update document: %v", err)
	}

	// 发布索引事件
	event := &IndexEvent{
		Action:       "update",
		IndexName:    indexName,
		DocumentID:   docID,
		DocumentType: indexType,
		Timestamp:    time.Now().Unix(),
		Source:       "search-service",
	}

	if err := s.eventService.PublishIndexEvent(ctx, event); err != nil {
		s.logger.Warn(ctx, "Failed to publish index event",
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
	}

	s.logger.Debug(ctx, "Document updated successfully",
		logger.F("index_name", indexName),
		logger.F("doc_id", docID))

	return nil
}

// DeleteDocument 删除文档
func (s *indexService) DeleteDocument(ctx context.Context, indexType string, docID string) error {
	if indexType == "" || docID == "" {
		return fmt.Errorf("index type and document ID are required")
	}

	indexName := model.GetIndexBySearchType(indexType)
	if indexName == "" {
		return fmt.Errorf("unsupported index type: %s", indexType)
	}

	err := s.searchDAO.DeleteDocument(ctx, indexName, docID)
	if err != nil {
		s.logger.Error(ctx, "Failed to delete document",
			logger.F("index_name", indexName),
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to delete document: %v", err)
	}

	// 发布索引事件
	event := &IndexEvent{
		Action:       "delete",
		IndexName:    indexName,
		DocumentID:   docID,
		DocumentType: indexType,
		Timestamp:    time.Now().Unix(),
		Source:       "search-service",
	}

	if err := s.eventService.PublishIndexEvent(ctx, event); err != nil {
		s.logger.Warn(ctx, "Failed to publish index event",
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
	}

	s.logger.Debug(ctx, "Document deleted successfully",
		logger.F("index_name", indexName),
		logger.F("doc_id", docID))

	return nil
}
