package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel/trace"
	"goim-social/pkg/logger"
)

// LokiLogger Loki集成的日志器
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

// LogEntry 结构化日志条目
type LogEntry struct {
	Level       string                 `json:"level"`
	Time        string                 `json:"time"`
	Message     string                 `json:"msg"`
	ServiceName string                 `json:"service_name"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	UserID      int64                  `json:"user_id,omitempty"`
	GroupID     int64                  `json:"group_id,omitempty"`
	MessageID   int64                  `json:"message_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

// Info 记录信息日志
func (l *LokiLogger) Info(ctx context.Context, msg string, fields ...logger.Field) {
	l.log(ctx, "info", msg, fields...)
	if l.baseLogger != nil {
		l.baseLogger.Info(ctx, msg, fields...)
	}
}

// Error 记录错误日志
func (l *LokiLogger) Error(ctx context.Context, msg string, fields ...logger.Field) {
	l.log(ctx, "error", msg, fields...)
	if l.baseLogger != nil {
		l.baseLogger.Error(ctx, msg, fields...)
	}
}

// Warn 记录警告日志
func (l *LokiLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
	l.log(ctx, "warn", msg, fields...)
	if l.baseLogger != nil {
		l.baseLogger.Warn(ctx, msg, fields...)
	}
}

// Debug 记录调试日志
func (l *LokiLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
	l.log(ctx, "debug", msg, fields...)
	if l.baseLogger != nil {
		l.baseLogger.Debug(ctx, msg, fields...)
	}
}

// log 内部日志记录方法
func (l *LokiLogger) log(ctx context.Context, level, msg string, fields ...logger.Field) {
	if !l.enabled {
		return
	}

	entry := LogEntry{
		Level:       level,
		Time:        time.Now().Format(time.RFC3339Nano),
		Message:     msg,
		ServiceName: l.serviceName,
		Fields:      make(map[string]interface{}),
	}

	// 提取OpenTelemetry信息
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		entry.TraceID = span.SpanContext().TraceID().String()
		entry.SpanID = span.SpanContext().SpanID().String()
	}

	// 提取业务信息
	if userID := getUserIDFromContext(ctx); userID > 0 {
		entry.UserID = userID
	}
	if groupID := getGroupIDFromContext(ctx); groupID > 0 {
		entry.GroupID = groupID
	}
	if messageID := getMessageIDFromContext(ctx); messageID > 0 {
		entry.MessageID = messageID
	}
	if requestID := getRequestIDFromContext(ctx); requestID != "" {
		entry.RequestID = requestID
	}

	// 处理额外字段
	for _, field := range fields {
		entry.Fields[field.Key] = field.Value
	}

	// 输出JSON格式日志（Promtail会自动收集）
	if jsonData, err := json.Marshal(entry); err == nil {
		fmt.Println(string(jsonData))
	}
}

// 从context中提取业务信息的辅助函数
func getUserIDFromContext(ctx context.Context) int64 {
	if userID, ok := ctx.Value("user_id").(int64); ok {
		return userID
	}
	return 0
}

func getGroupIDFromContext(ctx context.Context) int64 {
	if groupID, ok := ctx.Value("group_id").(int64); ok {
		return groupID
	}
	return 0
}

func getMessageIDFromContext(ctx context.Context) int64 {
	if messageID, ok := ctx.Value("message_id").(int64); ok {
		return messageID
	}
	return 0
}

func getRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// WithUserID 在context中设置用户ID
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, "user_id", userID)
}

// WithGroupID 在context中设置群组ID
func WithGroupID(ctx context.Context, groupID int64) context.Context {
	return context.WithValue(ctx, "group_id", groupID)
}

// WithMessageID 在context中设置消息ID
func WithMessageID(ctx context.Context, messageID int64) context.Context {
	return context.WithValue(ctx, "message_id", messageID)
}

// WithRequestID 在context中设置请求ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, "request_id", requestID)
}
