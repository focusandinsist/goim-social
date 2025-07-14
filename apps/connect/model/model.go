package model

// WSMessage represents a WebSocket message structure.
type WSMessage struct {
	MessageType int         `json:"message_type"`
	Content     interface{} `json:"content"`
	// Add other fields as needed
}

type Connection struct {
	UserID        int64  `json:"user_id"`
	ConnID        string `json:"conn_id"`
	Online        bool   `json:"online"`
	RemoteIP      string `json:"remote_ip"`
	ServerID      string `json:"server_id"`
	Timestamp     int64  `json:"timestamp"`
	LastHeartbeat int64  `json:"last_heartbeat"`
	ClientType    string `json:"client_type"`
}

type ConnectRequest struct {
	UserID     int64  `json:"user_id"`
	Token      string `json:"token"`
	ServerID   string `json:"server_id"`
	ClientType string `json:"client_type"`
	RemoteIP   string `json:"remote_ip"`
}

type DisconnectRequest struct {
	UserID int64  `json:"user_id"`
	ConnID string `json:"conn_id"`
}

type HeartbeatRequest struct {
	UserID int64  `json:"user_id"`
	ConnID string `json:"conn_id"`
}

type OnlineStatusRequest struct {
	UserIDs []int64 `json:"user_ids"`
}
