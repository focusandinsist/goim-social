package sessionlocator

import (
	"context"
	"fmt"
	"log"
	"time"

	redisClient "websocket-server/pkg/redis"

	"github.com/go-redis/redis/v8"
)

// HeartbeatManager 网关心跳管理器
// 用于网关服务自己管理注册、心跳和注销
type HeartbeatManager struct {
	redis      *redisClient.RedisClient
	instanceID string
	host       string
	port       int
	stopCh     chan struct{}
	ticker     *time.Ticker
}

// NewHeartbeatManager 创建心跳管理器
func NewHeartbeatManager(redis *redisClient.RedisClient, instanceID, host string, port int) *HeartbeatManager {
	return &HeartbeatManager{
		redis:      redis,
		instanceID: instanceID,
		host:       host,
		port:       port,
		stopCh:     make(chan struct{}),
	}
}

// Start 启动心跳管理器
func (hm *HeartbeatManager) Start(ctx context.Context) error {
	// 注册网关实例
	if err := hm.register(ctx); err != nil {
		return fmt.Errorf("注册网关实例失败: %v", err)
	}

	// 启动心跳
	hm.startHeartbeat()

	log.Printf("心跳管理器已启动: %s (%s:%d)", hm.instanceID, hm.host, hm.port)
	return nil
}

// Stop 停止心跳管理器
func (hm *HeartbeatManager) Stop(ctx context.Context) error {
	// 停止心跳
	close(hm.stopCh)
	if hm.ticker != nil {
		hm.ticker.Stop()
	}

	// 注销网关实例
	if err := hm.unregister(ctx); err != nil {
		log.Printf("注销网关实例失败: %v", err)
	}

	log.Printf("心跳管理器已停止: %s", hm.instanceID)
	return nil
}

// register 注册网关实例到Redis ZSET
func (hm *HeartbeatManager) register(ctx context.Context) error {
	// 添加到Redis ZSET
	score := float64(time.Now().Unix())
	z := &redis.Z{Score: score, Member: hm.instanceID}
	if err := hm.redis.ZAdd(ctx, ActiveGatewaysKey, z); err != nil {
		return fmt.Errorf("注册到Redis ZSET失败: %v", err)
	}

	// 可选：保存实例详细信息到Hash（用于获取host、port等信息）
	instanceKey := fmt.Sprintf(GatewayInstanceHashKeyFmt, hm.instanceID)
	instanceInfo := map[string]interface{}{
		"id":             hm.instanceID,
		"host":           hm.host,
		"port":           hm.port,
		"registered_at":  time.Now().Unix(),
		"last_heartbeat": time.Now().Unix(),
	}

	if err := hm.redis.HMSet(ctx, instanceKey, instanceInfo); err != nil {
		log.Printf("保存实例详细信息失败: %v", err)
	}

	// 设置Hash过期时间（心跳窗口+30秒缓冲）
	expireTime := time.Duration(HeartbeatWindow+30) * time.Second
	if err := hm.redis.Expire(ctx, instanceKey, expireTime); err != nil {
		log.Printf("设置实例信息过期时间失败: %v", err)
	}

	return nil
}

// unregister 从Redis ZSET注销网关实例
func (hm *HeartbeatManager) unregister(ctx context.Context) error {
	// 从Redis ZSET移除
	if err := hm.redis.ZRem(ctx, ActiveGatewaysKey, hm.instanceID); err != nil {
		return fmt.Errorf("从Redis ZSET移除失败: %v", err)
	}

	// 删除实例详细信息
	instanceKey := fmt.Sprintf(GatewayInstanceHashKeyFmt, hm.instanceID)
	if err := hm.redis.Del(ctx, instanceKey); err != nil {
		log.Printf("删除实例详细信息失败: %v", err)
	}

	return nil
}

// startHeartbeat 启动心跳循环
func (hm *HeartbeatManager) startHeartbeat() {
	// 使用常量定义的心跳间隔
	hm.ticker = time.NewTicker(HeartbeatInterval)

	go func() {
		defer hm.ticker.Stop()

		for {
			select {
			case <-hm.ticker.C:
				ctx := context.Background()
				if err := hm.sendHeartbeat(ctx); err != nil {
					log.Printf("发送心跳失败: %v", err)
				}
			case <-hm.stopCh:
				return
			}
		}
	}()
}

// sendHeartbeat 发送心跳
func (hm *HeartbeatManager) sendHeartbeat(ctx context.Context) error {
	// 更新Redis ZSET中的分数（时间戳）
	score := float64(time.Now().Unix())
	z := &redis.Z{Score: score, Member: hm.instanceID}
	if err := hm.redis.ZAdd(ctx, ActiveGatewaysKey, z); err != nil {
		return fmt.Errorf("更新心跳失败: %v", err)
	}

	// 更新实例详细信息中的心跳时间
	instanceKey := fmt.Sprintf(GatewayInstanceHashKeyFmt, hm.instanceID)
	if err := hm.redis.HSet(ctx, instanceKey, "last_heartbeat", time.Now().Unix()); err != nil {
		log.Printf("更新实例心跳时间失败: %v", err)
	}

	// 续期Hash（心跳窗口+30秒缓冲）
	expireTime := time.Duration(HeartbeatWindow+30) * time.Second
	if err := hm.redis.Expire(ctx, instanceKey, expireTime); err != nil {
		log.Printf("续期实例信息失败: %v", err)
	}

	return nil
}

// GetInstanceID 获取实例ID
func (hm *HeartbeatManager) GetInstanceID() string {
	return hm.instanceID
}

// GetAddress 获取实例地址
func (hm *HeartbeatManager) GetAddress() string {
	return fmt.Sprintf("%s:%d", hm.host, hm.port)
}

// IsRunning 检查心跳管理器是否正在运行
func (hm *HeartbeatManager) IsRunning() bool {
	select {
	case <-hm.stopCh:
		return false
	default:
		return hm.ticker != nil
	}
}
