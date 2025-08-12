package handler

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"goim-social/api/rest"
	"goim-social/apps/search-service/internal/model"
	"goim-social/apps/search-service/internal/service"
	"goim-social/pkg/logger"
)

// GRPCHandler gRPC处理器
type GRPCHandler struct {
	rest.UnimplementedSearchServiceServer
	rest.UnimplementedIndexServiceServer
	searchService service.SearchService
	indexService  service.IndexService
	logger        logger.Logger
}

// NewGRPCHandler 创建gRPC处理器
func NewGRPCHandler(searchService service.SearchService, indexService service.IndexService, logger logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		searchService: searchService,
		indexService:  indexService,
		logger:        logger,
	}
}

// ============ 搜索服务实现 ============

// Search 通用搜索
func (h *GRPCHandler) Search(ctx context.Context, req *rest.SearchRequest) (*rest.SearchResponse, error) {
	// 转换请求
	searchReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      req.Type,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
		Filters:   req.Filters,
	}

	// 执行搜索
	searchResp, err := h.searchService.Search(ctx, searchReq)
	if err != nil {
		h.logger.Error(ctx, "Search failed",
			logger.F("query", req.Query),
			logger.F("type", req.Type),
			logger.F("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	// 转换响应
	response := &rest.SearchResponse{
		Results:  make([]*rest.SearchResult, len(searchResp.Results)),
		Total:    searchResp.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Query:    req.Query,
		Type:     req.Type,
		Took:     0, // TODO: 添加执行时间统计
	}

	for i, result := range searchResp.Results {
		response.Results[i] = h.convertToSearchResult(result)
	}

	return response, nil
}

// SearchContent 内容搜索
func (h *GRPCHandler) SearchContent(ctx context.Context, req *rest.SearchContentRequest) (*rest.SearchContentResponse, error) {
	// 转换请求
	searchReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeContent,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
		Filters:   make(map[string]string),
	}

	// 添加过滤器
	if req.Category != "" {
		searchReq.Filters["category"] = req.Category
	}
	if req.AuthorId > 0 {
		searchReq.Filters["author_id"] = strconv.FormatInt(req.AuthorId, 10)
	}
	if req.Status != "" {
		searchReq.Filters["status"] = req.Status
	}
	if req.IsPublic {
		searchReq.Filters["is_public"] = "true"
	}
	if req.DateFrom != "" {
		searchReq.Filters["date_from"] = req.DateFrom
	}
	if req.DateTo != "" {
		searchReq.Filters["date_to"] = req.DateTo
	}

	// 执行搜索
	results, total, err := h.searchService.SearchContent(ctx, searchReq)
	if err != nil {
		h.logger.Error(ctx, "SearchContent failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "search content failed: %v", err)
	}

	// 转换响应
	response := &rest.SearchContentResponse{
		Results:  make([]*rest.ContentSearchResult, len(results)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Took:     0, // TODO: 添加执行时间统计
	}

	for i, result := range results {
		response.Results[i] = h.convertToContentSearchResult(result)
	}

	return response, nil
}

// SearchUsers 用户搜索
func (h *GRPCHandler) SearchUsers(ctx context.Context, req *rest.SearchUsersRequest) (*rest.SearchUsersResponse, error) {
	// 转换请求
	searchReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeUser,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
		Filters:   make(map[string]string),
	}

	// 添加过滤器
	if req.IsVerified {
		searchReq.Filters["is_verified"] = "true"
	}
	if req.Status != "" {
		searchReq.Filters["status"] = req.Status
	}
	if req.Role != "" {
		searchReq.Filters["role"] = req.Role
	}
	if req.DateFrom != "" {
		searchReq.Filters["date_from"] = req.DateFrom
	}
	if req.DateTo != "" {
		searchReq.Filters["date_to"] = req.DateTo
	}

	// 执行搜索
	results, total, err := h.searchService.SearchUsers(ctx, searchReq)
	if err != nil {
		h.logger.Error(ctx, "SearchUsers failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "search users failed: %v", err)
	}

	// 转换响应
	response := &rest.SearchUsersResponse{
		Results:  make([]*rest.UserSearchResult, len(results)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Took:     0, // TODO: 添加执行时间统计
	}

	for i, result := range results {
		response.Results[i] = h.convertToUserSearchResult(result)
	}

	return response, nil
}

// SearchMessages 消息搜索
func (h *GRPCHandler) SearchMessages(ctx context.Context, req *rest.SearchMessagesRequest) (*rest.SearchMessagesResponse, error) {
	// 转换请求
	searchReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeMessage,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
		Filters:   make(map[string]string),
	}

	// 添加过滤器
	if req.GroupId > 0 {
		searchReq.Filters["group_id"] = strconv.FormatInt(req.GroupId, 10)
	}
	if req.SenderId > 0 {
		searchReq.Filters["sender_id"] = strconv.FormatInt(req.SenderId, 10)
	}
	if req.MessageType != "" {
		searchReq.Filters["message_type"] = req.MessageType
	}
	if req.DateFrom != "" {
		searchReq.Filters["date_from"] = req.DateFrom
	}
	if req.DateTo != "" {
		searchReq.Filters["date_to"] = req.DateTo
	}

	// 执行搜索
	results, total, err := h.searchService.SearchMessages(ctx, searchReq)
	if err != nil {
		h.logger.Error(ctx, "SearchMessages failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "search messages failed: %v", err)
	}

	// 转换响应
	response := &rest.SearchMessagesResponse{
		Results:  make([]*rest.MessageSearchResult, len(results)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Took:     0, // TODO: 添加执行时间统计
	}

	for i, result := range results {
		response.Results[i] = h.convertToMessageSearchResult(result)
	}

	return response, nil
}

// SearchGroups 群组搜索
func (h *GRPCHandler) SearchGroups(ctx context.Context, req *rest.SearchGroupsRequest) (*rest.SearchGroupsResponse, error) {
	// 转换请求
	searchReq := &model.SearchRequest{
		Query:     req.Query,
		Type:      model.SearchTypeGroup,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Highlight: req.Highlight,
		UserID:    req.UserId,
		Filters:   make(map[string]string),
	}

	// 添加过滤器
	if req.IsPublic {
		searchReq.Filters["is_public"] = "true"
	}
	if req.Status != "" {
		searchReq.Filters["status"] = req.Status
	}
	if req.Category != "" {
		searchReq.Filters["category"] = req.Category
	}
	if req.DateFrom != "" {
		searchReq.Filters["date_from"] = req.DateFrom
	}
	if req.DateTo != "" {
		searchReq.Filters["date_to"] = req.DateTo
	}

	// 执行搜索
	results, total, err := h.searchService.SearchGroups(ctx, searchReq)
	if err != nil {
		h.logger.Error(ctx, "SearchGroups failed",
			logger.F("query", req.Query),
			logger.F("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "search groups failed: %v", err)
	}

	// 转换响应
	response := &rest.SearchGroupsResponse{
		Results:  make([]*rest.GroupSearchResult, len(results)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Took:     0, // TODO: 添加执行时间统计
	}

	for i, result := range results {
		response.Results[i] = h.convertToGroupSearchResult(result)
	}

	return response, nil
}

// ============ 转换函数 ============

// convertToSearchResult 转换为通用搜索结果
func (h *GRPCHandler) convertToSearchResult(result interface{}) *rest.SearchResult {
	// 这里需要根据实际的搜索结果类型进行转换
	// 暂时返回一个空的结果
	return &rest.SearchResult{
		Id:           "1",
		Type:         "content",
		Title:        "Sample Title",
		Content:      "Sample Content",
		Summary:      "Sample Summary",
		Tags:         []string{"tag1", "tag2"},
		AuthorId:     1,
		AuthorName:   "Sample Author",
		AuthorAvatar: "https://example.com/avatar.jpg",
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		Score:        float32(1.0),
		Highlights:   make(map[string]string),
		ExtraData:    make(map[string]string),
	}
}

// convertToContentSearchResult 转换为内容搜索结果
func (h *GRPCHandler) convertToContentSearchResult(result *model.ContentSearchResult) *rest.ContentSearchResult {
	highlights := make(map[string]string)
	for key, values := range result.Highlight {
		if len(values) > 0 {
			highlights[key] = values[0] // 取第一个高亮结果
		}
	}

	return &rest.ContentSearchResult{
		Id:           result.ID,
		Title:        result.Title,
		Content:      result.Content,
		Summary:      result.Summary,
		Category:     result.Category,
		Tags:         result.Tags,
		AuthorId:     result.AuthorID,
		AuthorName:   result.AuthorName,
		AuthorAvatar: "",          // 模型中没有这个字段
		Status:       "published", // 模型中没有这个字段，使用默认值
		IsPublic:     true,        // 模型中没有这个字段，使用默认值
		ViewCount:    result.ViewCount,
		LikeCount:    result.LikeCount,
		CommentCount: result.CommentCount,
		CreatedAt:    result.CreatedAt.Unix(),
		UpdatedAt:    result.CreatedAt.Unix(), // 模型中没有UpdatedAt，使用CreatedAt
		Score:        float32(result.Score),
		Highlights:   highlights,
	}
}

// convertToUserSearchResult 转换为用户搜索结果
func (h *GRPCHandler) convertToUserSearchResult(result *model.UserSearchResult) *rest.UserSearchResult {
	highlights := make(map[string]string)
	for key, values := range result.Highlight {
		if len(values) > 0 {
			highlights[key] = values[0] // 取第一个高亮结果
		}
	}

	return &rest.UserSearchResult{
		Id:             result.ID,
		Username:       result.Username,
		Nickname:       result.Nickname,
		Email:          "", // 模型中没有这个字段
		Phone:          "", // 模型中没有这个字段
		Avatar:         result.Avatar,
		Bio:            result.Bio,
		Status:         "active", // 模型中没有这个字段，使用默认值
		Role:           "user",   // 模型中没有这个字段，使用默认值
		IsVerified:     result.IsVerified,
		FollowerCount:  result.FollowerCount,
		FollowingCount: result.FriendCount, // 使用FriendCount作为FollowingCount
		ContentCount:   0,                  // 模型中没有这个字段
		CreatedAt:      time.Now().Unix(),  // 模型中没有CreatedAt字段
		UpdatedAt:      time.Now().Unix(),  // 模型中没有UpdatedAt字段
		LastLoginAt:    time.Now().Unix(),  // 模型中没有LastLoginAt字段
		Score:          float32(result.Score),
		Highlights:     highlights,
	}
}

// convertToMessageSearchResult 转换为消息搜索结果
func (h *GRPCHandler) convertToMessageSearchResult(result *model.MessageSearchResult) *rest.MessageSearchResult {
	highlights := make(map[string]string)
	for key, values := range result.Highlight {
		if len(values) > 0 {
			highlights[key] = values[0] // 取第一个高亮结果
		}
	}

	return &rest.MessageSearchResult{
		Id:           result.ID,
		Content:      result.Content,
		MessageType:  result.MessageType,
		SenderId:     result.FromUserID,
		SenderName:   result.FromUsername,
		SenderAvatar: "", // TODO模型中没有这个字段
		GroupId:      result.GroupID,
		GroupName:    result.GroupName,
		CreatedAt:    result.CreatedAt.Unix(),
		UpdatedAt:    result.CreatedAt.Unix(), // 模型中没有UpdatedAt，使用CreatedAt
		Score:        float32(result.Score),
		Highlights:   highlights,
		ExtraData:    make(map[string]string),
	}
}

// convertToGroupSearchResult 转换为群组搜索结果
func (h *GRPCHandler) convertToGroupSearchResult(result *model.GroupSearchResult) *rest.GroupSearchResult {
	highlights := make(map[string]string)
	for key, values := range result.Highlight {
		if len(values) > 0 {
			highlights[key] = values[0] // 取第一个高亮结果
		}
	}

	return &rest.GroupSearchResult{
		Id:           result.ID,
		Name:         result.Name,
		Description:  result.Description,
		Category:     result.Category,
		Avatar:       result.Avatar,
		Status:       "active", // TODO模型中没有这个字段，使用默认值
		IsPublic:     result.IsPublic,
		OwnerId:      result.OwnerID,
		OwnerName:    result.OwnerName,
		MemberCount:  result.MemberCount,
		MessageCount: 0,                 // 模型中没有这个字段
		CreatedAt:    time.Now().Unix(), // 模型中没有CreatedAt字段
		UpdatedAt:    time.Now().Unix(), // 模型中没有UpdatedAt字段
		Score:        float32(result.Score),
		Highlights:   highlights,
	}
}
