package utils

import (
	"time"
)

// GetCurrentTimestamp 返回当前的 Unix 时间戳（秒）
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// GetCurrentTimestampMs 返回当前的 Unix 时间戳（毫秒）
func GetCurrentTimestampMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
