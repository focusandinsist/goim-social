package middleware

import (
	"strings"

	"websocket-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

// APIGatewayMiddleware API网关中间件配置
type APIGatewayMiddleware struct {
	logger    logger.Logger
	jwtKey    string
	skipPaths []string // 跳过认证的路径
}

// NewAPIGatewayMiddleware 创建API网关中间件
func NewAPIGatewayMiddleware(logger logger.Logger, jwtKey string) *APIGatewayMiddleware {
	return &APIGatewayMiddleware{
		logger: logger,
		jwtKey: jwtKey,
		skipPaths: []string{
			// API网关管理接口
			"/api/v1/api-gateway/health",
			"/api/v1/api-gateway/services",
			"/api/v1/api-gateway/online_status",
			// 公开接口
			"/health",
			"/ping",
			"/metrics",
			// 动态路由的健康检查
			"/api/v1/*/health",
		},
	}
}

// NewAPIGatewayMiddlewareWithSkipPaths 创建API网关中间件并指定跳过路径
func NewAPIGatewayMiddlewareWithSkipPaths(logger logger.Logger, jwtKey string, skipPaths []string) *APIGatewayMiddleware {
	mw := NewAPIGatewayMiddleware(logger, jwtKey)
	mw.skipPaths = append(mw.skipPaths, skipPaths...)
	return mw
}

// AddSkipPath 添加跳过认证的路径
func (m *APIGatewayMiddleware) AddSkipPath(path string) {
	m.skipPaths = append(m.skipPaths, path)
}

// AddSkipPaths 批量添加跳过认证的路径
func (m *APIGatewayMiddleware) AddSkipPaths(paths []string) {
	m.skipPaths = append(m.skipPaths, paths...)
}

// shouldSkipAuth 检查是否应该跳过认证
func (m *APIGatewayMiddleware) shouldSkipAuth(path string) bool {
	for _, skipPath := range m.skipPaths {
		if path == skipPath {
			return true
		}
		// 支持前缀匹配 /api/v1/*
		if strings.HasSuffix(skipPath, "/*") {
			prefix := strings.TrimSuffix(skipPath, "/*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		// 支持通配符匹配 /api/v1/*/health
		if strings.Contains(skipPath, "*") {
			if matchWildcard(skipPath, path) {
				return true
			}
		}
	}
	return false
}

// matchWildcard 简单的通配符匹配
func matchWildcard(pattern, str string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == str
	}

	// 检查开头
	if !strings.HasPrefix(str, parts[0]) {
		return false
	}
	str = str[len(parts[0]):]

	// 检查中间部分
	for i := 1; i < len(parts)-1; i++ {
		part := parts[i]
		if part == "" {
			continue
		}
		idx := strings.Index(str, part)
		if idx == -1 {
			return false
		}
		str = str[idx+len(part):]
	}

	// 检查结尾
	lastPart := parts[len(parts)-1]
	if lastPart == "" {
		return true
	}
	return strings.HasSuffix(str, lastPart)
}

// GinAuth 返回Gin认证中间件
func (m *APIGatewayMiddleware) GinAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否跳过认证
		if m.shouldSkipAuth(c.Request.URL.Path) {
			c.Next()
			return
		}

		// TODO:Token验证
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.logger.Warn(c.Request.Context(), "Missing authorization header",
				logger.F("path", c.Request.URL.Path),
				logger.F("method", c.Request.Method))
			c.JSON(401, map[string]interface{}{
				"error": "Missing authorization header",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GinLogging 返回Gin日志中间件
func (m *APIGatewayMiddleware) GinLogging() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		m.logger.Info(param.Request.Context(), "API Gateway Request",
			logger.F("method", param.Method),
			logger.F("path", param.Path),
			logger.F("status", param.StatusCode),
			logger.F("latency", param.Latency),
			logger.F("client_ip", param.ClientIP),
			logger.F("user_agent", param.Request.UserAgent()))
		return ""
	})
}

// GinRecovery 返回Gin恢复中间件
func (m *APIGatewayMiddleware) GinRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		m.logger.Error(c.Request.Context(), "API Gateway Panic Recovered",
			logger.F("error", recovered),
			logger.F("path", c.Request.URL.Path),
			logger.F("method", c.Request.Method))
		c.JSON(500, map[string]interface{}{
			"error": "Internal server error",
		})
	})
}
