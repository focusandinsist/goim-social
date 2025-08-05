package telemetry

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config OpenTelemetry配置
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExporterType   string // "stdout", "jaeger", "otlp"
	JaegerEndpoint string
	OTLPEndpoint   string
	SampleRate     float64 // 采样率 0.0-1.0
}

// DefaultConfig 返回默认配置
func DefaultConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   "noop", // 默认不输出，避免控制台污染
		SampleRate:     1.0,
	}
}

// DevelopmentConfig 返回开发环境配置（会输出到控制台）
func DevelopmentConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   "stdout",
		SampleRate:     1.0, // 开发环境全采样
	}
}

// ProductionConfig 返回生产环境配置
func ProductionConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "production",
		ExporterType:   "jaeger", // 生产环境使用Jaeger
		JaegerEndpoint: "http://localhost:14268/api/traces",
		SampleRate:     0.1, // 生产环境10%采样
	}
}

// Provider OpenTelemetry提供者
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	config         *Config
}

// NewProvider 创建OpenTelemetry提供者
func NewProvider(config *Config) (*Provider, error) {
	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建导出器
	exporter, err := createExporter(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// 创建采样器
	sampler := sdktrace.AlwaysSample()
	if config.SampleRate < 1.0 {
		sampler = sdktrace.TraceIDRatioBased(config.SampleRate)
	}

	// 创建TracerProvider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局TracerProvider
	otel.SetTracerProvider(tracerProvider)

	// 设置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 创建Tracer
	tracer := otel.Tracer(config.ServiceName)

	provider := &Provider{
		tracerProvider: tracerProvider,
		tracer:         tracer,
		config:         config,
	}

	log.Printf("OpenTelemetry initialized for service: %s, exporter: %s",
		config.ServiceName, config.ExporterType)

	return provider, nil
}

// createExporter 创建导出器
func createExporter(config *Config) (sdktrace.SpanExporter, error) {
	switch config.ExporterType {
	case "stdout":
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
	case "noop":
		// 返回一个不做任何操作的导出器
		return &noopExporter{}, nil
	case "jaeger":
		// TODO: 实现Jaeger导出器
		return nil, fmt.Errorf("jaeger exporter not implemented yet")
	case "otlp":
		// TODO: 实现OTLP导出器
		return nil, fmt.Errorf("otlp exporter not implemented yet")
	default:
		// 默认使用noop，避免意外的控制台输出
		return &noopExporter{}, nil
	}
}

// noopExporter 不做任何操作的导出器
type noopExporter struct{}

func (e *noopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	// 不做任何操作，直接返回成功
	return nil
}

func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}

// GetTracer 获取Tracer
func (p *Provider) GetTracer() trace.Tracer {
	return p.tracer
}

// Shutdown 关闭Provider
func (p *Provider) Shutdown(ctx context.Context) error {
	return p.tracerProvider.Shutdown(ctx)
}

// StartSpan 开始一个新的span
func (p *Provider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return p.tracer.Start(ctx, name, opts...)
}

// 全局Provider实例
var globalProvider *Provider

// InitGlobal 初始化全局Provider
func InitGlobal(config *Config) error {
	var err error
	globalProvider, err = NewProvider(config)
	return err
}

// GetGlobalProvider 获取全局Provider
func GetGlobalProvider() *Provider {
	return globalProvider
}

// GetGlobalTracer 获取全局Tracer
func GetGlobalTracer() trace.Tracer {
	if globalProvider == nil {
		log.Println("Warning: OpenTelemetry not initialized, using NoOp tracer")
		return otel.Tracer("noop")
	}
	return globalProvider.GetTracer()
}

// StartSpan 使用全局tracer开始span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := GetGlobalTracer()
	return tracer.Start(ctx, name, opts...)
}

// ShutdownGlobal 关闭全局Provider
func ShutdownGlobal(ctx context.Context) error {
	if globalProvider != nil {
		return globalProvider.Shutdown(ctx)
	}
	return nil
}
