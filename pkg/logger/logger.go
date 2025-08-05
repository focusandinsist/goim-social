package logger

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	tracecontext "goim-social/pkg/context"
)

// Logger 日志接口
type Logger interface {
	Info(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Debug(ctx context.Context, msg string, fields ...Field)
	Fatal(ctx context.Context, msg string, fields ...Field)
	WithContext(ctx context.Context) Logger
}

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// logger 日志实现
type logger struct {
	zapLogger *zap.Logger
}

// NewLogger 创建日志实例
func NewLogger(level string) (Logger, error) {
	// 设置日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "fatal":
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 配置zap
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &logger{zapLogger: zapLogger}, nil
}

// Info 信息日志
func (l *logger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.InfoLevel, msg, fields...)
}

// Error 错误日志
func (l *logger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.ErrorLevel, msg, fields...)
}

// Warn 警告日志
func (l *logger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.WarnLevel, msg, fields...)
}

// Debug 调试日志
func (l *logger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.DebugLevel, msg, fields...)
}

// Fatal 致命错误日志
func (l *logger) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, zapcore.FatalLevel, msg, fields...)
}

// WithContext 带上下文的日志
func (l *logger) WithContext(ctx context.Context) Logger {
	return &logger{zapLogger: l.zapLogger.With(l.extractFields(ctx)...)}
}

// log 内部日志方法
func (l *logger) log(ctx context.Context, level zapcore.Level, msg string, fields ...Field) {
	zapFields := make([]zap.Field, 0, len(fields)+10) // 预分配更多空间

	// 添加OpenTelemetry trace信息
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		zapFields = append(zapFields,
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// 添加业务追踪信息
	traceCtx := tracecontext.ExtractTraceContext(ctx)
	for key, value := range traceCtx.ToMap() {
		zapFields = append(zapFields, zap.Any(key, value))
	}

	// 添加服务信息
	if serviceName := tracecontext.GetServiceName(ctx); serviceName != "" {
		zapFields = append(zapFields, zap.String("service_name", serviceName))
	}
	if serviceID := tracecontext.GetServiceID(ctx); serviceID != "" {
		zapFields = append(zapFields, zap.String("service_id", serviceID))
	}

	// 添加客户端信息
	if clientIP := tracecontext.GetClientIP(ctx); clientIP != "" {
		zapFields = append(zapFields, zap.String("client_ip", clientIP))
	}

	// 添加时间戳
	zapFields = append(zapFields, zap.Time("timestamp", time.Now()))

	// 添加自定义字段
	for _, field := range fields {
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}

	switch level {
	case zapcore.InfoLevel:
		l.zapLogger.Info(msg, zapFields...)
	case zapcore.ErrorLevel:
		l.zapLogger.Error(msg, zapFields...)
	case zapcore.WarnLevel:
		l.zapLogger.Warn(msg, zapFields...)
	case zapcore.DebugLevel:
		l.zapLogger.Debug(msg, zapFields...)
	case zapcore.FatalLevel:
		l.zapLogger.Fatal(msg, zapFields...)
	}
}

// extractFields 从上下文提取字段
func (l *logger) extractFields(ctx context.Context) []zap.Field {
	fields := make([]zap.Field, 0, 10)

	// 添加OpenTelemetry trace信息
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		fields = append(fields,
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// 添加业务追踪信息
	traceCtx := tracecontext.ExtractTraceContext(ctx)
	for key, value := range traceCtx.ToMap() {
		fields = append(fields, zap.Any(key, value))
	}

	// 添加服务信息
	if serviceName := tracecontext.GetServiceName(ctx); serviceName != "" {
		fields = append(fields, zap.String("service_name", serviceName))
	}
	if serviceID := tracecontext.GetServiceID(ctx); serviceID != "" {
		fields = append(fields, zap.String("service_id", serviceID))
	}

	// 兼容旧的方法
	if userID := getUserID(ctx); userID != "" {
		fields = append(fields, zap.String("user_id_legacy", userID))
	}
	if serviceName := getServiceName(ctx); serviceName != "" {
		fields = append(fields, zap.String("service_legacy", serviceName))
	}

	return fields
}

// 辅助函数
func getRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

func getUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}

func getServiceName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if serviceName, ok := ctx.Value("service_name").(string); ok {
		return serviceName
	}
	return ""
}

// 便捷函数
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// 默认日志实例
var defaultLogger Logger

// Init 初始化默认日志
func Init(level string) error {
	var err error
	defaultLogger, err = NewLogger(level)
	return err
}

// GetLogger 获取默认日志实例
func GetLogger() Logger {
	if defaultLogger == nil {
		// 使用标准库作为fallback
		log.Println("Warning: Using fallback logger")
		return &fallbackLogger{}
	}
	return defaultLogger
}

// fallbackLogger 备用日志实现
type fallbackLogger struct{}

func (l *fallbackLogger) Info(ctx context.Context, msg string, fields ...Field) {
	log.Printf("[INFO] %s", msg)
}

func (l *fallbackLogger) Error(ctx context.Context, msg string, fields ...Field) {
	log.Printf("[ERROR] %s", msg)
}

func (l *fallbackLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	log.Printf("[WARN] %s", msg)
}

func (l *fallbackLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	log.Printf("[DEBUG] %s", msg)
}

func (l *fallbackLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	log.Fatalf("[FATAL] %s", msg)
}

func (l *fallbackLogger) WithContext(ctx context.Context) Logger {
	return l
}
