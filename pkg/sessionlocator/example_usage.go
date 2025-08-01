package sessionlocator

import (
	"context"
	"fmt"
	"log"
	"time"

	"goim-social/pkg/redis"
)

// LogicService 示例Logic服务，展示如何使用会话定位器
type LogicService struct {
	redis          *redis.RedisClient
	sessionLocator *Locator
}

// NewLogicService 创建Logic服务实例
func NewLogicService(redis *redis.RedisClient) *LogicService {
	return &LogicService{
		redis:          redis,
		sessionLocator: NewLocator(redis),
	}
}

// SendMessageToUser 向用户发送消息的示例
func (ls *LogicService) SendMessageToUser(ctx context.Context, userID string, message string) error {
	// 1. 使用一致性哈希找到用户对应的网关实例
	gateway, err := ls.sessionLocator.GetGatewayForUser(userID)
	if err != nil {
		return fmt.Errorf("获取用户网关失败: %v", err)
	}

	log.Printf("用户 %s 路由到网关: %s (%s)", userID, gateway.ID, gateway.GetAddress())

	// 2. 向该网关发送消息（这里只是示例，实际需要gRPC调用）
	return ls.forwardMessageToGateway(ctx, gateway, userID, message)
}

// SendMessageToRoom 向房间发送消息的示例
func (ls *LogicService) SendMessageToRoom(ctx context.Context, roomID string, message string) error {
	// 1. 使用一致性哈希找到房间对应的网关实例
	gateway, err := ls.sessionLocator.GetGatewayForRoom(roomID)
	if err != nil {
		return fmt.Errorf("获取房间网关失败: %v", err)
	}

	log.Printf("房间 %s 路由到网关: %s (%s)", roomID, gateway.ID, gateway.GetAddress())

	// 2. 向该网关发送消息（这里只是示例，实际需要gRPC调用）
	return ls.forwardRoomMessageToGateway(ctx, gateway, roomID, message)
}

// GetUserGateway 获取用户所在的网关实例
func (ls *LogicService) GetUserGateway(userID string) (*GatewayInstance, error) {
	return ls.sessionLocator.GetGatewayForUser(userID)
}

// GetRoomGateway 获取房间所在的网关实例
func (ls *LogicService) GetRoomGateway(roomID string) (*GatewayInstance, error) {
	return ls.sessionLocator.GetGatewayForRoom(roomID)
}

// GetGatewayStats 获取网关路由统计信息
func (ls *LogicService) GetGatewayStats() map[string]interface{} {
	return ls.sessionLocator.GetStats()
}

// GetAllActiveGateways 获取所有活跃网关
func (ls *LogicService) GetAllActiveGateways() []*GatewayInstance {
	return ls.sessionLocator.GetAllActiveGateways()
}

// forwardMessageToGateway 向网关转发用户消息（示例实现）
func (ls *LogicService) forwardMessageToGateway(ctx context.Context, gateway *GatewayInstance, userID, message string) error {
	// 这里应该是实际的gRPC调用或HTTP请求
	log.Printf("转发消息到网关 %s: 用户=%s, 消息=%s", gateway.ID, userID, message)

	// 示例：可以通过Redis发布消息到特定网关
	channel := fmt.Sprintf("gateway:%s:user_message", gateway.ID)
	payload := fmt.Sprintf(`{"user_id":"%s","message":"%s"}`, userID, message)

	return ls.redis.Publish(ctx, channel, payload)
}

// forwardRoomMessageToGateway 向网关转发房间消息（示例实现）
func (ls *LogicService) forwardRoomMessageToGateway(ctx context.Context, gateway *GatewayInstance, roomID, message string) error {
	// 这里应该是实际的gRPC调用或HTTP请求
	log.Printf("转发房间消息到网关 %s: 房间=%s, 消息=%s", gateway.ID, roomID, message)

	// 示例：可以通过Redis发布消息到特定网关
	channel := fmt.Sprintf("gateway:%s:room_message", gateway.ID)
	payload := fmt.Sprintf(`{"room_id":"%s","message":"%s"}`, roomID, message)

	return ls.redis.Publish(ctx, channel, payload)
}

// MonitorGatewayChanges 监控网关变化的示例
func (ls *LogicService) MonitorGatewayChanges() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	lastStats := ls.GetGatewayStats()
	log.Printf("初始网关状态: %+v", lastStats)

	for range ticker.C {
		currentStats := ls.GetGatewayStats()

		// 检查网关数量是否发生变化
		if currentStats["active_gateways"] != lastStats["active_gateways"] {
			log.Printf("网关数量发生变化: %d -> %d",
				lastStats["active_gateways"], currentStats["active_gateways"])

			// 打印当前所有活跃网关
			gateways := ls.GetAllActiveGateways()
			log.Printf("当前活跃网关:")
			for _, gw := range gateways {
				log.Printf("  - %s (%s)", gw.ID, gw.GetAddress())
			}
		}

		lastStats = currentStats
	}
}

// Stop 停止Logic服务
func (ls *LogicService) Stop() {
	if ls.sessionLocator != nil {
		ls.sessionLocator.Stop()
	}
}

// 使用示例函数
func ExampleUsage() {
	// 创建Redis客户端
	redisClient := redis.NewRedisClient("localhost:6379")

	// 创建Logic服务
	logicService := NewLogicService(redisClient)
	defer logicService.Stop()

	ctx := context.Background()

	// 示例1：向用户发送消息
	if err := logicService.SendMessageToUser(ctx, "user123", "Hello, user!"); err != nil {
		log.Printf("发送用户消息失败: %v", err)
	}

	// 示例2：向房间发送消息
	if err := logicService.SendMessageToRoom(ctx, "room456", "Hello, room!"); err != nil {
		log.Printf("发送房间消息失败: %v", err)
	}

	// 示例3：获取用户所在网关
	if gateway, err := logicService.GetUserGateway("user123"); err != nil {
		log.Printf("获取用户网关失败: %v", err)
	} else {
		log.Printf("用户user123在网关: %s (%s)", gateway.ID, gateway.GetAddress())
	}

	// 示例4：获取统计信息
	stats := logicService.GetGatewayStats()
	log.Printf("网关统计信息: %+v", stats)

	// 示例5：启动网关变化监控
	go logicService.MonitorGatewayChanges()

	// 等待一段时间观察效果
	time.Sleep(60 * time.Second)
}

// TestConsistentRouting 测试一致性路由
func TestConsistentRouting(redisClient *redis.RedisClient) {
	locator := NewLocator(redisClient)
	defer locator.Stop()

	// 测试多个用户的路由一致性
	testUsers := []string{"user1", "user2", "user3", "user4", "user5"}

	log.Println("测试用户路由一致性:")
	for i := 0; i < 3; i++ { // 测试3轮
		log.Printf("第 %d 轮测试:", i+1)
		for _, userID := range testUsers {
			if gateway, err := locator.GetGatewayForUser(userID); err != nil {
				log.Printf("  用户 %s: 路由失败 - %v", userID, err)
			} else {
				log.Printf("  用户 %s: 网关 %s", userID, gateway.ID)
			}
		}
		time.Sleep(5 * time.Second)
	}
}
