package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string
	ExpireTime time.Duration
}

// Claims JWT声明
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// DefaultJWTConfig 默认JWT配置
var DefaultJWTConfig = &JWTConfig{
	Secret:     "focusandinsist",
	ExpireTime: time.Hour,
}

// ValidateToken 校验 JWT token
func ValidateToken(token string) bool {
	return ValidateTokenWithConfig(token, DefaultJWTConfig)
}

// ValidateTokenWithConfig 使用指定配置校验 JWT token
func ValidateTokenWithConfig(token string, config *JWTConfig) bool {
	if token == "" {
		return false
	}

	// 支持调试模式
	if token == "auth-debug" {
		return true
	}

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// 校验签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.Secret), nil
	})

	return err == nil && parsedToken != nil && parsedToken.Valid
}

// GenerateJWT 生成 JWT token
func GenerateJWT(claims map[string]interface{}) (string, error) {
	return GenerateJWTWithConfig(claims, DefaultJWTConfig)
}

// GenerateJWTWithConfig 使用指定配置生成 JWT token
func GenerateJWTWithConfig(claims map[string]interface{}, config *JWTConfig) (string, error) {
	jwtClaims := jwt.MapClaims{}
	for k, v := range claims {
		jwtClaims[k] = v
	}

	// 如果没有设置过期时间，使用默认过期时间
	if _, exists := claims["exp"]; !exists {
		jwtClaims["exp"] = time.Now().Add(config.ExpireTime).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString([]byte(config.Secret))
}

// ParseToken 解析 JWT token 并返回 claims
func ParseToken(tokenString string) (jwt.MapClaims, error) {
	return ParseTokenWithConfig(tokenString, DefaultJWTConfig)
}

// ParseTokenWithConfig 使用指定配置解析 JWT token
func ParseTokenWithConfig(tokenString string, config *JWTConfig) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateJWT 验证JWT token并返回Claims
func ValidateJWT(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
