package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	logger kratoslog.Logger
}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware(logger kratoslog.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// GinLogging Gin日志中间件
func (lm *LoggingMiddleware) GinLogging() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 使用Kratos日志器记录请求
		lm.logger.Log(kratoslog.LevelInfo,
			"msg", "HTTP request",
			"method", param.Method,
			"path", param.Path,
			"status", param.StatusCode,
			"latency", param.Latency.String(),
			"client_ip", param.ClientIP,
			"user_agent", param.Request.UserAgent(),
			"error", param.ErrorMessage,
		)
		return ""
	})
}

// GRPCLogging gRPC日志拦截器
func (lm *LoggingMiddleware) GRPCLogging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// 记录请求开始
		lm.logger.Log(kratoslog.LevelDebug,
			"msg", "gRPC request started",
			"method", info.FullMethod,
		)

		// 执行处理器
		resp, err := handler(ctx, req)

		// 计算耗时
		duration := time.Since(start)

		// 获取状态码
		st := status.Convert(err)

		// 记录请求完成
		if err != nil {
			lm.logger.Log(kratoslog.LevelError,
				"msg", "gRPC request completed with error",
				"method", info.FullMethod,
				"duration", duration.String(),
				"code", st.Code().String(),
				"error", err.Error(),
			)
		} else {
			lm.logger.Log(kratoslog.LevelInfo,
				"msg", "gRPC request completed",
				"method", info.FullMethod,
				"duration", duration.String(),
				"code", st.Code().String(),
			)
		}

		return resp, err
	}
}

// GRPCStreamLogging gRPC流日志拦截器
func (lm *LoggingMiddleware) GRPCStreamLogging() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// 记录流开始
		lm.logger.Log(kratoslog.LevelDebug,
			"msg", "gRPC stream started",
			"method", info.FullMethod,
		)

		// 执行处理器
		err := handler(srv, ss)

		// 计算耗时
		duration := time.Since(start)

		// 获取状态码
		st := status.Convert(err)

		// 记录流完成
		if err != nil {
			lm.logger.Log(kratoslog.LevelError,
				"msg", "gRPC stream completed with error",
				"method", info.FullMethod,
				"duration", duration.String(),
				"code", st.Code().String(),
				"error", err.Error(),
			)
		} else {
			lm.logger.Log(kratoslog.LevelInfo,
				"msg", "gRPC stream completed",
				"method", info.FullMethod,
				"duration", duration.String(),
				"code", st.Code().String(),
			)
		}

		return err
	}
}

// GinRecovery Gin恢复中间件
func (lm *LoggingMiddleware) GinRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		lm.logger.Log(kratoslog.LevelError,
			"msg", "HTTP request panic recovered",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"panic", recovered,
		)
		c.AbortWithStatus(500)
	})
}

// GRPCRecovery gRPC恢复拦截器
func (lm *LoggingMiddleware) GRPCRecovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				lm.logger.Log(kratoslog.LevelError,
					"msg", "gRPC request panic recovered",
					"method", info.FullMethod,
					"panic", r,
				)
				err = status.Errorf(status.Code(err), "Internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// GRPCStreamRecovery gRPC流恢复拦截器
func (lm *LoggingMiddleware) GRPCStreamRecovery() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				lm.logger.Log(kratoslog.LevelError,
					"msg", "gRPC stream panic recovered",
					"method", info.FullMethod,
					"panic", r,
				)
				err = status.Errorf(status.Code(err), "Internal server error")
			}
		}()

		return handler(srv, ss)
	}
}
