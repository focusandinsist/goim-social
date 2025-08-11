package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/search-service/model"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// Search 通用搜索
func (h *HTTPHandler) Search(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.SearchRequest{}
	if err = h.bindSearchRequest(c, req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 转换为model请求
	modelReq := h.converter.SearchRequestFromProto(req)

	result, err := h.searchService.Search(ctx, modelReq)
	if err != nil {
		h.logger.Error(ctx, "Search failed",
			logger.F("query", req.Query),
			logger.F("type", req.Type),
			logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("search failed: " + err.Error())
	} else {
		resp = h.converter.BuildHTTPSearchResponse(result)
	}

	httpx.WriteObject(c, resp, err)
}

// SearchContent 内容搜索
func (h *HTTPHandler) SearchContent(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.SearchContentRequest{}
	if err = h.bindSearchContentRequest(c, req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 转换为model请求
	modelReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeContent,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
	}

	results, total, err := h.searchService.SearchContent(ctx, modelReq)
	if err != nil {
		h.logger.Error(ctx, "SearchContent failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("search content failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data": map[string]interface{}{
				"results":   results,
				"total":     total,
				"page":      req.Page,
				"page_size": req.PageSize,
			},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// SearchUsers 用户搜索
func (h *HTTPHandler) SearchUsers(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.SearchUsersRequest{}
	if err = h.bindSearchUsersRequest(c, req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 转换为model请求
	modelReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeUser,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
	}

	results, total, err := h.searchService.SearchUsers(ctx, modelReq)
	if err != nil {
		h.logger.Error(ctx, "SearchUsers failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("search users failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data": map[string]interface{}{
				"results":   results,
				"total":     total,
				"page":      req.Page,
				"page_size": req.PageSize,
			},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// SearchMessages 消息搜索
func (h *HTTPHandler) SearchMessages(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.SearchMessagesRequest{}
	if err = h.bindSearchMessagesRequest(c, req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 转换为model请求
	modelReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeMessage,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
	}

	results, total, err := h.searchService.SearchMessages(ctx, modelReq)
	if err != nil {
		h.logger.Error(ctx, "SearchMessages failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("search messages failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data": map[string]interface{}{
				"results":   results,
				"total":     total,
				"page":      req.Page,
				"page_size": req.PageSize,
			},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// SearchGroups 群组搜索
func (h *HTTPHandler) SearchGroups(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.SearchGroupsRequest{}
	if err = h.bindSearchGroupsRequest(c, req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 转换为model请求
	modelReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeGroup,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
	}

	results, total, err := h.searchService.SearchGroups(ctx, modelReq)
	if err != nil {
		h.logger.Error(ctx, "SearchGroups failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("search groups failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data": map[string]interface{}{
				"results":   results,
				"total":     total,
				"page":      req.Page,
				"page_size": req.PageSize,
			},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// MultiSearch 多类型搜索
func (h *HTTPHandler) MultiSearch(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.MultiSearchRequest{}
	if err = h.bindMultiSearchRequest(c, req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 转换为model请求
	modelReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeAll, // 多类型搜索
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
	}

	result, err := h.searchService.MultiSearch(ctx, modelReq)
	if err != nil {
		h.logger.Error(ctx, "MultiSearch failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("multi search failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}

// ============ 请求绑定辅助方法 ============

// bindSearchRequest 绑定搜索请求
func (h *HTTPHandler) bindSearchRequest(c *gin.Context, req *rest.SearchRequest) error {
	// 绑定查询参数
	req.Query = c.Query("query")
	req.Type = c.Query("type")
	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = int32(p)
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = int32(ps)
		}
	}

	if highlight := c.Query("highlight"); highlight == "true" {
		req.Highlight = true
	}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = int32(model.DefaultPageSize)
	}
	if req.PageSize > int32(model.MaxPageSize) {
		req.PageSize = int32(model.MaxPageSize)
	}

	return nil
}

// bindSearchContentRequest 绑定内容搜索请求
func (h *HTTPHandler) bindSearchContentRequest(c *gin.Context, req *rest.SearchContentRequest) error {
	req.Query = c.Query("query")

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = int32(p)
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = int32(ps)
		}
	}

	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")
	req.Category = c.Query("category")
	req.Status = c.Query("status")

	if highlight := c.Query("highlight"); highlight == "true" {
		req.Highlight = true
	}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	if authorID := c.Query("author_id"); authorID != "" {
		if aid, err := strconv.ParseInt(authorID, 10, 64); err == nil {
			req.AuthorId = aid
		}
	}

	if isPublic := c.Query("is_public"); isPublic == "true" {
		req.IsPublic = true
	}

	req.DateFrom = c.Query("date_from")
	req.DateTo = c.Query("date_to")

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = int32(model.DefaultPageSize)
	}

	return nil
}

// bindSearchUsersRequest 绑定用户搜索请求
func (h *HTTPHandler) bindSearchUsersRequest(c *gin.Context, req *rest.SearchUsersRequest) error {
	req.Query = c.Query("query")

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = int32(p)
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = int32(ps)
		}
	}

	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")
	req.Status = c.Query("status")
	req.Role = c.Query("role")

	if highlight := c.Query("highlight"); highlight == "true" {
		req.Highlight = true
	}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	if isVerified := c.Query("is_verified"); isVerified == "true" {
		req.IsVerified = true
	}

	req.DateFrom = c.Query("date_from")
	req.DateTo = c.Query("date_to")

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = int32(model.DefaultPageSize)
	}

	return nil
}

// bindSearchMessagesRequest 绑定消息搜索请求
func (h *HTTPHandler) bindSearchMessagesRequest(c *gin.Context, req *rest.SearchMessagesRequest) error {
	req.Query = c.Query("query")

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = int32(p)
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = int32(ps)
		}
	}

	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")
	req.MessageType = c.Query("message_type")

	if highlight := c.Query("highlight"); highlight == "true" {
		req.Highlight = true
	}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	if groupID := c.Query("group_id"); groupID != "" {
		if gid, err := strconv.ParseInt(groupID, 10, 64); err == nil {
			req.GroupId = gid
		}
	}

	if senderID := c.Query("sender_id"); senderID != "" {
		if sid, err := strconv.ParseInt(senderID, 10, 64); err == nil {
			req.SenderId = sid
		}
	}

	req.DateFrom = c.Query("date_from")
	req.DateTo = c.Query("date_to")

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = int32(model.DefaultPageSize)
	}

	return nil
}

// bindSearchGroupsRequest 绑定群组搜索请求
func (h *HTTPHandler) bindSearchGroupsRequest(c *gin.Context, req *rest.SearchGroupsRequest) error {
	req.Query = c.Query("query")

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = int32(p)
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = int32(ps)
		}
	}

	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")
	req.Status = c.Query("status")
	req.Category = c.Query("category")

	if highlight := c.Query("highlight"); highlight == "true" {
		req.Highlight = true
	}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	if isPublic := c.Query("is_public"); isPublic == "true" {
		req.IsPublic = true
	}

	req.DateFrom = c.Query("date_from")
	req.DateTo = c.Query("date_to")

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = int32(model.DefaultPageSize)
	}

	return nil
}

// bindMultiSearchRequest 绑定多类型搜索请求
func (h *HTTPHandler) bindMultiSearchRequest(c *gin.Context, req *rest.MultiSearchRequest) error {
	req.Query = c.Query("query")

	// 处理types参数（逗号分隔）
	if typesStr := c.Query("types"); typesStr != "" {
		// 这里简化处理，实际应该解析逗号分隔的字符串
		req.Types = []string{typesStr}
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = int32(p)
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = int32(ps)
		}
	}

	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")

	if highlight := c.Query("highlight"); highlight == "true" {
		req.Highlight = true
	}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
			req.UserId = uid
		}
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = int32(model.DefaultPageSize)
	}

	return nil
}
