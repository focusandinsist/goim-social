package context

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// 上下文键类型
type contextKey string

const (
	// 业务相关的上下文键
	TraceIDKey   contextKey = "trace_id"
	UserIDKey    contextKey = "user_id"
	MessageIDKey contextKey = "message_id"
	GroupIDKey   contextKey = "group_id"
	SessionIDKey contextKey = "session_id"
	RequestIDKey contextKey = "request_id"
	
	// 服务相关的上下文键
	ServiceNameKey contextKey = "service_name"
	ServiceIDKey   contextKey = "service_id"
	ClientIPKey    contextKey = "client_ip"
	UserAgentKey   contextKey = "user_agent"
)

// TraceContext 业务追踪上下文
type TraceContext struct {
	TraceID   string
	UserID    int64
	MessageID int64
	GroupID   int64
	SessionID string
	RequestID string
}

// WithTraceID 在context中设置TraceID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		traceID = GenerateTraceID()
	}
	
	// 同时设置到OpenTelemetry span中
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.String("trace.id", traceID))
	}
	
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从context中获取TraceID
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	
	// 优先从OpenTelemetry span中获取
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	
	// 从context value中获取
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	
	return ""
}

// WithUserID 在context中设置UserID
func WithUserID(ctx context.Context, userID int64) context.Context {
	if userID <= 0 {
		return ctx
	}
	
	// 同时设置到OpenTelemetry span中
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int64("user.id", userID))
	}
	
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID 从context中获取UserID
func GetUserID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if userID, ok := ctx.Value(UserIDKey).(int64); ok {
		return userID
	}
	return 0
}

// WithMessageID 在context中设置MessageID
func WithMessageID(ctx context.Context, messageID int64) context.Context {
	if messageID <= 0 {
		return ctx
	}
	
	// 同时设置到OpenTelemetry span中
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int64("message.id", messageID))
	}
	
	return context.WithValue(ctx, MessageIDKey, messageID)
}

// GetMessageID 从context中获取MessageID
func GetMessageID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if messageID, ok := ctx.Value(MessageIDKey).(int64); ok {
		return messageID
	}
	return 0
}

// WithGroupID 在context中设置GroupID
func WithGroupID(ctx context.Context, groupID int64) context.Context {
	if groupID <= 0 {
		return ctx
	}
	
	// 同时设置到OpenTelemetry span中
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int64("group.id", groupID))
	}
	
	return context.WithValue(ctx, GroupIDKey, groupID)
}

// GetGroupID 从context中获取GroupID
func GetGroupID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if groupID, ok := ctx.Value(GroupIDKey).(int64); ok {
		return groupID
	}
	return 0
}

// WithSessionID 在context中设置SessionID
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	if sessionID == "" {
		return ctx
	}
	
	// 同时设置到OpenTelemetry span中
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.String("session.id", sessionID))
	}
	
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

// GetSessionID 从context中获取SessionID
func GetSessionID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if sessionID, ok := ctx.Value(SessionIDKey).(string); ok {
		return sessionID
	}
	return ""
}

// WithRequestID 在context中设置RequestID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		requestID = GenerateRequestID()
	}
	
	// 同时设置到OpenTelemetry span中
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.String("request.id", requestID))
	}
	
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID 从context中获取RequestID
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithServiceInfo 在context中设置服务信息
func WithServiceInfo(ctx context.Context, serviceName, serviceID string) context.Context {
	ctx = context.WithValue(ctx, ServiceNameKey, serviceName)
	ctx = context.WithValue(ctx, ServiceIDKey, serviceID)
	
	// 同时设置到OpenTelemetry span中
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.id", serviceID),
		)
	}
	
	return ctx
}

// GetServiceName 从context中获取服务名
func GetServiceName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if serviceName, ok := ctx.Value(ServiceNameKey).(string); ok {
		return serviceName
	}
	return ""
}

// GetServiceID 从context中获取服务ID
func GetServiceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if serviceID, ok := ctx.Value(ServiceIDKey).(string); ok {
		return serviceID
	}
	return ""
}

// WithClientInfo 在context中设置客户端信息
func WithClientInfo(ctx context.Context, clientIP, userAgent string) context.Context {
	ctx = context.WithValue(ctx, ClientIPKey, clientIP)
	ctx = context.WithValue(ctx, UserAgentKey, userAgent)
	
	// 同时设置到OpenTelemetry span中
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.String("client.ip", clientIP),
			attribute.String("client.user_agent", userAgent),
		)
	}
	
	return ctx
}

// GetClientIP 从context中获取客户端IP
func GetClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if clientIP, ok := ctx.Value(ClientIPKey).(string); ok {
		return clientIP
	}
	return ""
}

// GetUserAgent 从context中获取用户代理
func GetUserAgent(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if userAgent, ok := ctx.Value(UserAgentKey).(string); ok {
		return userAgent
	}
	return ""
}

// GenerateTraceID 生成TraceID
func GenerateTraceID() string {
	return uuid.New().String()
}

// GenerateRequestID 生成RequestID
func GenerateRequestID() string {
	return uuid.New().String()
}

// ExtractTraceContext 从context中提取业务追踪信息
func ExtractTraceContext(ctx context.Context) *TraceContext {
	return &TraceContext{
		TraceID:   GetTraceID(ctx),
		UserID:    GetUserID(ctx),
		MessageID: GetMessageID(ctx),
		GroupID:   GetGroupID(ctx),
		SessionID: GetSessionID(ctx),
		RequestID: GetRequestID(ctx),
	}
}

// ToMap 将TraceContext转换为map，用于日志输出
func (tc *TraceContext) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	
	if tc.TraceID != "" {
		result["trace_id"] = tc.TraceID
	}
	if tc.UserID > 0 {
		result["user_id"] = tc.UserID
	}
	if tc.MessageID > 0 {
		result["message_id"] = tc.MessageID
	}
	if tc.GroupID > 0 {
		result["group_id"] = tc.GroupID
	}
	if tc.SessionID != "" {
		result["session_id"] = tc.SessionID
	}
	if tc.RequestID != "" {
		result["request_id"] = tc.RequestID
	}
	
	return result
}

// ToString 将数值转换为字符串，用于兼容现有代码
func Int64ToString(val int64) string {
	if val <= 0 {
		return ""
	}
	return strconv.FormatInt(val, 10)
}
