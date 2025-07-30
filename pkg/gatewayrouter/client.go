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

// Client Logic服务使用的网关路由客户端
// 负责监控Redis ZSET变化并同步到本地一致性哈希环
type Client struct {
	redis        *redisClient.RedisClient
	ring         *consistent.Consistent
	instances    map[string]*GatewayInstance // 实例ID -> 实例信息
	mu           sync.RWMutex
	stopCh       chan struct{}
	syncTicker   *time.Ticker
	lastSyncTime int64 // 上次同步时间戳，用于检测变化
}

// NewClient 创建网关路由客户端
func NewClient(redis *redisClient.RedisClient) *Client {
	// 配置一致性哈希环
	config := consistent.Config{
		Hasher:            consistent.NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	client := &Client{
		redis:     redis,
		ring:      consistent.New(nil, config),
		instances: make(map[string]*GatewayInstance),
		stopCh:    make(chan struct{}),
	}

	// 启动时同步Redis中的活跃实例
	if err := client.syncActiveGateways(); err != nil {
		log.Printf("初始化同步活跃网关失败: %v", err)
	}

	// 启动后台监控任务
	client.startMonitoring()

	return client
}

// GetGatewayForUser 根据用户ID获取对应的网关实例
func (c *Client) GetGatewayForUser(userID string) (*GatewayInstance, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("user:%s", userID)
	member := c.ring.LocateKey([]byte(key))
	if member == nil {
		return nil, fmt.Errorf("没有可用的网关实例")
	}

	if instance, ok := member.(*GatewayInstance); ok {
		return instance, nil
	}

	return nil, fmt.Errorf("网关实例类型错误")
}

// GetGatewayForRoom 根据房间ID获取对应的网关实例
func (c *Client) GetGatewayForRoom(roomID string) (*GatewayInstance, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("room:%s", roomID)
	member := c.ring.LocateKey([]byte(key))
	if member == nil {
		return nil, fmt.Errorf("没有可用的网关实例")
	}

	if instance, ok := member.(*GatewayInstance); ok {
		return instance, nil
	}

	return nil, fmt.Errorf("网关实例类型错误")
}

// GetAllActiveGateways 获取所有活跃的网关实例
func (c *Client) GetAllActiveGateways() []*GatewayInstance {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var instances []*GatewayInstance
	for _, instance := range c.instances {
		instances = append(instances, instance)
	}
	return instances
}

// GetStats 获取路由统计信息
func (c *Client) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	loadDist := c.ring.LoadDistribution()
	return map[string]interface{}{
		"active_gateways":   len(c.instances),
		"average_load":      c.ring.AverageLoad(),
		"load_distribution": loadDist,
		"last_sync_time":    c.lastSyncTime,
	}
}

// syncActiveGateways 从Redis同步活跃网关实例
func (c *Client) syncActiveGateways() error {
	ctx := context.Background()

	// 获取当前时间窗口内的活跃实例
	minScore := strconv.FormatInt(time.Now().Unix()-HeartbeatWindow, 10)
	maxScore := "+inf"

	opt := &redis.ZRangeBy{
		Min: minScore,
		Max: maxScore,
	}

	activeIDs, err := c.redis.ZRangeByScore(ctx, ActiveGatewaysKey, opt)
	if err != nil {
		return fmt.Errorf("获取活跃网关列表失败: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检测变化
	currentInstanceIDs := make(map[string]bool)
	for _, id := range activeIDs {
		currentInstanceIDs[id] = true
	}

	// 检查是否有变化
	hasChanges := len(currentInstanceIDs) != len(c.instances)
	if !hasChanges {
		for id := range c.instances {
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
		if _, exists := c.instances[id]; !exists {
			addedInstances = append(addedInstances, id)
		}
	}

	// 找出移除的实例
	for id := range c.instances {
		if !currentInstanceIDs[id] {
			removedInstances = append(removedInstances, id)
		}
	}

	// 移除不再活跃的实例
	for _, instanceID := range removedInstances {
		delete(c.instances, instanceID)
		c.ring.Remove(instanceID)
		log.Printf("移除网关实例: %s", instanceID)
	}

	// 添加新的活跃实例
	for _, instanceID := range addedInstances {
		// TODO 从Redis Hash获取实例详细信息
		instance, err := c.getInstanceDetails(ctx, instanceID)
		if err != nil {
			log.Printf("获取实例 %s 详细信息失败: %v", instanceID, err)
			instance = &GatewayInstance{
				ID:            instanceID,
				Host:          "localhost",
				Port:          8080,
				LastHeartbeat: time.Now().Unix(),
			}
		}

		c.instances[instanceID] = instance
		c.ring.Add(instance)
		log.Printf("添加网关实例: %s (%s)", instanceID, instance.GetAddress())
	}

	c.lastSyncTime = time.Now().Unix()
	log.Printf("网关实例同步完成，当前活跃数量: %d (新增: %d, 移除: %d)",
		len(c.instances), len(addedInstances), len(removedInstances))

	return nil
}

// getInstanceDetails 从Redis Hash获取实例详细信息
func (c *Client) getInstanceDetails(ctx context.Context, instanceID string) (*GatewayInstance, error) {
	key := fmt.Sprintf("connect_instances:%s", instanceID)
	fields, err := c.redis.HGetAll(ctx, key)
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
func (c *Client) startMonitoring() {
	// 启动定期同步任务（频率较高，用于快速检测变化）
	c.syncTicker = time.NewTicker(10 * time.Second)
	go c.periodicSync()
}

// periodicSync 定期同步Redis中的活跃实例
func (c *Client) periodicSync() {
	defer c.syncTicker.Stop()

	for {
		select {
		case <-c.syncTicker.C:
			if err := c.syncActiveGateways(); err != nil {
				log.Printf("定期同步活跃网关失败: %v", err)
			}
		case <-c.stopCh:
			return
		}
	}
}

// Stop 停止路由客户端
func (c *Client) Stop() {
	close(c.stopCh)
	if c.syncTicker != nil {
		c.syncTicker.Stop()
	}
}

// ForceSync 强制同步（用于手动触发同步）
func (c *Client) ForceSync() error {
	return c.syncActiveGateways()
}
