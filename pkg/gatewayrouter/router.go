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

// Router 网关路由器
// 负责监控Redis ZSET变化并同步到本地一致性哈希环，提供路由决策
type Router struct {
	redis        *redisClient.RedisClient
	ring         *consistent.Consistent
	instances    map[string]*GatewayInstance // 实例ID -> 实例信息
	mu           sync.RWMutex
	stopCh       chan struct{}
	syncTicker   *time.Ticker
	lastSyncTime int64 // 上次同步时间戳，用于检测变化
}

// NewRouter 创建网关路由器
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

	// 启动后台监控任务
	router.startMonitoring()

	return router
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
		"last_sync_time":    r.lastSyncTime,
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

	// 检测变化
	currentInstanceIDs := make(map[string]bool)
	for _, id := range activeIDs {
		currentInstanceIDs[id] = true
	}

	// 检查是否有变化
	hasChanges := len(currentInstanceIDs) != len(r.instances)
	if !hasChanges {
		for id := range r.instances {
			if !currentInstanceIDs[id] {
				hasChanges = true
				break
			}
		}
	}

	// 如果没有变化，直接返回
	if !hasChanges {
		return nil
	}

	log.Printf("检测到网关实例变化，开始同步...")

	// 记录变化
	var addedInstances, removedInstances []string

	// 找出新增的实例
	for id := range currentInstanceIDs {
		if _, exists := r.instances[id]; !exists {
			addedInstances = append(addedInstances, id)
		}
	}

	// 找出移除的实例
	for id := range r.instances {
		if !currentInstanceIDs[id] {
			removedInstances = append(removedInstances, id)
		}
	}

	// 优化：在无锁状态下获取所有新实例的详细信息
	newInstances := make(map[string]*GatewayInstance)
	for _, instanceID := range addedInstances {
		instance, err := r.getInstanceDetails(ctx, instanceID)
		if err != nil {
			log.Printf("获取实例 %s 详细信息失败: %v", instanceID, err)
			// 跳过获取失败的实例，而不是使用默认值
			continue
		}
		newInstances[instanceID] = instance
	}

	// 现在获取写锁，进行纯内存操作
	r.mu.Lock()
	defer r.mu.Unlock()

	// 移除不再活跃的实例
	for _, instanceID := range removedInstances {
		delete(r.instances, instanceID)
		r.ring.Remove(instanceID)
		log.Printf("移除网关实例: %s", instanceID)
	}

	// 添加新的活跃实例
	for instanceID, instance := range newInstances {
		// 防止在获取详情期间，节点又下线了，做最终检查
		if currentInstanceIDs[instanceID] {
			r.instances[instanceID] = instance
			r.ring.Add(instance)
			log.Printf("添加网关实例: %s (%s)", instanceID, instance.GetAddress())
		}
	}

	r.lastSyncTime = time.Now().Unix()
	log.Printf("网关实例同步完成，当前活跃数量: %d (新增: %d, 移除: %d)",
		len(r.instances), len(addedInstances), len(removedInstances))

	return nil
}

// getInstanceDetails 从Redis Hash获取实例详细信息
func (r *Router) getInstanceDetails(ctx context.Context, instanceID string) (*GatewayInstance, error) {
	key := fmt.Sprintf("connect_instances:%s", instanceID)
	fields, err := r.redis.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("实例信息不存在")
	}

	// 解析端口号
	port := 8080 // 默认值
	if portStr, exists := fields["port"]; exists {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// 解析最后心跳时间
	lastHeartbeat := time.Now().Unix()
	if hbStr, exists := fields["last_ping"]; exists {
		if hb, err := strconv.ParseInt(hbStr, 10, 64); err == nil {
			lastHeartbeat = hb
		}
	}

	return &GatewayInstance{
		ID:            instanceID,
		Host:          fields["host"],
		Port:          port,
		LastHeartbeat: lastHeartbeat,
	}, nil
}

// startMonitoring 启动后台监控任务
func (r *Router) startMonitoring() {
	// 启动定期同步任务（频率较高，用于快速检测变化）
	r.syncTicker = time.NewTicker(10 * time.Second)
	go r.periodicSync()
}

// periodicSync 定期同步Redis中的活跃实例
func (r *Router) periodicSync() {
	defer r.syncTicker.Stop()

	for {
		select {
		case <-r.syncTicker.C:
			if err := r.syncActiveGateways(); err != nil {
				log.Printf("定期同步活跃网关失败: %v", err)
			}
		case <-r.stopCh:
			return
		}
	}
}

// Stop 停止路由客户端
func (r *Router) Stop() {
	close(r.stopCh)
	if r.syncTicker != nil {
		r.syncTicker.Stop()
	}
}

// ForceSync 强制同步（用于手动触发同步）
func (r *Router) ForceSync() error {
	return r.syncActiveGateways()
}
