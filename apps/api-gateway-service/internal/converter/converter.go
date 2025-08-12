package converter

import (
	"time"

	"goim-social/api/rest"
	"goim-social/apps/api-gateway-service/internal/model"
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

// OnlineStatusResultToProto 将在线状态结果Model转换为Protobuf
func (c *Converter) OnlineStatusResultToProto(result *model.OnlineStatusResult) *rest.OnlineStatusResponse {
	if result == nil {
		return &rest.OnlineStatusResponse{
			Status: make(map[int64]bool),
		}
	}

	return &rest.OnlineStatusResponse{
		Status: result.Status,
	}
}

// OnlineStatusParamsFromProto 将Protobuf转换为在线状态查询参数Model
func (c *Converter) OnlineStatusParamsFromProto(req *rest.OnlineStatusRequest) *model.OnlineStatusParams {
	if req == nil {
		return &model.OnlineStatusParams{
			UserIDs: []int64{},
		}
	}

	return &model.OnlineStatusParams{
		UserIDs: req.UserIds,
	}
}

// 响应构建方法

// BuildOnlineStatusResponse 构建在线状态响应
func (c *Converter) BuildOnlineStatusResponse(status map[int64]bool) *rest.OnlineStatusResponse {
	if status == nil {
		status = make(map[int64]bool)
	}

	return &rest.OnlineStatusResponse{
		Status: status,
	}
}

// BuildEmptyOnlineStatusResponse 构建空的在线状态响应
func (c *Converter) BuildEmptyOnlineStatusResponse() *rest.OnlineStatusResponse {
	return &rest.OnlineStatusResponse{
		Status: make(map[int64]bool),
	}
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
			"count":  len(status),
		},
	}
}

// BuildHTTPServicesResponse 构建HTTP服务列表响应
func (c *Converter) BuildHTTPServicesResponse(services interface{}) map[string]interface{} {
	if services == nil {
		services = make(map[string]interface{})
	}

	// 计算服务数量
	var count int
	switch v := services.(type) {
	case map[string]interface{}:
		count = len(v)
	case []string:
		count = len(v)
	default:
		count = 0
	}

	return map[string]interface{}{
		"success": true,
		"message": "获取成功",
		"data": map[string]interface{}{
			"services": services,
			"count":    count,
		},
	}
}

// BuildHTTPHealthResponse 构建HTTP健康检查响应
func (c *Converter) BuildHTTPHealthResponse(version string) map[string]interface{} {
	return map[string]interface{}{
		"status":    "healthy",
		"service":   "api-gateway-service",
		"version":   version,
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

// BuildHTTPErrorResponse 构建HTTP错误响应
func (c *Converter) BuildHTTPErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": message,
		"data":    nil,
	}
}

// BuildHTTPSuccessResponse 构建HTTP成功响应
func (c *Converter) BuildHTTPSuccessResponse(message string, data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": message,
		"data":    data,
	}
}

// 动态路由相关响应构建方法

// BuildHTTPProxyErrorResponse 构建HTTP代理错误响应
func (c *Converter) BuildHTTPProxyErrorResponse(service string, message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": message,
		"service": service,
		"error":   "proxy_error",
	}
}

// BuildHTTPServiceNotFoundResponse 构建HTTP服务未找到响应
func (c *Converter) BuildHTTPServiceNotFoundResponse(service string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": "服务未找到或不可用",
		"service": service,
		"error":   "service_not_found",
	}
}

// BuildHTTPInvalidRouteResponse 构建HTTP无效路由响应
func (c *Converter) BuildHTTPInvalidRouteResponse(path string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": "无效的路由路径",
		"path":    path,
		"error":   "invalid_route",
	}
}

// 连接管理相关响应构建方法

// BuildHTTPConnectionsResponse 构建HTTP连接列表响应
func (c *Converter) BuildHTTPConnectionsResponse(connections []*model.Connection) map[string]interface{} {
	connectionsData := c.ConnectionModelsToMaps(connections)

	return map[string]interface{}{
		"success": true,
		"message": "获取成功",
		"data": map[string]interface{}{
			"connections": connectionsData,
			"count":       len(connectionsData),
		},
	}
}

// BuildHTTPConnectionResponse 构建HTTP单个连接响应
func (c *Converter) BuildHTTPConnectionResponse(connection *model.Connection) map[string]interface{} {
	connectionData := c.ConnectionModelToMap(connection)

	return map[string]interface{}{
		"success": true,
		"message": "获取成功",
		"data":    connectionData,
	}
}

// 统计信息响应构建方法

// BuildHTTPStatsResponse 构建HTTP统计信息响应
func (c *Converter) BuildHTTPStatsResponse(stats map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": "获取成功",
		"data":    stats,
	}
}

// BuildHTTPGatewayStatsResponse 构建HTTP网关统计响应
func (c *Converter) BuildHTTPGatewayStatsResponse(totalRequests, successRequests, errorRequests int64, uptime string) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": "获取成功",
		"data": map[string]interface{}{
			"total_requests":   totalRequests,
			"success_requests": successRequests,
			"error_requests":   errorRequests,
			"success_rate":     c.calculateSuccessRate(successRequests, totalRequests),
			"uptime":           uptime,
			"timestamp":        time.Now().Format(time.RFC3339),
		},
	}
}

// calculateSuccessRate 计算成功率
func (c *Converter) calculateSuccessRate(success, total int64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(success) / float64(total) * 100.0
}

// 便捷方法：常用错误响应

// BuildHTTPBadRequestResponse 构建HTTP错误请求响应
func (c *Converter) BuildHTTPBadRequestResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": "请求参数错误: " + message,
		"error":   "bad_request",
	}
}

// BuildHTTPInternalErrorResponse 构建HTTP内部错误响应
func (c *Converter) BuildHTTPInternalErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": "内部服务错误: " + message,
		"error":   "internal_error",
	}
}

// BuildHTTPTimeoutResponse 构建HTTP超时响应
func (c *Converter) BuildHTTPTimeoutResponse(service string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": "服务请求超时",
		"service": service,
		"error":   "timeout",
	}
}
