package model

// Connection 连接信息
type Connection struct {
	UserID        int64  `json:"user_id"`
	ConnID        string `json:"conn_id"`
	ServerID      string `json:"server_id"`
	Timestamp     int64  `json:"timestamp"`
	LastHeartbeat int64  `json:"last_heartbeat"`
	ClientType    string `json:"client_type"`
	Online        bool   `json:"online"`
}

// OnlineStatusParams 在线状态查询参数
type OnlineStatusParams struct {
	UserIDs []int64 `json:"user_ids"`
}

// OnlineStatusResult 在线状态查询结果
type OnlineStatusResult struct {
	Status map[int64]bool `json:"status"`
}
