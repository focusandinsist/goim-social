package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	kratoslog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"websocket-server/pkg/auth"
)

// AuthMiddleware 认证中间件配置
type AuthMiddleware struct {
	logger kratoslog.Logger
	jwtKey string
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(logger kratoslog.Logger, jwtKey string) *AuthMiddleware {
	return &AuthMiddleware{
		logger: logger,
		jwtKey: jwtKey,
	}
}

// GinAuth Gin认证中间件
func (am *AuthMiddleware) GinAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过健康检查和公开接口
		if am.shouldSkipAuth(c.Request.URL.Path) {
			c.Next()
			return
		}

		token := am.extractTokenFromHeader(c.GetHeader("Authorization"))
		if token == "" {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Missing authorization token", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			c.Abort()
			return
		}

		// 验证JWT token
		claims, err := auth.ValidateJWT(token, am.jwtKey)
		if err != nil {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Invalid token", "error", err, "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)

		am.logger.Log(kratoslog.LevelDebug, "msg", "User authenticated", "userID", claims.UserID, "path", c.Request.URL.Path)
		c.Next()
	}
}

// GRPCAuth gRPC认证拦截器
func (am *AuthMiddleware) GRPCAuth() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 跳过健康检查和公开接口
		if am.shouldSkipGRPCAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Missing metadata", "method", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "Missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Missing authorization token", "method", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "Missing authorization token")
		}

		token := am.extractTokenFromHeader(tokens[0])
		if token == "" {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Invalid authorization header", "method", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "Invalid authorization header")
		}

		// 验证JWT token
		claims, err := auth.ValidateJWT(token, am.jwtKey)
		if err != nil {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Invalid token", "error", err, "method", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "Invalid token")
		}

		// 将用户信息添加到上下文
		ctx = context.WithValue(ctx, "userID", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)

		am.logger.Log(kratoslog.LevelDebug, "msg", "User authenticated", "userID", claims.UserID, "method", info.FullMethod)
		return handler(ctx, req)
	}
}

// GRPCStreamAuth gRPC流认证拦截器
func (am *AuthMiddleware) GRPCStreamAuth() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 跳过健康检查和公开接口
		if am.shouldSkipGRPCAuth(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Missing metadata", "method", info.FullMethod)
			return status.Errorf(codes.Unauthenticated, "Missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Missing authorization token", "method", info.FullMethod)
			return status.Errorf(codes.Unauthenticated, "Missing authorization token")
		}

		token := am.extractTokenFromHeader(tokens[0])
		if token == "" {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Invalid authorization header", "method", info.FullMethod)
			return status.Errorf(codes.Unauthenticated, "Invalid authorization header")
		}

		// 验证JWT token
		claims, err := auth.ValidateJWT(token, am.jwtKey)
		if err != nil {
			am.logger.Log(kratoslog.LevelWarn, "msg", "Invalid token", "error", err, "method", info.FullMethod)
			return status.Errorf(codes.Unauthenticated, "Invalid token")
		}

		// 创建包装的流，包含用户信息
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ctx, "userID", claims.UserID),
		}

		am.logger.Log(kratoslog.LevelDebug, "msg", "User authenticated", "userID", claims.UserID, "method", info.FullMethod)
		return handler(srv, wrappedStream)
	}
}

// extractTokenFromHeader 从Authorization头中提取token
func (am *AuthMiddleware) extractTokenFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	// 支持 "Bearer token" 和直接的 "token" 格式
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}

// shouldSkipAuth 判断是否跳过认证
func (am *AuthMiddleware) shouldSkipAuth(path string) bool {
	skipPaths := []string{
		"/health",
		"/metrics",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/connect/ws", // WebSocket连接有自己的认证逻辑
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

// shouldSkipGRPCAuth 判断是否跳过gRPC认证
func (am *AuthMiddleware) shouldSkipGRPCAuth(method string) bool {
	skipMethods := []string{
		"/grpc.health.v1.Health/Check",
		"/UserService/Login",
		"/UserService/Register",
	}

	for _, skipMethod := range skipMethods {
		if strings.Contains(method, skipMethod) {
			return true
		}
	}

	return false
}

// wrappedServerStream 包装的服务器流
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回包装的上下文
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
