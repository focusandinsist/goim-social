package dao

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"goim-social/apps/search-service/internal/model"
	"goim-social/pkg/logger"
)

// elasticsearchDAO ElasticSearch数据访问对象
type elasticsearchDAO struct {
	client *elasticsearch.Client
	logger logger.Logger
}

// NewElasticsearchDAO 创建ElasticSearch DAO实例
func NewElasticsearchDAO(client *elasticsearch.Client, log logger.Logger) SearchDAO {
	return &elasticsearchDAO{
		client: client,
		logger: log,
	}
}

// ============ 索引管理 ============

// CreateIndex 创建索引
func (d *elasticsearchDAO) CreateIndex(ctx context.Context, indexName string, mapping map[string]interface{}, settings map[string]interface{}) error {
	// 构建索引配置
	indexConfig := map[string]interface{}{
		"mappings": mapping,
		"settings": settings,
	}

	configJSON, err := json.Marshal(indexConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal index config: %v", err)
	}

	// 创建索引请求
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  bytes.NewReader(configJSON),
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		d.logger.Error(ctx, "Failed to create index",
			logger.F("index", indexName),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to create index: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	d.logger.Info(ctx, "Index created successfully",
		logger.F("index", indexName))
	return nil
}

// DeleteIndex 删除索引
func (d *elasticsearchDAO) DeleteIndex(ctx context.Context, indexName string) error {
	req := esapi.IndicesDeleteRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("failed to delete index: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to delete index: %s", res.String())
	}

	d.logger.Info(ctx, "Index deleted successfully",
		logger.F("index", indexName))
	return nil
}

// IndexExists 检查索引是否存在
func (d *elasticsearchDAO) IndexExists(ctx context.Context, indexName string) (bool, error) {
	req := esapi.IndicesExistsRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return false, fmt.Errorf("failed to check index existence: %v", err)
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

// GetIndexMapping 获取索引映射
func (d *elasticsearchDAO) GetIndexMapping(ctx context.Context, indexName string) (map[string]interface{}, error) {
	req := esapi.IndicesGetMappingRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get index mapping: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to get index mapping: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode mapping response: %v", err)
	}

	return result, nil
}

// UpdateIndexSettings 更新索引设置
func (d *elasticsearchDAO) UpdateIndexSettings(ctx context.Context, indexName string, settings map[string]interface{}) error {
	settingsJSON, err := json.Marshal(map[string]interface{}{
		"settings": settings,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %v", err)
	}

	req := esapi.IndicesPutSettingsRequest{
		Index: []string{indexName},
		Body:  bytes.NewReader(settingsJSON),
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("failed to update index settings: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to update index settings: %s", res.String())
	}

	return nil
}

// ============ 文档操作 ============

// IndexDocument 索引文档
func (d *elasticsearchDAO) IndexDocument(ctx context.Context, indexName, docID string, document interface{}) error {
	docJSON, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}

	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: docID,
		Body:       bytes.NewReader(docJSON),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		d.logger.Error(ctx, "Failed to index document",
			logger.F("index", indexName),
			logger.F("doc_id", docID),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to index document: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index document: %s", res.String())
	}

	d.logger.Debug(ctx, "Document indexed successfully",
		logger.F("index", indexName),
		logger.F("doc_id", docID))
	return nil
}

// BulkIndexDocuments 批量索引文档
func (d *elasticsearchDAO) BulkIndexDocuments(ctx context.Context, indexName string, documents []BulkDocument) error {
	if len(documents) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, doc := range documents {
		// 构建操作元数据
		meta := map[string]interface{}{
			doc.Action: map[string]interface{}{
				"_index": indexName,
				"_id":    doc.ID,
			},
		}
		metaJSON, _ := json.Marshal(meta)
		buf.Write(metaJSON)
		buf.WriteByte('\n')

		// 如果不是删除操作，添加文档内容
		if doc.Action != "delete" {
			docJSON, err := json.Marshal(doc.Document)
			if err != nil {
				return fmt.Errorf("failed to marshal document %s: %v", doc.ID, err)
			}
			buf.Write(docJSON)
			buf.WriteByte('\n')
		}
	}

	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "true",
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		d.logger.Error(ctx, "Failed to bulk index documents",
			logger.F("index", indexName),
			logger.F("count", len(documents)),
			logger.F("error", err.Error()))
		return fmt.Errorf("failed to bulk index documents: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to bulk index documents: %s", res.String())
	}

	// 解析响应检查错误
	var bulkResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&bulkResponse); err != nil {
		return fmt.Errorf("failed to decode bulk response: %v", err)
	}

	if errors, ok := bulkResponse["errors"].(bool); ok && errors {
		d.logger.Warn(ctx, "Bulk operation had errors",
			logger.F("index", indexName),
			logger.F("response", bulkResponse))
	}

	d.logger.Info(ctx, "Bulk index completed",
		logger.F("index", indexName),
		logger.F("count", len(documents)))
	return nil
}

// UpdateDocument 更新文档
func (d *elasticsearchDAO) UpdateDocument(ctx context.Context, indexName, docID string, document interface{}) error {
	updateDoc := map[string]interface{}{
		"doc": document,
	}

	docJSON, err := json.Marshal(updateDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal update document: %v", err)
	}

	req := esapi.UpdateRequest{
		Index:      indexName,
		DocumentID: docID,
		Body:       bytes.NewReader(docJSON),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to update document: %s", res.String())
	}

	return nil
}

// DeleteDocument 删除文档
func (d *elasticsearchDAO) DeleteDocument(ctx context.Context, indexName, docID string) error {
	req := esapi.DeleteRequest{
		Index:      indexName,
		DocumentID: docID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("failed to delete document: %s", res.String())
	}

	return nil
}

// GetDocument 获取文档
func (d *elasticsearchDAO) GetDocument(ctx context.Context, indexName, docID string) (map[string]interface{}, error) {
	req := esapi.GetRequest{
		Index:      indexName,
		DocumentID: docID,
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to get document: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode document: %v", err)
	}

	if source, ok := result["_source"].(map[string]interface{}); ok {
		return source, nil
	}

	return result, nil
}

// ============ 搜索操作 ============

// Search 通用搜索
func (d *elasticsearchDAO) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	queryJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %v", err)
	}

	searchReq := esapi.SearchRequest{
		Index: []string{req.Index},
		Body:  bytes.NewReader(queryJSON),
	}

	res, err := searchReq.Do(ctx, d.client)
	if err != nil {
		d.logger.Error(ctx, "Failed to execute search",
			logger.F("index", req.Index),
			logger.F("error", err.Error()))
		return nil, fmt.Errorf("failed to execute search: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search failed: %s", res.String())
	}

	var response SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %v", err)
	}

	return &response, nil
}

// SearchContent 内容搜索
func (d *elasticsearchDAO) SearchContent(ctx context.Context, req *model.SearchRequest) ([]*model.ContentSearchResult, int64, error) {
	// 构建搜索查询
	query := d.buildContentSearchQuery(req)

	searchReq := &SearchRequest{
		Index: model.IndexContent,
		Query: query,
		From:  (req.Page - 1) * req.PageSize,
		Size:  req.PageSize,
	}

	// 添加排序
	if req.SortBy != "" {
		searchReq.Sort = d.buildSortQuery(req.SortBy, req.SortOrder)
	}

	// 添加高亮
	if req.Highlight {
		searchReq.Highlight = d.buildHighlightQuery([]string{"title", "content", "summary"})
	}

	// 执行搜索
	response, err := d.Search(ctx, searchReq)
	if err != nil {
		return nil, 0, err
	}

	// 转换结果
	results := make([]*model.ContentSearchResult, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		result, err := d.convertToContentResult(hit)
		if err != nil {
			d.logger.Warn(ctx, "Failed to convert content result",
				logger.F("doc_id", hit.ID),
				logger.F("error", err.Error()))
			continue
		}
		results = append(results, result)
	}

	return results, response.Hits.Total.Value, nil
}

// SearchUsers 用户搜索
func (d *elasticsearchDAO) SearchUsers(ctx context.Context, req *model.SearchRequest) ([]*model.UserSearchResult, int64, error) {
	// 构建搜索查询
	query := d.buildUserSearchQuery(req)

	searchReq := &SearchRequest{
		Index: model.IndexUser,
		Query: query,
		From:  (req.Page - 1) * req.PageSize,
		Size:  req.PageSize,
	}

	// 添加排序
	if req.SortBy != "" {
		searchReq.Sort = d.buildSortQuery(req.SortBy, req.SortOrder)
	}

	// 添加高亮
	if req.Highlight {
		searchReq.Highlight = d.buildHighlightQuery([]string{"username", "nickname", "bio"})
	}

	// 执行搜索
	response, err := d.Search(ctx, searchReq)
	if err != nil {
		return nil, 0, err
	}

	// 转换结果
	results := make([]*model.UserSearchResult, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		result, err := d.convertToUserResult(hit)
		if err != nil {
			d.logger.Warn(ctx, "Failed to convert user result",
				logger.F("doc_id", hit.ID),
				logger.F("error", err.Error()))
			continue
		}
		results = append(results, result)
	}

	return results, response.Hits.Total.Value, nil
}

// SearchMessages 消息搜索
func (d *elasticsearchDAO) SearchMessages(ctx context.Context, req *model.SearchRequest) ([]*model.MessageSearchResult, int64, error) {
	// 构建搜索查询
	query := d.buildMessageSearchQuery(req)

	searchReq := &SearchRequest{
		Index: model.IndexMessage,
		Query: query,
		From:  (req.Page - 1) * req.PageSize,
		Size:  req.PageSize,
	}

	// 添加排序
	if req.SortBy != "" {
		searchReq.Sort = d.buildSortQuery(req.SortBy, req.SortOrder)
	}

	// 添加高亮
	if req.Highlight {
		searchReq.Highlight = d.buildHighlightQuery([]string{"content"})
	}

	// 执行搜索
	response, err := d.Search(ctx, searchReq)
	if err != nil {
		return nil, 0, err
	}

	// 转换结果
	results := make([]*model.MessageSearchResult, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		result, err := d.convertToMessageResult(hit)
		if err != nil {
			d.logger.Warn(ctx, "Failed to convert message result",
				logger.F("doc_id", hit.ID),
				logger.F("error", err.Error()))
			continue
		}
		results = append(results, result)
	}

	return results, response.Hits.Total.Value, nil
}

// SearchGroups 群组搜索
func (d *elasticsearchDAO) SearchGroups(ctx context.Context, req *model.SearchRequest) ([]*model.GroupSearchResult, int64, error) {
	// 构建搜索查询
	query := d.buildGroupSearchQuery(req)

	searchReq := &SearchRequest{
		Index: model.IndexGroup,
		Query: query,
		From:  (req.Page - 1) * req.PageSize,
		Size:  req.PageSize,
	}

	// 添加排序
	if req.SortBy != "" {
		searchReq.Sort = d.buildSortQuery(req.SortBy, req.SortOrder)
	}

	// 添加高亮
	if req.Highlight {
		searchReq.Highlight = d.buildHighlightQuery([]string{"name", "description"})
	}

	// 执行搜索
	response, err := d.Search(ctx, searchReq)
	if err != nil {
		return nil, 0, err
	}

	// 转换结果
	results := make([]*model.GroupSearchResult, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		result, err := d.convertToGroupResult(hit)
		if err != nil {
			d.logger.Warn(ctx, "Failed to convert group result",
				logger.F("doc_id", hit.ID),
				logger.F("error", err.Error()))
			continue
		}
		results = append(results, result)
	}

	return results, response.Hits.Total.Value, nil
}

// MultiSearch 多类型搜索
func (d *elasticsearchDAO) MultiSearch(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	startTime := time.Now()

	// 构建多搜索请求
	var buf bytes.Buffer
	indices := []string{model.IndexContent, model.IndexUser, model.IndexMessage, model.IndexGroup}

	for _, index := range indices {
		// 搜索头部
		header := map[string]interface{}{
			"index": index,
		}
		headerJSON, _ := json.Marshal(header)
		buf.Write(headerJSON)
		buf.WriteByte('\n')

		// 搜索体
		query := d.buildMultiSearchQuery(req, index)
		searchBody := map[string]interface{}{
			"query": query,
			"from":  0,
			"size":  req.PageSize / 4, // 每个类型返回1/4的结果
		}

		if req.Highlight {
			searchBody["highlight"] = d.buildHighlightQuery(d.getHighlightFields(index))
		}

		bodyJSON, _ := json.Marshal(searchBody)
		buf.Write(bodyJSON)
		buf.WriteByte('\n')
	}

	// 执行多搜索
	msearchReq := esapi.MsearchRequest{
		Body: &buf,
	}

	res, err := msearchReq.Do(ctx, d.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute multi-search: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("multi-search failed: %s", res.String())
	}

	// 解析响应
	var msearchResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&msearchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode multi-search response: %v", err)
	}

	// 构建响应
	response := &model.SearchResponse{
		Query:    req.Query,
		Type:     req.Type,
		Page:     req.Page,
		PageSize: req.PageSize,
		Results:  make([]interface{}, 0),
		Duration: time.Since(startTime).Milliseconds(),
	}

	// 处理每个索引的结果
	if responses, ok := msearchResponse["responses"].([]interface{}); ok {
		var totalHits int64
		for i, resp := range responses {
			if respMap, ok := resp.(map[string]interface{}); ok {
				if hits, ok := respMap["hits"].(map[string]interface{}); ok {
					if total, ok := hits["total"].(map[string]interface{}); ok {
						if value, ok := total["value"].(float64); ok {
							totalHits += int64(value)
						}
					}

					if hitsList, ok := hits["hits"].([]interface{}); ok {
						indexName := indices[i]
						for _, hit := range hitsList {
							if hitMap, ok := hit.(map[string]interface{}); ok {
								result := d.convertMultiSearchResult(hitMap, indexName)
								if result != nil {
									response.Results = append(response.Results, result)
								}
							}
						}
					}
				}
			}
		}
		response.Total = totalHits
	}

	return response, nil
}

// GetSuggestions 获取搜索建议
func (d *elasticsearchDAO) GetSuggestions(ctx context.Context, query string, searchType string, limit int) ([]model.SearchSuggestion, error) {
	// 构建建议查询
	suggestQuery := map[string]interface{}{
		"suggest": map[string]interface{}{
			"text": query,
			"completion_suggest": map[string]interface{}{
				"completion": map[string]interface{}{
					"field": "suggest",
					"size":  limit,
				},
			},
		},
	}

	// 根据搜索类型选择索引
	index := model.GetIndexBySearchType(searchType)
	if index == "" {
		index = model.IndexContent // 默认使用内容索引
	}

	queryJSON, err := json.Marshal(suggestQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal suggest query: %v", err)
	}

	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  bytes.NewReader(queryJSON),
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute suggest query: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("suggest query failed: %s", res.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode suggest response: %v", err)
	}

	// 解析建议结果
	suggestions := make([]model.SearchSuggestion, 0)
	if suggest, ok := response["suggest"].(map[string]interface{}); ok {
		if completionSuggest, ok := suggest["completion_suggest"].([]interface{}); ok {
			for _, item := range completionSuggest {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if options, ok := itemMap["options"].([]interface{}); ok {
						for _, option := range options {
							if optionMap, ok := option.(map[string]interface{}); ok {
								suggestion := model.SearchSuggestion{
									Type:   model.SuggestionTypeCompletion,
									Source: model.SuggestionSourceAuto,
								}

								if text, ok := optionMap["text"].(string); ok {
									suggestion.Text = text
								}

								if score, ok := optionMap["_score"].(float64); ok {
									suggestion.Score = score
								}

								suggestions = append(suggestions, suggestion)
							}
						}
					}
				}
			}
		}
	}

	return suggestions, nil
}

// GetAutoComplete 获取自动完成建议
func (d *elasticsearchDAO) GetAutoComplete(ctx context.Context, req *model.AutoCompleteRequest) (*model.AutoCompleteResponse, error) {
	startTime := time.Now()

	suggestions, err := d.GetSuggestions(ctx, req.Query, req.Type, req.Limit)
	if err != nil {
		return nil, err
	}

	response := &model.AutoCompleteResponse{
		Query:       req.Query,
		Suggestions: suggestions,
		Duration:    time.Since(startTime).Milliseconds(),
	}

	return response, nil
}

// GetAggregations 获取聚合数据
func (d *elasticsearchDAO) GetAggregations(ctx context.Context, indexName string, aggs map[string]interface{}) (map[string]interface{}, error) {
	query := map[string]interface{}{
		"size": 0,
		"aggs": aggs,
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal aggregation query: %v", err)
	}

	req := esapi.SearchRequest{
		Index: []string{indexName},
		Body:  bytes.NewReader(queryJSON),
	}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation query: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("aggregation query failed: %s", res.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation response: %v", err)
	}

	if aggregations, ok := response["aggregations"].(map[string]interface{}); ok {
		return aggregations, nil
	}

	return make(map[string]interface{}), nil
}

// GetSearchStats 获取搜索统计
func (d *elasticsearchDAO) GetSearchStats(ctx context.Context, timeRange string) (*model.SearchStats, error) {
	// 这里可以实现基于ElasticSearch的搜索统计
	// 暂时返回空统计，实际实现需要根据具体需求来构建聚合查询
	stats := &model.SearchStats{
		TotalSearches:   0,
		UniqueUsers:     0,
		AvgResponseTime: 0,
		TopQueries:      make([]model.QueryStat, 0),
		SearchesByType:  make(map[string]int64),
		SearchesByHour:  make(map[string]int64),
		CacheHitRate:    0,
	}

	return stats, nil
}

// Ping 检查ElasticSearch连接
func (d *elasticsearchDAO) Ping(ctx context.Context) error {
	req := esapi.PingRequest{}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return fmt.Errorf("failed to ping elasticsearch: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch ping failed: %s", res.String())
	}

	return nil
}

// GetClusterHealth 获取集群健康状态
func (d *elasticsearchDAO) GetClusterHealth(ctx context.Context) (map[string]interface{}, error) {
	req := esapi.ClusterHealthRequest{}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster health: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("get cluster health failed: %s", res.String())
	}

	var health map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode cluster health: %v", err)
	}

	return health, nil
}

// GetClusterStats 获取集群统计信息
func (d *elasticsearchDAO) GetClusterStats(ctx context.Context) (map[string]interface{}, error) {
	req := esapi.ClusterStatsRequest{}

	res, err := req.Do(ctx, d.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stats: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("get cluster stats failed: %s", res.String())
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode cluster stats: %v", err)
	}

	return stats, nil
}

// generateQueryHash 生成查询哈希
func (d *elasticsearchDAO) generateQueryHash(query string, searchType string, filters map[string]string) string {
	data := fmt.Sprintf("%s:%s:%v", query, searchType, filters)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}
