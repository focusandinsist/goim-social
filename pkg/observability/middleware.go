package observability

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"goim-social/pkg/logger"
)

// ObservabilityMiddleware 统一的可观测性中间件
type ObservabilityMiddleware struct {
	serviceName string
	logger      *LokiLogger
	tracer      trace.Tracer
}

// NewObservabilityMiddleware 创建可观测性中间件
func NewObservabilityMiddleware(serviceName string, lokiLogger *LokiLogger) *ObservabilityMiddleware {
	return &ObservabilityMiddleware{
		serviceName: serviceName,
		logger:      lokiLogger,
		tracer:      otel.Tracer(serviceName),
	}
}

// GinMiddleware Gin HTTP中间件
func (m *ObservabilityMiddleware) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成或获取请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 创建span
		ctx, span := m.tracer.Start(c.Request.Context(), c.Request.Method+" "+c.Request.URL.Path)
		defer span.End()

		// 设置span属性
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.route", c.FullPath()),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("request.id", requestID),
		)

		// 在context中设置业务信息（类型安全）
		ctx = WithRequestID(ctx, requestID)
		ctx = WithClientIP(ctx, c.ClientIP())

		// 从JWT或其他方式获取用户ID
		if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
			if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
				ctx = WithUserID(ctx, userID)
				span.SetAttributes(attribute.Int64("user.id", userID))
			}
		}

		// 更新请求context
		c.Request = c.Request.WithContext(ctx)

		// 记录请求开始
		start := time.Now()
		m.logger.Info(ctx, "HTTP request started",
			logger.F("method", c.Request.Method),
			logger.F("path", c.Request.URL.Path),
			logger.F("remote_addr", c.ClientIP()),
		)

		// 处理请求
		c.Next()

		// 记录请求结束
		duration := time.Since(start)
		status := c.Writer.Status()

		span.SetAttributes(
			attribute.Int("http.status_code", status),
			attribute.Int64("http.response_size", int64(c.Writer.Size())),
			attribute.Float64("http.duration_ms", float64(duration.Nanoseconds())/1e6),
		)

		// 设置span状态
		if status >= 400 {
			span.SetStatus(codes.Error, "HTTP error")
		} else {
			span.SetStatus(codes.Ok, "HTTP success")
		}

		// 记录响应日志
		logLevel := "info"
		if status >= 500 {
			logLevel = "error"
		} else if status >= 400 {
			logLevel = "warn"
		}

		// 记录响应日志
		if logLevel == "error" {
			m.logger.Error(ctx, "HTTP request completed",
				logger.F("method", c.Request.Method),
				logger.F("path", c.Request.URL.Path),
				logger.F("status", status),
				logger.F("duration_ms", float64(duration.Nanoseconds())/1e6),
				logger.F("remote_addr", c.ClientIP()),
			)
		} else if logLevel == "warn" {
			m.logger.Warn(ctx, "HTTP request completed",
				logger.F("method", c.Request.Method),
				logger.F("path", c.Request.URL.Path),
				logger.F("status", status),
				logger.F("duration_ms", float64(duration.Nanoseconds())/1e6),
				logger.F("remote_addr", c.ClientIP()),
			)
		} else {
			m.logger.Info(ctx, "HTTP request completed",
				logger.F("method", c.Request.Method),
				logger.F("path", c.Request.URL.Path),
				logger.F("status", status),
				logger.F("duration_ms", float64(duration.Nanoseconds())/1e6),
				logger.F("remote_addr", c.ClientIP()),
			)
		}
	}
}

// GRPCUnaryInterceptor gRPC一元拦截器
func (m *ObservabilityMiddleware) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 生成请求ID
		requestID := uuid.New().String()

		// 创建span
		ctx, span := m.tracer.Start(ctx, info.FullMethod)
		defer span.End()

		// 设置span属性
		span.SetAttributes(
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.service", info.FullMethod),
			attribute.String("request.id", requestID),
		)

		// 在context中设置请求ID（类型安全）
		ctx = WithRequestID(ctx, requestID)

		// 从metadata中获取用户信息
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if userIDs := md.Get("user-id"); len(userIDs) > 0 {
				if userID, err := strconv.ParseInt(userIDs[0], 10, 64); err == nil {
					ctx = WithUserID(ctx, userID)
					span.SetAttributes(attribute.Int64("user.id", userID))
				}
			}

			// 获取客户端IP（如果有的话）
			if clientIPs := md.Get("client-ip"); len(clientIPs) > 0 {
				ctx = WithClientIP(ctx, clientIPs[0])
			}
		}

		// 记录请求开始
		start := time.Now()
		m.logger.Info(ctx, "gRPC request started",
			logger.F("method", info.FullMethod),
		)

		// 处理请求
		resp, err := handler(ctx, req)

		// 记录请求结束
		duration := time.Since(start)

		span.SetAttributes(
			attribute.Float64("rpc.duration_ms", float64(duration.Nanoseconds())/1e6),
		)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			m.logger.Error(ctx, "gRPC request failed",
				logger.F("method", info.FullMethod),
				logger.F("error", err.Error()),
				logger.F("duration_ms", float64(duration.Nanoseconds())/1e6),
			)
		} else {
			span.SetStatus(codes.Ok, "gRPC success")
			m.logger.Info(ctx, "gRPC request completed",
				logger.F("method", info.FullMethod),
				logger.F("duration_ms", float64(duration.Nanoseconds())/1e6),
			)
		}

		return resp, err
	}
}

// GRPCStreamInterceptor gRPC流拦截器
func (m *ObservabilityMiddleware) GRPCStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 生成请求ID
		requestID := uuid.New().String()

		// 创建span
		ctx, span := m.tracer.Start(ss.Context(), info.FullMethod)
		defer span.End()

		// 设置span属性
		span.SetAttributes(
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.service", info.FullMethod),
			attribute.String("rpc.method_type", "stream"),
			attribute.String("request.id", requestID),
		)

		// 在context中设置请求ID
		ctx = WithRequestID(ctx, requestID)

		// 包装ServerStream以传递新的context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// 记录流开始
		start := time.Now()
		m.logger.Info(ctx, "gRPC stream started",
			logger.F("method", info.FullMethod),
		)

		// 处理流
		err := handler(srv, wrappedStream)

		// 记录流结束
		duration := time.Since(start)

		span.SetAttributes(
			attribute.Float64("rpc.duration_ms", float64(duration.Nanoseconds())/1e6),
		)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			m.logger.Error(ctx, "gRPC stream failed",
				logger.F("method", info.FullMethod),
				logger.F("error", err.Error()),
				logger.F("duration_ms", float64(duration.Nanoseconds())/1e6),
			)
		} else {
			span.SetStatus(codes.Ok, "gRPC stream success")
			m.logger.Info(ctx, "gRPC stream completed",
				logger.F("method", info.FullMethod),
				logger.F("duration_ms", float64(duration.Nanoseconds())/1e6),
			)
		}

		return err
	}
}

// wrappedServerStream 包装ServerStream以传递context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
