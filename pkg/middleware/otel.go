package middleware

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/logger"
)

// OTelMiddleware OpenTelemetry中间件配置
type OTelMiddleware struct {
	serviceName string
	logger      logger.Logger
}

// NewOTelMiddleware 创建OpenTelemetry中间件
func NewOTelMiddleware(serviceName string, logger logger.Logger) *OTelMiddleware {
	return &OTelMiddleware{
		serviceName: serviceName,
		logger:      logger,
	}
}

// GinMiddleware 返回Gin的OpenTelemetry中间件
func (m *OTelMiddleware) GinMiddleware() gin.HandlerFunc {
	// 使用官方的otelgin中间件作为基础
	baseMiddleware := otelgin.Middleware(m.serviceName)

	return gin.HandlerFunc(func(c *gin.Context) {
		// 先执行基础的OpenTelemetry中间件
		baseMiddleware(c)

		// 增强context，添加业务信息
		ctx := m.enhanceContext(c.Request.Context(), c)
		c.Request = c.Request.WithContext(ctx)

		// 继续处理请求
		c.Next()
	})
}

// enhanceContext 增强context，添加业务追踪信息
func (m *OTelMiddleware) enhanceContext(ctx context.Context, c *gin.Context) context.Context {
	// 生成或提取TraceID
	traceID := c.GetHeader("X-Trace-ID")
	if traceID == "" {
		// 如果没有外部TraceID，使用OpenTelemetry的TraceID
		if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}
	}
	ctx = tracecontext.WithTraceID(ctx, traceID)

	// 提取RequestID
	requestID := c.GetHeader("X-Request-ID")
	ctx = tracecontext.WithRequestID(ctx, requestID)

	// 提取UserID（从认证中间件设置的值）
	if userIDVal, exists := c.Get("userID"); exists {
		if userID, ok := userIDVal.(int64); ok {
			ctx = tracecontext.WithUserID(ctx, userID)
		}
	}

	// 设置服务信息
	ctx = tracecontext.WithServiceInfo(ctx, m.serviceName, "")

	// 设置客户端信息
	ctx = tracecontext.WithClientInfo(ctx, c.ClientIP(), c.GetHeader("User-Agent"))

	// 将业务信息添加到OpenTelemetry span
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.route", c.FullPath()),
			attribute.String("http.user_agent", c.GetHeader("User-Agent")),
			attribute.String("http.client_ip", c.ClientIP()),
		)

		// 添加业务属性
		if userID := tracecontext.GetUserID(ctx); userID > 0 {
			span.SetAttributes(attribute.Int64("user.id", userID))
		}
	}

	return ctx
}

// GRPCUnaryServerInterceptor 返回gRPC一元服务器拦截器
func (m *OTelMiddleware) GRPCUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 增强context，添加业务信息
		ctx = m.enhanceGRPCContext(ctx, info)

		// 调用实际的处理器
		return handler(ctx, req)
	}
}

// GRPCStreamServerInterceptor 返回gRPC流服务器拦截器
func (m *OTelMiddleware) GRPCStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 增强context，添加业务信息
		ctx := m.enhanceGRPCContext(ss.Context(), info)

		// 创建包装的流
		wrappedStream := &otelWrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// 调用实际的处理器
		return handler(srv, wrappedStream)
	}
}

// enhanceGRPCContext 增强gRPC context
func (m *OTelMiddleware) enhanceGRPCContext(ctx context.Context, info interface{}) context.Context {
	// 从metadata中提取信息
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// 提取TraceID
		if traceIDs := md.Get("x-trace-id"); len(traceIDs) > 0 {
			ctx = tracecontext.WithTraceID(ctx, traceIDs[0])
		}

		// 提取RequestID
		if requestIDs := md.Get("x-request-id"); len(requestIDs) > 0 {
			ctx = tracecontext.WithRequestID(ctx, requestIDs[0])
		}

		// 提取UserID
		if userIDs := md.Get("x-user-id"); len(userIDs) > 0 {
			if userID, err := strconv.ParseInt(userIDs[0], 10, 64); err == nil {
				ctx = tracecontext.WithUserID(ctx, userID)
			}
		}

		// 提取MessageID
		if messageIDs := md.Get("x-message-id"); len(messageIDs) > 0 {
			if messageID, err := strconv.ParseInt(messageIDs[0], 10, 64); err == nil {
				ctx = tracecontext.WithMessageID(ctx, messageID)
			}
		}

		// 提取GroupID
		if groupIDs := md.Get("x-group-id"); len(groupIDs) > 0 {
			if groupID, err := strconv.ParseInt(groupIDs[0], 10, 64); err == nil {
				ctx = tracecontext.WithGroupID(ctx, groupID)
			}
		}
	}

	// 设置服务信息
	ctx = tracecontext.WithServiceInfo(ctx, m.serviceName, "")

	// 将业务信息添加到OpenTelemetry span
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		var methodName string
		switch v := info.(type) {
		case *grpc.UnaryServerInfo:
			methodName = v.FullMethod
		case *grpc.StreamServerInfo:
			methodName = v.FullMethod
		}

		span.SetAttributes(
			attribute.String("rpc.method", methodName),
			attribute.String("rpc.service", m.serviceName),
		)

		// 添加业务属性
		if userID := tracecontext.GetUserID(ctx); userID > 0 {
			span.SetAttributes(attribute.Int64("user.id", userID))
		}
		if messageID := tracecontext.GetMessageID(ctx); messageID > 0 {
			span.SetAttributes(attribute.Int64("message.id", messageID))
		}
		if groupID := tracecontext.GetGroupID(ctx); groupID > 0 {
			span.SetAttributes(attribute.Int64("group.id", groupID))
		}
	}

	return ctx
}

// GRPCUnaryClientInterceptor 返回gRPC一元客户端拦截器
func (m *OTelMiddleware) GRPCUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 将业务信息注入到metadata中
		ctx = m.injectBusinessMetadata(ctx)

		// 执行调用
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// GRPCStreamClientInterceptor 返回gRPC流客户端拦截器
func (m *OTelMiddleware) GRPCStreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// 将业务信息注入到metadata中
		ctx = m.injectBusinessMetadata(ctx)

		// 执行调用
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// injectBusinessMetadata 将业务信息注入到gRPC metadata中
func (m *OTelMiddleware) injectBusinessMetadata(ctx context.Context) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	if md == nil {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	// 注入业务信息
	if traceID := tracecontext.GetTraceID(ctx); traceID != "" {
		md.Set("x-trace-id", traceID)
	}
	if requestID := tracecontext.GetRequestID(ctx); requestID != "" {
		md.Set("x-request-id", requestID)
	}
	if userID := tracecontext.GetUserID(ctx); userID > 0 {
		md.Set("x-user-id", strconv.FormatInt(userID, 10))
	}
	if messageID := tracecontext.GetMessageID(ctx); messageID > 0 {
		md.Set("x-message-id", strconv.FormatInt(messageID, 10))
	}
	if groupID := tracecontext.GetGroupID(ctx); groupID > 0 {
		md.Set("x-group-id", strconv.FormatInt(groupID, 10))
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// otelWrappedServerStream 包装的服务器流，用于传递增强的context
type otelWrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *otelWrappedServerStream) Context() context.Context {
	return w.ctx
}
