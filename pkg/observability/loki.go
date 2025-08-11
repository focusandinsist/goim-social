package observability

import (
	"context"
	"os"

	"goim-social/pkg/logger"
)

// 类型安全的 context keys
// 使用未导出的类型，确保外部包无法创建相同的键，避免冲突
type contextKey int

const (
	userIDKey contextKey = iota
	groupIDKey
	messageIDKey
	requestIDKey
	serviceNameKey
	clientIPKey
)

// LokiLogger 高性能的可观测性日志器（基于现有的 zap logger）
// 1. 直接使用高性能的 zap logger，2. 避免阻塞 IO 操作
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
		// 添加业务上下文信息到字段中
		enrichedFields := l.enrichFieldsWithContext(ctx, fields...)
		// 使用高性能 zap logger，它已经处理了所有的性能优化
		l.baseLogger.Info(ctx, msg, enrichedFields...)
	}
}

// Error 记录错误日志
func (l *LokiLogger) Error(ctx context.Context, msg string, fields ...logger.Field) {
	if l.enabled && l.baseLogger != nil {
		enrichedFields := l.enrichFieldsWithContext(ctx, fields...)
		l.baseLogger.Error(ctx, msg, enrichedFields...)
	}
}

// Warn 记录警告日志
func (l *LokiLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
	if l.enabled && l.baseLogger != nil {
		enrichedFields := l.enrichFieldsWithContext(ctx, fields...)
		l.baseLogger.Warn(ctx, msg, enrichedFields...)
	}
}

// Debug 记录调试日志
func (l *LokiLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
	if l.enabled && l.baseLogger != nil {
		enrichedFields := l.enrichFieldsWithContext(ctx, fields...)
		l.baseLogger.Debug(ctx, msg, enrichedFields...)
	}
}

// WithContext 带上下文的日志器
func (l *LokiLogger) WithContext(ctx context.Context) logger.Logger {
	if l.enabled && l.baseLogger != nil {
		return l.baseLogger.WithContext(ctx)
	}
	return l.baseLogger
}

// 类型安全的 context 操作函数

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

// WithServiceName 在context中设置服务名称（类型安全）
func WithServiceName(ctx context.Context, serviceName string) context.Context {
	return context.WithValue(ctx, serviceNameKey, serviceName)
}

// WithClientIP 在context中设置客户端IP（类型安全）
func WithClientIP(ctx context.Context, clientIP string) context.Context {
	return context.WithValue(ctx, clientIPKey, clientIP)
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

func GetServiceNameFromContext(ctx context.Context) (string, bool) {
	if serviceName, ok := ctx.Value(serviceNameKey).(string); ok {
		return serviceName, true
	}
	return "", false
}

func GetClientIPFromContext(ctx context.Context) (string, bool) {
	if clientIP, ok := ctx.Value(clientIPKey).(string); ok {
		return clientIP, true
	}
	return "", false
}

// enrichFieldsWithContext 从context中提取业务信息并添加到日志字段中
func (l *LokiLogger) enrichFieldsWithContext(ctx context.Context, fields ...logger.Field) []logger.Field {
	// 预分配足够的空间，避免多次扩容
	enrichedFields := make([]logger.Field, 0, len(fields)+10)

	// 先添加原有字段
	enrichedFields = append(enrichedFields, fields...)

	// 安全地添加业务上下文信息，每个字段独立处理，避免一个失败影响其他
	if userID, ok := GetUserIDFromContext(ctx); ok {
		enrichedFields = append(enrichedFields, logger.F("user_id", userID))
	}

	if groupID, ok := GetGroupIDFromContext(ctx); ok {
		enrichedFields = append(enrichedFields, logger.F("group_id", groupID))
	}

	if messageID, ok := GetMessageIDFromContext(ctx); ok {
		enrichedFields = append(enrichedFields, logger.F("message_id", messageID))
	}

	if requestID, ok := GetRequestIDFromContext(ctx); ok {
		enrichedFields = append(enrichedFields, logger.F("request_id", requestID))
	}

	if serviceName, ok := GetServiceNameFromContext(ctx); ok {
		enrichedFields = append(enrichedFields, logger.F("service_name", serviceName))
	}

	if clientIP, ok := GetClientIPFromContext(ctx); ok {
		enrichedFields = append(enrichedFields, logger.F("client_ip", clientIP))
	}

	return enrichedFields
}
