package observability

import (
	"context"
	"os"

	"goim-social/pkg/logger"
)

// 类型安全的 context keys（解决同事提出的类型安全问题）
type contextKey string

const (
	userIDKey    contextKey = "observability.user_id"
	groupIDKey   contextKey = "observability.group_id"
	messageIDKey contextKey = "observability.message_id"
	requestIDKey contextKey = "observability.request_id"
)

// LokiLogger 高性能的可观测性日志器（基于现有的 zap logger）
// 解决了同事提出的所有性能问题：
// 1. 不再使用低效的 json.Marshal 和 fmt.Println
// 2. 直接使用高性能的 zap logger
// 3. 避免阻塞 IO 操作
type LokiLogger struct {
	serviceName string
	baseLogger  logger.Logger
	enabled     bool
}

// NewLokiLogger 创建Loki日志器
func NewLokiLogger(serviceName string, baseLogger logger.Logger) *LokiLogger {
	enabled := os.Getenv("LOKI_ENABLED") != "false" // 默认启用
	
	return &LokiLogger{
		serviceName: serviceName,
		baseLogger:  baseLogger,
		enabled:     enabled,
	}
}

// Info 记录信息日志
func (l *LokiLogger) Info(ctx context.Context, msg string, fields ...logger.Field) {
	if l.enabled && l.baseLogger != nil {
		// 直接使用现有的高性能 zap logger，它已经处理了所有的性能优化
		// 包括：异步写入、缓冲IO、零分配JSON编码、错误处理等
		l.baseLogger.Info(ctx, msg, fields...)
	}
}

// Error 记录错误日志
func (l *LokiLogger) Error(ctx context.Context, msg string, fields ...logger.Field) {
	if l.enabled && l.baseLogger != nil {
		l.baseLogger.Error(ctx, msg, fields...)
	}
}

// Warn 记录警告日志
func (l *LokiLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
	if l.enabled && l.baseLogger != nil {
		l.baseLogger.Warn(ctx, msg, fields...)
	}
}

// Debug 记录调试日志
func (l *LokiLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
	if l.enabled && l.baseLogger != nil {
		l.baseLogger.Debug(ctx, msg, fields...)
	}
}

// WithContext 带上下文的日志器
func (l *LokiLogger) WithContext(ctx context.Context) logger.Logger {
	if l.enabled && l.baseLogger != nil {
		return l.baseLogger.WithContext(ctx)
	}
	return l.baseLogger
}

// 类型安全的 context 操作函数（解决同事提出的类型安全问题）

// WithUserID 在context中设置用户ID（类型安全）
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithGroupID 在context中设置群组ID（类型安全）
func WithGroupID(ctx context.Context, groupID int64) context.Context {
	return context.WithValue(ctx, groupIDKey, groupID)
}

// WithMessageID 在context中设置消息ID（类型安全）
func WithMessageID(ctx context.Context, messageID int64) context.Context {
	return context.WithValue(ctx, messageIDKey, messageID)
}

// WithRequestID 在context中设置请求ID（类型安全）
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// 从context中提取业务信息的辅助函数（类型安全）

func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	if userID, ok := ctx.Value(userIDKey).(int64); ok {
		return userID, true
	}
	return 0, false
}

func GetGroupIDFromContext(ctx context.Context) (int64, bool) {
	if groupID, ok := ctx.Value(groupIDKey).(int64); ok {
		return groupID, true
	}
	return 0, false
}

func GetMessageIDFromContext(ctx context.Context) (int64, bool) {
	if messageID, ok := ctx.Value(messageIDKey).(int64); ok {
		return messageID, true
	}
	return 0, false
}

func GetRequestIDFromContext(ctx context.Context) (string, bool) {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID, true
	}
	return "", false
}
