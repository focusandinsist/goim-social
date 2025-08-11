package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/search-service/model"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// GetSuggestions 获取搜索建议
func (h *HTTPHandler) GetSuggestions(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.GetSuggestionsRequest{}
	req.Query = c.Query("query")
	req.Type = c.Query("type")

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			req.Limit = int32(l)
		}
	}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	result, err := h.searchService.GetSuggestions(ctx, req.Query, req.Type, int(req.Limit), req.UserId)
	if err != nil {
		h.logger.Error(ctx, "GetSuggestions failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("get suggestions failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}

// GetAutoComplete 获取自动完成
func (h *HTTPHandler) GetAutoComplete(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.GetAutoCompleteRequest{}
	req.Query = c.Query("query")
	req.Type = c.Query("type")

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			req.Limit = int32(l)
		}
	}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	// 转换为model请求
	modelReq := &model.AutoCompleteRequest{
		Query:  req.Query,
		Type:   req.Type,
		Limit:  int(req.Limit),
		UserID: req.UserId,
	}

	result, err := h.searchService.GetAutoComplete(ctx, modelReq)
	if err != nil {
		h.logger.Error(ctx, "GetAutoComplete failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("get autocomplete failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}

// GetHotSearches 获取热门搜索
func (h *HTTPHandler) GetHotSearches(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.GetHotSearchesRequest{}
	req.Type = c.Query("type")

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			req.Limit = int32(l)
		}
	}

	result, err := h.searchService.GetHotSearches(ctx, req.Type, int(req.Limit))
	if err != nil {
		h.logger.Error(ctx, "GetHotSearches failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("get hot searches failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}

// GetUserSearchHistory 获取用户搜索历史
func (h *HTTPHandler) GetUserSearchHistory(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.GetSearchHistoryRequest{}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	req.Type = c.Query("type")

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			req.Limit = int32(l)
		}
	}

	result, err := h.searchService.GetUserSearchHistory(ctx, req.UserId, req.Type, int(req.Limit))
	if err != nil {
		h.logger.Error(ctx, "GetUserSearchHistory failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("get search history failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}

// ClearUserSearchHistory 清空用户搜索历史
func (h *HTTPHandler) ClearUserSearchHistory(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.ClearSearchHistoryRequest{}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	req.Type = c.Query("type")

	err = h.searchService.ClearUserSearchHistory(ctx, req.UserId, req.Type)
	if err != nil {
		h.logger.Error(ctx, "ClearUserSearchHistory failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("clear search history failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// DeleteSearchHistory 删除搜索历史项
func (h *HTTPHandler) DeleteSearchHistory(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.DeleteSearchHistoryItemRequest{}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	if itemID := c.Param("id"); itemID != "" {
		if iid, err := strconv.ParseInt(itemID, 10, 64); err == nil {
			req.ItemId = iid
		}
	}

	err = h.searchService.DeleteSearchHistory(ctx, req.UserId, req.ItemId)
	if err != nil {
		h.logger.Error(ctx, "DeleteSearchHistory failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("delete search history failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// GetUserPreference 获取用户搜索偏好
func (h *HTTPHandler) GetUserPreference(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.GetUserSearchPreferenceRequest{}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	result, err := h.searchService.GetUserPreference(ctx, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "GetUserPreference failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("get user preference failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}

// UpdateUserPreference 更新用户搜索偏好
func (h *HTTPHandler) UpdateUserPreference(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.UpdateUserSearchPreferenceRequest{}
	if err = c.Bind(req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 这里需要将proto请求转换为model.UserSearchPreference
	// 简化处理，实际应该有完整的转换逻辑
	preference := &model.UserSearchPreference{
		UserID: req.UserId,
		// 其他字段需要从req中提取
	}

	err = h.searchService.UpdateUserPreference(ctx, preference)
	if err != nil {
		h.logger.Error(ctx, "UpdateUserPreference failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("update user preference failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// GetSearchStats 获取搜索统计
func (h *HTTPHandler) GetSearchStats(c *gin.Context) {
	var (
		resp interface{}
		err  error
	)

	// 这里简化处理，实际应该有更复杂的统计请求结构
	resp = map[string]interface{}{
		"code":    0,
		"message": "success",
		"data":    map[string]interface{}{"stats": "placeholder"},
	}

	httpx.WriteObject(c, resp, err)
}

// GetSearchAnalytics 获取搜索分析
func (h *HTTPHandler) GetSearchAnalytics(c *gin.Context) {
	var (
		resp interface{}
		err  error
	)

	// 这里简化处理，实际应该有更复杂的分析请求结构
	resp = map[string]interface{}{
		"code":    0,
		"message": "success",
		"data":    map[string]interface{}{"analytics": "placeholder"},
	}

	httpx.WriteObject(c, resp, err)
}
