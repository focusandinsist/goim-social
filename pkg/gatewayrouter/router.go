package gatewayrouter

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"websocket-server/pkg/consistent"
	redisClient "websocket-server/pkg/redis"

	"github.com/go-redis/redis/v8"
)

const (
	// ActiveGatewaysKey Redis ZSET键名，存储活跃网关实例
	ActiveGatewaysKey = "active_gateways"
	// HeartbeatWindow 心跳窗口时间（秒），超过此时间认为实例不活跃
	HeartbeatWindow = 90
	// CleanupInterval 清理过期实例的间隔时间
	CleanupInterval = 60 * time.Second
)

// GatewayInstance 网关实例信息
type GatewayInstance struct {
	ID            string `json:"id"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	LastHeartbeat int64  `json:"last_heartbeat"`
}

// String 实现 consistent.Member 接口
func (g *GatewayInstance) String() string {
	return g.ID
}

// GetAddress 获取网关地址
func (g *GatewayInstance) GetAddress() string {
	return fmt.Sprintf("%s:%d", g.Host, g.Port)
}

// Router 网关路由管理器
type Router struct {
	redis         *redisClient.RedisClient
	ring          *consistent.Consistent
	instances     map[string]*GatewayInstance // 实例ID -> 实例信息
	mu            sync.RWMutex
	stopCh        chan struct{}
	cleanupTicker *time.Ticker
}

// NewRouter 创建网关路由管理器
func NewRouter(redis *redisClient.RedisClient) *Router {
	// 配置一致性哈希环
	config := consistent.Config{
		Hasher:            consistent.NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	router := &Router{
		redis:     redis,
		ring:      consistent.New(nil, config),
		instances: make(map[string]*GatewayInstance),
		stopCh:    make(chan struct{}),
	}

	// 启动时同步Redis中的活跃实例
	if err := router.syncActiveGateways(); err != nil {
		log.Printf("初始化同步活跃网关失败: %v", err)
	}

	// 启动后台监控和清理任务
	router.startBackgroundTasks()

	return router
}

// RegisterGateway 注册网关实例（网关启动时调用）
func (r *Router) RegisterGateway(ctx context.Context, instanceID, host string, port int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 创建实例信息
	instance := &GatewayInstance{
		ID:            instanceID,
		Host:          host,
		Port:          port,
		LastHeartbeat: time.Now().Unix(),
	}

	// 添加到Redis ZSET
	score := float64(time.Now().Unix())
	z := &redis.Z{Score: score, Member: instanceID}
	if err := r.redis.ZAdd(ctx, ActiveGatewaysKey, z); err != nil {
		return fmt.Errorf("注册网关到Redis失败: %v", err)
	}

	// 添加到本地缓存和一致性哈希环
	r.instances[instanceID] = instance
	r.ring.Add(instance)

	log.Printf("网关实例已注册: %s (%s)", instanceID, instance.GetAddress())
	return nil
}

// UnregisterGateway 注销网关实例（网关关闭时调用）
func (r *Router) UnregisterGateway(ctx context.Context, instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 从Redis ZSET移除
	if err := r.redis.ZRem(ctx, ActiveGatewaysKey, instanceID); err != nil {
		log.Printf("从Redis移除网关失败: %v", err)
	}

	// 从本地缓存和一致性哈希环移除
	delete(r.instances, instanceID)
	r.ring.Remove(instanceID)

	log.Printf("网关实例已注销: %s", instanceID)
	return nil
}

// Heartbeat 网关心跳上报（网关定期调用）
func (r *Router) Heartbeat(ctx context.Context, instanceID string) error {
	// 更新Redis ZSET中的分数（时间戳）
	score := float64(time.Now().Unix())
	z := &redis.Z{Score: score, Member: instanceID}
	if err := r.redis.ZAdd(ctx, ActiveGatewaysKey, z); err != nil {
		return fmt.Errorf("更新心跳失败: %v", err)
	}

	// 更新本地缓存
	r.mu.Lock()
	if instance, exists := r.instances[instanceID]; exists {
		instance.LastHeartbeat = time.Now().Unix()
	}
	r.mu.Unlock()

	return nil
}

// GetGatewayForUser 根据用户ID获取对应的网关实例
func (r *Router) GetGatewayForUser(userID string) (*GatewayInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("user:%s", userID)
	member := r.ring.LocateKey([]byte(key))
	if member == nil {
		return nil, fmt.Errorf("没有可用的网关实例")
	}

	if instance, ok := member.(*GatewayInstance); ok {
		return instance, nil
	}

	return nil, fmt.Errorf("网关实例类型错误")
}

// GetGatewayForRoom 根据房间ID获取对应的网关实例
func (r *Router) GetGatewayForRoom(roomID string) (*GatewayInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("room:%s", roomID)
	member := r.ring.LocateKey([]byte(key))
	if member == nil {
		return nil, fmt.Errorf("没有可用的网关实例")
	}

	if instance, ok := member.(*GatewayInstance); ok {
		return instance, nil
	}

	return nil, fmt.Errorf("网关实例类型错误")
}

// GetAllActiveGateways 获取所有活跃的网关实例
func (r *Router) GetAllActiveGateways() []*GatewayInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var instances []*GatewayInstance
	for _, instance := range r.instances {
		instances = append(instances, instance)
	}
	return instances
}

// GetStats 获取路由统计信息
func (r *Router) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	loadDist := r.ring.LoadDistribution()
	return map[string]interface{}{
		"active_gateways":   len(r.instances),
		"average_load":      r.ring.AverageLoad(),
		"load_distribution": loadDist,
	}
}

// syncActiveGateways 从Redis同步活跃网关实例
func (r *Router) syncActiveGateways() error {
	ctx := context.Background()

	// 获取当前时间窗口内的活跃实例
	minScore := strconv.FormatInt(time.Now().Unix()-HeartbeatWindow, 10)
	maxScore := "+inf"

	opt := &redis.ZRangeBy{
		Min: minScore,
		Max: maxScore,
	}

	activeIDs, err := r.redis.ZRangeByScore(ctx, ActiveGatewaysKey, opt)
	if err != nil {
		return fmt.Errorf("获取活跃网关列表失败: %v", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 清空当前实例
	r.instances = make(map[string]*GatewayInstance)
	r.ring = consistent.New(nil, consistent.Config{
		Hasher:            consistent.NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	})

	// 重新构建实例列表和哈希环
	for _, instanceID := range activeIDs {
		// TODO: 从hash节点信息里获取host和port信息
		instance := &GatewayInstance{
			ID:            instanceID,
			Host:          "localhost",
			Port:          8080,
			LastHeartbeat: time.Now().Unix(),
		}

		r.instances[instanceID] = instance
		r.ring.Add(instance)
	}

	log.Printf("同步完成，当前活跃网关数量: %d", len(r.instances))
	return nil
}

// startBackgroundTasks 启动后台监控和清理任务
func (r *Router) startBackgroundTasks() {
	// 启动定期同步任务
	go r.periodicSync()

	// 启动清理任务
	// TODO: 选举leader清理,或者扔给k8s
	r.cleanupTicker = time.NewTicker(CleanupInterval)
	go r.cleanupExpiredGateways()
}

// periodicSync 定期同步Redis中的活跃实例
func (r *Router) periodicSync() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.syncActiveGateways(); err != nil {
				log.Printf("定期同步活跃网关失败: %v", err)
			}
		case <-r.stopCh:
			return
		}
	}
}

// cleanupExpiredGateways 清理过期的网关实例
func (r *Router) cleanupExpiredGateways() {
	defer r.cleanupTicker.Stop()

	for {
		select {
		case <-r.cleanupTicker.C:
			ctx := context.Background()

			// 移除90秒之前的过期实例
			maxScore := strconv.FormatInt(time.Now().Unix()-HeartbeatWindow, 10)
			if err := r.redis.ZRemRangeByScore(ctx, ActiveGatewaysKey, "0", maxScore); err != nil {
				log.Printf("清理过期网关实例失败: %v", err)
			} else {
				log.Printf("已清理过期网关实例")
			}
		case <-r.stopCh:
			return
		}
	}
}

// Stop 停止路由管理器
func (r *Router) Stop() {
	close(r.stopCh)
	if r.cleanupTicker != nil {
		r.cleanupTicker.Stop()
	}
}
