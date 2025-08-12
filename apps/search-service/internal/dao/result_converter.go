package dao

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"goim-social/apps/search-service/internal/model"
)

// ============ 结果转换器 ============

// convertToContentResult 转换为内容搜索结果
func (d *elasticsearchDAO) convertToContentResult(hit SearchHit) (*model.ContentSearchResult, error) {
	result := &model.ContentSearchResult{
		Score:     hit.Score,
		Highlight: hit.Highlight,
	}

	// 转换基础字段
	if id, ok := hit.Source["id"]; ok {
		if idFloat, ok := id.(float64); ok {
			result.ID = int64(idFloat)
		} else if idStr, ok := id.(string); ok {
			if parsedID, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				result.ID = parsedID
			}
		}
	}

	if title, ok := hit.Source["title"].(string); ok {
		result.Title = title
	}

	if content, ok := hit.Source["content"].(string); ok {
		result.Content = content
	}

	if summary, ok := hit.Source["summary"].(string); ok {
		result.Summary = summary
	}

	if authorID, ok := hit.Source["author_id"]; ok {
		if authorIDFloat, ok := authorID.(float64); ok {
			result.AuthorID = int64(authorIDFloat)
		}
	}

	if authorName, ok := hit.Source["author_name"].(string); ok {
		result.AuthorName = authorName
	}

	if category, ok := hit.Source["category"].(string); ok {
		result.Category = category
	}

	// 转换标签
	if tags, ok := hit.Source["tags"].([]interface{}); ok {
		result.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				result.Tags = append(result.Tags, tagStr)
			}
		}
	}

	// 转换计数字段
	if viewCount, ok := hit.Source["view_count"]; ok {
		if viewCountFloat, ok := viewCount.(float64); ok {
			result.ViewCount = int64(viewCountFloat)
		}
	}

	if likeCount, ok := hit.Source["like_count"]; ok {
		if likeCountFloat, ok := likeCount.(float64); ok {
			result.LikeCount = int64(likeCountFloat)
		}
	}

	if commentCount, ok := hit.Source["comment_count"]; ok {
		if commentCountFloat, ok := commentCount.(float64); ok {
			result.CommentCount = int64(commentCountFloat)
		}
	}

	// 转换时间字段
	if createdAt, ok := hit.Source["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			result.CreatedAt = t
		}
	}

	return result, nil
}

// convertToUserResult 转换为用户搜索结果
func (d *elasticsearchDAO) convertToUserResult(hit SearchHit) (*model.UserSearchResult, error) {
	result := &model.UserSearchResult{
		Score:     hit.Score,
		Highlight: hit.Highlight,
	}

	// 转换基础字段
	if id, ok := hit.Source["id"]; ok {
		if idFloat, ok := id.(float64); ok {
			result.ID = int64(idFloat)
		}
	}

	if username, ok := hit.Source["username"].(string); ok {
		result.Username = username
	}

	if nickname, ok := hit.Source["nickname"].(string); ok {
		result.Nickname = nickname
	}

	if avatar, ok := hit.Source["avatar"].(string); ok {
		result.Avatar = avatar
	}

	if bio, ok := hit.Source["bio"].(string); ok {
		result.Bio = bio
	}

	if location, ok := hit.Source["location"].(string); ok {
		result.Location = location
	}

	// 转换标签
	if tags, ok := hit.Source["tags"].([]interface{}); ok {
		result.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				result.Tags = append(result.Tags, tagStr)
			}
		}
	}

	// 转换布尔字段
	if isVerified, ok := hit.Source["is_verified"].(bool); ok {
		result.IsVerified = isVerified
	}

	// 转换计数字段
	if friendCount, ok := hit.Source["friend_count"]; ok {
		if friendCountFloat, ok := friendCount.(float64); ok {
			result.FriendCount = int64(friendCountFloat)
		}
	}

	if followerCount, ok := hit.Source["follower_count"]; ok {
		if followerCountFloat, ok := followerCount.(float64); ok {
			result.FollowerCount = int64(followerCountFloat)
		}
	}

	return result, nil
}

// convertToMessageResult 转换为消息搜索结果
func (d *elasticsearchDAO) convertToMessageResult(hit SearchHit) (*model.MessageSearchResult, error) {
	result := &model.MessageSearchResult{
		Score:     hit.Score,
		Highlight: hit.Highlight,
	}

	// 转换基础字段
	if id, ok := hit.Source["id"]; ok {
		if idFloat, ok := id.(float64); ok {
			result.ID = int64(idFloat)
		}
	}

	if fromUserID, ok := hit.Source["from_user_id"]; ok {
		if fromUserIDFloat, ok := fromUserID.(float64); ok {
			result.FromUserID = int64(fromUserIDFloat)
		}
	}

	if fromUsername, ok := hit.Source["from_username"].(string); ok {
		result.FromUsername = fromUsername
	}

	if toUserID, ok := hit.Source["to_user_id"]; ok {
		if toUserIDFloat, ok := toUserID.(float64); ok {
			result.ToUserID = int64(toUserIDFloat)
		}
	}

	if toUsername, ok := hit.Source["to_username"].(string); ok {
		result.ToUsername = toUsername
	}

	if groupID, ok := hit.Source["group_id"]; ok {
		if groupIDFloat, ok := groupID.(float64); ok {
			result.GroupID = int64(groupIDFloat)
		}
	}

	if groupName, ok := hit.Source["group_name"].(string); ok {
		result.GroupName = groupName
	}

	if content, ok := hit.Source["content"].(string); ok {
		result.Content = content
	}

	if messageType, ok := hit.Source["message_type"].(string); ok {
		result.MessageType = messageType
	}

	// 转换时间字段
	if createdAt, ok := hit.Source["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			result.CreatedAt = t
		}
	}

	return result, nil
}

// convertToGroupResult 转换为群组搜索结果
func (d *elasticsearchDAO) convertToGroupResult(hit SearchHit) (*model.GroupSearchResult, error) {
	result := &model.GroupSearchResult{
		Score:     hit.Score,
		Highlight: hit.Highlight,
	}

	// 转换基础字段
	if id, ok := hit.Source["id"]; ok {
		if idFloat, ok := id.(float64); ok {
			result.ID = int64(idFloat)
		}
	}

	if name, ok := hit.Source["name"].(string); ok {
		result.Name = name
	}

	if description, ok := hit.Source["description"].(string); ok {
		result.Description = description
	}

	if avatar, ok := hit.Source["avatar"].(string); ok {
		result.Avatar = avatar
	}

	if ownerID, ok := hit.Source["owner_id"]; ok {
		if ownerIDFloat, ok := ownerID.(float64); ok {
			result.OwnerID = int64(ownerIDFloat)
		}
	}

	if ownerName, ok := hit.Source["owner_name"].(string); ok {
		result.OwnerName = ownerName
	}

	if memberCount, ok := hit.Source["member_count"]; ok {
		if memberCountFloat, ok := memberCount.(float64); ok {
			result.MemberCount = int64(memberCountFloat)
		}
	}

	if isPublic, ok := hit.Source["is_public"].(bool); ok {
		result.IsPublic = isPublic
	}

	// 转换标签
	if tags, ok := hit.Source["tags"].([]interface{}); ok {
		result.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				result.Tags = append(result.Tags, tagStr)
			}
		}
	}

	if category, ok := hit.Source["category"].(string); ok {
		result.Category = category
	}

	return result, nil
}

// convertMultiSearchResult 转换多搜索结果
func (d *elasticsearchDAO) convertMultiSearchResult(hit map[string]interface{}, index string) interface{} {
	// 构建SearchHit结构
	searchHit := SearchHit{
		Score: 0,
	}

	if score, ok := hit["_score"].(float64); ok {
		searchHit.Score = score
	}

	if id, ok := hit["_id"].(string); ok {
		searchHit.ID = id
	}

	if source, ok := hit["_source"].(map[string]interface{}); ok {
		searchHit.Source = source
	}

	if highlight, ok := hit["highlight"].(map[string]interface{}); ok {
		searchHit.Highlight = make(map[string][]string)
		for field, values := range highlight {
			if valueList, ok := values.([]interface{}); ok {
				stringList := make([]string, 0, len(valueList))
				for _, v := range valueList {
					if str, ok := v.(string); ok {
						stringList = append(stringList, str)
					}
				}
				searchHit.Highlight[field] = stringList
			}
		}
	}

	// 根据索引类型转换结果
	switch index {
	case model.IndexContent:
		if result, err := d.convertToContentResult(searchHit); err == nil {
			return result
		}
	case model.IndexUser:
		if result, err := d.convertToUserResult(searchHit); err == nil {
			return result
		}
	case model.IndexMessage:
		if result, err := d.convertToMessageResult(searchHit); err == nil {
			return result
		}
	case model.IndexGroup:
		if result, err := d.convertToGroupResult(searchHit); err == nil {
			return result
		}
	}

	return nil
}

// convertToSearchResult 转换为通用搜索结果
func (d *elasticsearchDAO) convertToSearchResult(hit SearchHit) *model.SearchResult {
	return &model.SearchResult{
		ID:        hit.ID,
		Type:      hit.Index,
		Score:     hit.Score,
		Source:    hit.Source,
		Highlight: hit.Highlight,
	}
}

// parseTimeField 解析时间字段
func parseTimeField(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		// 尝试多种时间格式
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unable to parse time: %s", v)
	case float64:
		// Unix时间戳（秒）
		return time.Unix(int64(v), 0), nil
	case int64:
		// Unix时间戳（秒）
		return time.Unix(v, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported time type: %T", value)
	}
}

// parseIntField 解析整数字段
func parseIntField(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported int type: %T", value)
	}
}

// parseStringSlice 解析字符串切片
func parseStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case []string:
		return v
	case string:
		// 如果是JSON字符串，尝试解析
		var result []string
		if err := json.Unmarshal([]byte(v), &result); err == nil {
			return result
		}
		// 否则返回单个元素的切片
		return []string{v}
	default:
		return []string{}
	}
}
