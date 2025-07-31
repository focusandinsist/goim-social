package consistent

import "fmt"

// GatewayMember 网关成员实现
type GatewayMember struct {
	ID   string // 网关实例ID
	Host string // 主机地址
	Port int    // 端口号
}

// Clone 创建成员的深拷贝，确保并发安全
func (g *GatewayMember) Clone() *GatewayMember {
	return &GatewayMember{
		ID:   g.ID,
		Host: g.Host,
		Port: g.Port,
	}
}

// NewGatewayMember 创建新的网关成员
func NewGatewayMember(id, host string, port int) *GatewayMember {
	return &GatewayMember{
		ID:   id,
		Host: host,
		Port: port,
	}
}

// String 返回成员的字符串表示
func (g *GatewayMember) String() string {
	return fmt.Sprintf("%s:%s:%d", g.ID, g.Host, g.Port)
}

// GetID 获取网关ID
func (g *GatewayMember) GetID() string {
	return g.ID
}

// GetHost 获取主机地址
func (g *GatewayMember) GetHost() string {
	return g.Host
}

// GetPort 获取端口号
func (g *GatewayMember) GetPort() int {
	return g.Port
}

// GetAddress 获取完整地址
func (g *GatewayMember) GetAddress() string {
	return fmt.Sprintf("%s:%d", g.Host, g.Port)
}
