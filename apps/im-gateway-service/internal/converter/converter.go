package converter

import (
	"time"

	"goim-social/api/rest"
	"goim-social/apps/im-gateway-service/internal/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// ConnectionModelToMap 将连接Model转换为Map格式（用于HTTP响应）
func (c *Converter) ConnectionModelToMap(conn *model.Connection) map[string]interface{} {
	if conn == nil {
		return nil
	}

	return map[string]interface{}{
		"user_id":        conn.UserID,
		"conn_id":        conn.ConnID,
		"server_id":      conn.ServerID,
		"timestamp":      conn.Timestamp,
		"last_heartbeat": conn.LastHeartbeat,
		"client_type":    conn.ClientType,
		"online":         conn.Online,
		"remote_ip":      conn.RemoteIP,
	}
}

// ConnectionModelsToMaps 将连接Model列表转换为Map列表
func (c *Converter) ConnectionModelsToMaps(connections []*model.Connection) []map[string]interface{} {
	if connections == nil {
		return []map[string]interface{}{}
	}

	result := make([]map[string]interface{}, 0, len(connections))
	for _, conn := range connections {
		if connMap := c.ConnectionModelToMap(conn); connMap != nil {
			result = append(result, connMap)
		}
	}
	return result
}

// OnlineStatusResultToProto 将在线状态结果转换为Protobuf
func (c *Converter) OnlineStatusResultToProto(status map[int64]bool) *rest.OnlineStatusResponse {
	if status == nil {
		status = make(map[int64]bool)
	}

	return &rest.OnlineStatusResponse{
		Status: status,
	}
}

// OnlineStatusParamsFromProto 将Protobuf转换为在线状态查询参数
func (c *Converter) OnlineStatusParamsFromProto(req *rest.OnlineStatusRequest) []int64 {
	if req == nil {
		return []int64{}
	}

	return req.UserIds
}

// 响应构建方法

// BuildOnlineStatusResponse 构建在线状态响应
func (c *Converter) BuildOnlineStatusResponse(status map[int64]bool) *rest.OnlineStatusResponse {
	return c.OnlineStatusResultToProto(status)
}

// BuildEmptyOnlineStatusResponse 构建空的在线状态响应
func (c *Converter) BuildEmptyOnlineStatusResponse() *rest.OnlineStatusResponse {
	return &rest.OnlineStatusResponse{
		Status: make(map[int64]bool),
	}
}

// BuildErrorOnlineStatusResponse 构建在线状态错误响应
func (c *Converter) BuildErrorOnlineStatusResponse() *rest.OnlineStatusResponse {
	return c.BuildEmptyOnlineStatusResponse()
}

// HTTP响应构建方法

// BuildHTTPOnlineStatusResponse 构建HTTP在线状态响应
func (c *Converter) BuildHTTPOnlineStatusResponse(status map[int64]bool) map[string]interface{} {
	if status == nil {
		status = make(map[int64]bool)
	}

	return map[string]interface{}{
		"success": true,
		"message": "查询成功",
		"data": map[string]interface{}{
			"status": status,
		},
	}
}

// BuildHTTPErrorOnlineStatusResponse 构建HTTP在线状态错误响应
func (c *Converter) BuildHTTPErrorOnlineStatusResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": message,
		"data": map[string]interface{}{
			"status": make(map[int64]bool),
		},
	}
}

// BuildHTTPInvalidRequestResponse 构建HTTP无效请求响应
func (c *Converter) BuildHTTPInvalidRequestResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": "请求参数无效: " + message,
		"error":   "invalid_request",
	}
}

// BuildHTTPConnectionStatsResponse 构建HTTP连接统计响应
func (c *Converter) BuildHTTPConnectionStatsResponse(totalConnections, activeConnections int64) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": "获取成功",
		"data": map[string]interface{}{
			"total_connections":  totalConnections,
			"active_connections": activeConnections,
			"timestamp":          time.Now().Format(time.RFC3339),
		},
	}
}

// BuildHTTPConnectionListResponse 构建HTTP连接列表响应
func (c *Converter) BuildHTTPConnectionListResponse(connections []*model.Connection) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": "获取成功",
		"data": map[string]interface{}{
			"connections": c.ConnectionModelsToMaps(connections),
			"count":       len(connections),
			"timestamp":   time.Now().Format(time.RFC3339),
		},
	}
}

// BuildHTTPHealthResponse 构建HTTP健康检查响应
func (c *Converter) BuildHTTPHealthResponse(serviceName string, timestamp int64) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": "服务健康",
		"data": map[string]interface{}{
			"service":   serviceName,
			"status":    "healthy",
			"timestamp": timestamp,
			"uptime":    time.Now().Format(time.RFC3339),
		},
	}
}

// BuildHTTPWebSocketUpgradeErrorResponse 构建WebSocket升级错误响应
func (c *Converter) BuildHTTPWebSocketUpgradeErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": "WebSocket升级失败: " + message,
		"error":   "websocket_upgrade_failed",
	}
}

// BuildHTTPMessageForwardErrorResponse 构建消息转发错误响应
func (c *Converter) BuildHTTPMessageForwardErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": "消息转发失败: " + message,
		"error":   "message_forward_failed",
	}
}

// BuildHTTPServiceUnavailableResponse 构建服务不可用响应
func (c *Converter) BuildHTTPServiceUnavailableResponse(serviceName string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": serviceName + " 服务暂时不可用",
		"error":   "service_unavailable",
	}
}

// 便捷方法：构建成功响应

// BuildSuccessOnlineStatusResponse 构建在线状态成功响应
func (c *Converter) BuildSuccessOnlineStatusResponse(status map[int64]bool) *rest.OnlineStatusResponse {
	return c.BuildOnlineStatusResponse(status)
}

// 便捷方法：构建错误响应

// BuildErrorOnlineStatusResponseWithMessage 构建带消息的在线状态错误响应
func (c *Converter) BuildErrorOnlineStatusResponseWithMessage(message string) *rest.OnlineStatusResponse {
	// gRPC响应中通常不包含错误消息，错误通过error返回
	return c.BuildEmptyOnlineStatusResponse()
}
