package dao

import (
	"strconv"
	"strings"
	"time"

	"goim-social/apps/search-service/model"
)

// ============ 查询构建器 ============

// buildContentSearchQuery 构建内容搜索查询
func (d *elasticsearchDAO) buildContentSearchQuery(req *model.SearchRequest) map[string]interface{} {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must":   []interface{}{},
			"filter": []interface{}{},
		},
	}

	boolQuery := query["bool"].(map[string]interface{})

	// 主查询
	if req.Query != "" {
		multiMatch := map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": req.Query,
				"fields": []string{
					"title^3",
					"content^1",
					"summary^2",
					"tags^2",
				},
				"type":      "best_fields",
				"fuzziness": "AUTO",
			},
		}
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), multiMatch)
	} else {
		// 如果没有查询词，使用match_all
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"match_all": map[string]interface{}{},
		})
	}

	// 添加过滤器
	d.addFilters(boolQuery, req.Filters)

	// 添加状态过滤
	boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
		"term": map[string]interface{}{
			"status": model.ContentStatusPublished,
		},
	})

	return query
}

// buildUserSearchQuery 构建用户搜索查询
func (d *elasticsearchDAO) buildUserSearchQuery(req *model.SearchRequest) map[string]interface{} {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must":   []interface{}{},
			"filter": []interface{}{},
		},
	}

	boolQuery := query["bool"].(map[string]interface{})

	// 主查询
	if req.Query != "" {
		multiMatch := map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": req.Query,
				"fields": []string{
					"username^3",
					"nickname^2.5",
					"bio^1",
					"tags^2",
				},
				"type":      "best_fields",
				"fuzziness": "AUTO",
			},
		}
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), multiMatch)
	} else {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"match_all": map[string]interface{}{},
		})
	}

	// 添加过滤器
	d.addFilters(boolQuery, req.Filters)

	// 添加状态过滤
	boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
		"term": map[string]interface{}{
			"status": model.UserStatusActive,
		},
	})

	return query
}

// buildMessageSearchQuery 构建消息搜索查询
func (d *elasticsearchDAO) buildMessageSearchQuery(req *model.SearchRequest) map[string]interface{} {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must":   []interface{}{},
			"filter": []interface{}{},
		},
	}

	boolQuery := query["bool"].(map[string]interface{})

	// 主查询
	if req.Query != "" {
		match := map[string]interface{}{
			"match": map[string]interface{}{
				"content": map[string]interface{}{
					"query":     req.Query,
					"fuzziness": "AUTO",
				},
			},
		}
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), match)
	} else {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"match_all": map[string]interface{}{},
		})
	}

	// 添加过滤器
	d.addFilters(boolQuery, req.Filters)

	// 如果有用户ID，添加权限过滤（只能搜索自己的消息或群消息）
	if req.UserID > 0 {
		userFilter := map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"from_user_id": req.UserID,
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"to_user_id": req.UserID,
						},
					},
					map[string]interface{}{
						"exists": map[string]interface{}{
							"field": "group_id",
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
		boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), userFilter)
	}

	return query
}

// buildGroupSearchQuery 构建群组搜索查询
func (d *elasticsearchDAO) buildGroupSearchQuery(req *model.SearchRequest) map[string]interface{} {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must":   []interface{}{},
			"filter": []interface{}{},
		},
	}

	boolQuery := query["bool"].(map[string]interface{})

	// 主查询
	if req.Query != "" {
		multiMatch := map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": req.Query,
				"fields": []string{
					"name^3",
					"description^1.5",
					"tags^2",
				},
				"type":      "best_fields",
				"fuzziness": "AUTO",
			},
		}
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), multiMatch)
	} else {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"match_all": map[string]interface{}{},
		})
	}

	// 添加过滤器
	d.addFilters(boolQuery, req.Filters)

	// 添加状态过滤
	boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
		"term": map[string]interface{}{
			"status": model.GroupStatusActive,
		},
	})

	return query
}

// buildMultiSearchQuery 构建多类型搜索查询
func (d *elasticsearchDAO) buildMultiSearchQuery(req *model.SearchRequest, index string) map[string]interface{} {
	switch index {
	case model.IndexContent:
		return d.buildContentSearchQuery(req)
	case model.IndexUser:
		return d.buildUserSearchQuery(req)
	case model.IndexMessage:
		return d.buildMessageSearchQuery(req)
	case model.IndexGroup:
		return d.buildGroupSearchQuery(req)
	default:
		return map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}
}

// buildSortQuery 构建排序查询
func (d *elasticsearchDAO) buildSortQuery(sortBy, sortOrder string) []map[string]interface{} {
	if sortOrder == "" {
		sortOrder = model.SortOrderDesc
	}

	sort := make([]map[string]interface{}, 0)

	switch sortBy {
	case model.SortByRelevance:
		sort = append(sort, map[string]interface{}{
			"_score": map[string]interface{}{
				"order": sortOrder,
			},
		})
	case model.SortByTime, model.SortByCreated:
		sort = append(sort, map[string]interface{}{
			"created_at": map[string]interface{}{
				"order": sortOrder,
			},
		})
	case model.SortByUpdated:
		sort = append(sort, map[string]interface{}{
			"updated_at": map[string]interface{}{
				"order": sortOrder,
			},
		})
	case model.SortByViews:
		sort = append(sort, map[string]interface{}{
			"view_count": map[string]interface{}{
				"order": sortOrder,
			},
		})
	case model.SortByLikes:
		sort = append(sort, map[string]interface{}{
			"like_count": map[string]interface{}{
				"order": sortOrder,
			},
		})
	case model.SortByComments:
		sort = append(sort, map[string]interface{}{
			"comment_count": map[string]interface{}{
				"order": sortOrder,
			},
		})
	case model.SortByMembers:
		sort = append(sort, map[string]interface{}{
			"member_count": map[string]interface{}{
				"order": sortOrder,
			},
		})
	default:
		// 默认按相关性排序
		sort = append(sort, map[string]interface{}{
			"_score": map[string]interface{}{
				"order": model.SortOrderDesc,
			},
		})
	}

	return sort
}

// buildHighlightQuery 构建高亮查询
func (d *elasticsearchDAO) buildHighlightQuery(fields []string) map[string]interface{} {
	highlight := map[string]interface{}{
		"pre_tags":  []string{model.DefaultHighlightPreTag},
		"post_tags": []string{model.DefaultHighlightPostTag},
		"fields":    map[string]interface{}{},
	}

	highlightFields := highlight["fields"].(map[string]interface{})
	for _, field := range fields {
		highlightFields[field] = map[string]interface{}{
			"fragment_size":       150,
			"number_of_fragments": 3,
		}
	}

	return highlight
}

// addFilters 添加过滤器
func (d *elasticsearchDAO) addFilters(boolQuery map[string]interface{}, filters map[string]string) {
	if filters == nil {
		return
	}

	for key, value := range filters {
		if value == "" {
			continue
		}

		switch key {
		case "author_id", "user_id", "owner_id", "from_user_id", "to_user_id", "group_id":
			if id, err := strconv.ParseInt(value, 10, 64); err == nil {
				boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
					"term": map[string]interface{}{
						key: id,
					},
				})
			}
		case "category", "status", "message_type":
			boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
				"term": map[string]interface{}{
					key: value,
				},
			})
		case "tags":
			tags := strings.Split(value, ",")
			boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
				"terms": map[string]interface{}{
					key: tags,
				},
			})
		case "date_from":
			if t, err := time.Parse("2006-01-02", value); err == nil {
				boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
					"range": map[string]interface{}{
						"created_at": map[string]interface{}{
							"gte": t.Format(time.RFC3339),
						},
					},
				})
			}
		case "date_to":
			if t, err := time.Parse("2006-01-02", value); err == nil {
				boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
					"range": map[string]interface{}{
						"created_at": map[string]interface{}{
							"lte": t.Format(time.RFC3339),
						},
					},
				})
			}
		case "is_public":
			if isPublic, err := strconv.ParseBool(value); err == nil {
				boolQuery["filter"] = append(boolQuery["filter"].([]interface{}), map[string]interface{}{
					"term": map[string]interface{}{
						"is_public": isPublic,
					},
				})
			}
		}
	}
}

// getHighlightFields 获取高亮字段
func (d *elasticsearchDAO) getHighlightFields(index string) []string {
	switch index {
	case model.IndexContent:
		return []string{"title", "content", "summary"}
	case model.IndexUser:
		return []string{"username", "nickname", "bio"}
	case model.IndexMessage:
		return []string{"content"}
	case model.IndexGroup:
		return []string{"name", "description"}
	default:
		return []string{}
	}
}
