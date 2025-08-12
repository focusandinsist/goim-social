package converter

import (
	"goim-social/api/rest"
	"goim-social/apps/search-service/internal/model"
)

// Converter 转换器
type Converter struct{}

// NewConverter 创建转换器
func NewConverter() *Converter {
	return &Converter{}
}

// ============ 请求转换 ============

// SearchRequestFromProto 从proto转换搜索请求
func (c *Converter) SearchRequestFromProto(req *rest.SearchRequest) *model.SearchRequest {
	return &model.SearchRequest{
		Query:     req.Query,
		Type:      req.Type,
		Filters:   req.Filters,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		Highlight: req.Highlight,
		UserID:    req.UserId,
	}
}

// SearchRequestToProto 转换搜索请求到proto
func (c *Converter) SearchRequestToProto(req *model.SearchRequest) *rest.SearchRequest {
	return &rest.SearchRequest{
		Query:     req.Query,
		Type:      req.Type,
		Filters:   req.Filters,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Page:      int32(req.Page),
		PageSize:  int32(req.PageSize),
		Highlight: req.Highlight,
		UserId:    req.UserID,
	}
}

// ============ 响应转换 ============

// SearchResponseToProto 转换搜索响应到proto
func (c *Converter) SearchResponseToProto(resp *model.SearchResponse) *rest.SearchResponse {
	results := make([]*rest.SearchResult, len(resp.Results))
	for i, result := range resp.Results {
		if sr, ok := result.(model.SearchResult); ok {
			results[i] = c.SearchResultToProto(&sr)
		}
	}

	return &rest.SearchResponse{
		Results:  results,
		Total:    resp.Total,
		Page:     int32(resp.Page),
		PageSize: int32(resp.PageSize),
		Query:    resp.Query,
		Type:     resp.Type,
		Took:     resp.Duration,
	}
}

// SearchResultToProto 转换搜索结果到proto
func (c *Converter) SearchResultToProto(result *model.SearchResult) *rest.SearchResult {
	highlights := make(map[string]string)
	for k, v := range result.Highlight {
		if len(v) > 0 {
			highlights[k] = v[0] // 取第一个高亮片段
		}
	}

	extraData := make(map[string]string)
	for k, v := range result.Source {
		if str, ok := v.(string); ok {
			extraData[k] = str
		}
	}

	return &rest.SearchResult{
		Id:         result.ID,
		Type:       result.Type,
		Score:      float32(result.Score),
		Highlights: highlights,
		ExtraData:  extraData,
	}
}

// ============ HTTP响应构建 ============

// BuildHTTPSearchResponse 构建HTTP搜索响应
func (c *Converter) BuildHTTPSearchResponse(resp *model.SearchResponse) map[string]interface{} {
	return map[string]interface{}{
		"code":    0,
		"message": "success",
		"data":    resp,
	}
}

// BuildHTTPErrorResponse 构建HTTP错误响应
func (c *Converter) BuildHTTPErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"code":    -1,
		"message": message,
		"data":    nil,
	}
}
