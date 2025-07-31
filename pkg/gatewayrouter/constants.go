package gatewayrouter

import "time"

const (
	// ActiveGatewaysKey Redis ZSET键名，存储活跃网关实例
	ActiveGatewaysKey = "active_gateways"
	
	// GatewayInstanceHashKeyFmt 网关实例详细信息Hash键格式
	// 使用方式: fmt.Sprintf(GatewayInstanceHashKeyFmt, instanceID)
	GatewayInstanceHashKeyFmt = "gateway_instances:%s"
	
	// HeartbeatWindow 心跳窗口时间（秒），超过此时间认为实例不活跃
	HeartbeatWindow = 90
	
	// CleanupInterval 清理过期实例的间隔时间
	CleanupInterval = 60 * time.Second
	
	// HeartbeatInterval 心跳发送间隔时间
	HeartbeatInterval = 30 * time.Second
	
	// SyncInterval 同步间隔时间
	SyncInterval = 10 * time.Second
)
