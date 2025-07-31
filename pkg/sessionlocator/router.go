package sessionlocator

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

// Locator 会话定位器
// 负责监控Redis ZSET变化并同步到本地一致性哈希环，提供路由决策
type Locator struct {
	redis        *redisClient.RedisClient
	ring         *consistent.Consistent
	instances    map[string]*GatewayInstance // 实例ID -> 实例信息
	mu           sync.RWMutex
	stopCh       chan struct{}
	syncTicker   *time.Ticker
	lastSyncTime int64 // 上次同步时间戳，用于检测变化
}

// NewLocator 创建会话定位器
func NewLocator(redis *redisClient.RedisClient) *Locator {
	// 配置一致性哈希环
	config := consistent.Config{
		Hasher:            consistent.NewCRC64Hasher(),
		PartitionCount:    271,
		ReplicationFactor: 20,
		Load:              1.25,
	}

	locator := &Locator{
		redis:     redis,
		ring:      consistent.New(nil, config),
		instances: make(map[string]*GatewayInstance),
		stopCh:    make(chan struct{}),
	}

	// 启动时同步Redis中的活跃实例
	if err := locator.syncActiveGateways(); err != nil {
		log.Printf("初始化同步活跃网关失败: %v", err)
	}

	// 启动后台监控任务
	locator.startMonitoring()

	return locator
}

// GetGatewayForUser 根据用户ID获取对应的网关实例
func (l *Locator) GetGatewayForUser(userID string) (*GatewayInstance, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	key := fmt.Sprintf("user:%s", userID)
	member := l.ring.LocateKey([]byte(key))
	if member == nil {
		return nil, fmt.Errorf("没有可用的网关实例")
	}

	// 直接从本地缓存中获取实例信息，性能极高
	if instance, ok := l.instances[member.String()]; ok {
		return instance, nil
	}

	return nil, fmt.Errorf("在哈希环上找到的实例 %s 不存在于本地缓存中，数据可能不一致", member.String())
}

// GetGatewayForRoom 根据房间ID获取对应的网关实例
func (l *Locator) GetGatewayForRoom(roomID string) (*GatewayInstance, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	key := fmt.Sprintf("room:%s", roomID)
	member := l.ring.LocateKey([]byte(key))
	if member == nil {
		return nil, fmt.Errorf("没有可用的网关实例")
	}

	// 直接从本地缓存中获取实例信息，性能极高
	if instance, ok := l.instances[member.String()]; ok {
		return instance, nil
	}

	return nil, fmt.Errorf("在哈希环上找到的实例 %s 不存在于本地缓存中，数据可能不一致", member.String())
}

// GetAllActiveGateways 获取所有活跃的网关实例
func (l *Locator) GetAllActiveGateways() []*GatewayInstance {
	l.mu.RLock()
	defer l.mu.RUnlock()

	instances := make([]*GatewayInstance, 0, len(l.instances))
	for _, instance := range l.instances {
		instances = append(instances, instance)
	}
	return instances
}

// GetStats 获取路由统计信息
func (l *Locator) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	loadDist := l.ring.LoadDistribution()
	return map[string]interface{}{
		"active_gateways":   len(l.instances),
		"average_load":      l.ring.AverageLoad(),
		"load_distribution": loadDist,
		"last_sync_time":    l.lastSyncTime,
	}
}

// syncActiveGateways 从Redis同步活跃网关实例
func (l *Locator) syncActiveGateways() error {
	ctx := context.Background()

	// 1: 从Redis获取所有活跃的网关ID(无锁)
	minScore := strconv.FormatInt(time.Now().Unix()-HeartbeatWindow, 10)
	opt := &redis.ZRangeBy{Min: minScore, Max: "+inf"}
	activeIDs, err := l.redis.ZRangeByScore(ctx, ActiveGatewaysKey, opt)
	if err != nil {
		return fmt.Errorf("获取活跃网关列表失败: %v", err)
	}

	// 2: 构建期望的最新状态，并获取所有实例的详细信息(无锁)
	newState := make(map[string]*GatewayInstance)
	for _, instanceID := range activeIDs {
		instance, err := l.getInstanceDetails(ctx, instanceID)
		if err != nil {
			log.Printf("获取实例 %s 详细信息失败: %v, 将在本次同步中跳过该实例", instanceID, err)
			continue // 跳过获取失败的实例
		}
		newState[instanceID] = instance
	}

	// 3:对比并更新本地状态(一次性写锁)
	l.mu.Lock()
	defer l.mu.Unlock()

	var addedCount, removedCount int

	// 找出并移除已下线的实例
	for localID := range l.instances {
		if _, existsInNewState := newState[localID]; !existsInNewState {
			l.ring.Remove(localID)
			delete(l.instances, localID)
			removedCount++
			log.Printf("移除网关实例: %s", localID)
		}
	}

	// 找出并添加新上线的实例
	for newID, newInstance := range newState {
		if _, existsLocally := l.instances[newID]; !existsLocally {
			l.instances[newID] = newInstance
			l.ring.Add(newInstance)
			addedCount++
			log.Printf("添加网关实例: %s (%s)", newID, newInstance.GetAddress())
		}
	}

	if addedCount > 0 || removedCount > 0 {
		l.lastSyncTime = time.Now().Unix()
		log.Printf("网关实例同步完成，当前活跃数量: %d (新增: %d, 移除: %d)",
			len(l.instances), addedCount, removedCount)
	}

	return nil
}

// getInstanceDetails 从Redis Hash获取实例详细信息
func (l *Locator) getInstanceDetails(ctx context.Context, instanceID string) (*GatewayInstance, error) {
	key := fmt.Sprintf(GatewayInstanceHashKeyFmt, instanceID)
	fields, err := l.redis.HGetAll(ctx, key)
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
func (l *Locator) startMonitoring() {
	// 使用常量定义的同步间隔
	l.syncTicker = time.NewTicker(SyncInterval)
	go l.periodicSync()
}

// periodicSync 定期同步Redis中的活跃实例
func (l *Locator) periodicSync() {
	defer l.syncTicker.Stop()

	for {
		select {
		case <-l.syncTicker.C:
			if err := l.syncActiveGateways(); err != nil {
				log.Printf("定期同步活跃网关失败: %v", err)
			}
		case <-l.stopCh:
			return
		}
	}
}

// Stop 停止定位器
func (l *Locator) Stop() {
	close(l.stopCh)
	if l.syncTicker != nil {
		l.syncTicker.Stop()
	}
}

// ForceSync 强制同步（用于手动触发同步）
func (l *Locator) ForceSync() error {
	return l.syncActiveGateways()
}
