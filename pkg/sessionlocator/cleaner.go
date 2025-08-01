package sessionlocator

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	redisClient "goim-social/pkg/redis"

	"github.com/go-redis/redis/v8"
)

/*
  TODO
  用于在实例下线/故障时,清理redis实例路由表。
  这是个临时方案,用redis实现了个分布式锁来进行领导者选举,以保证Cleaner是全局单点的,任务不被其他实例重复执行。
  后续定时清理的任务,将改为k8s cronJob执行或新增个monitor服务,待定。
*/

const (
	// LeaderLockKey 领导者选举锁的Redis key
	LeaderLockKey = "service:logic:leader_lock"

	// LeaderElectionInterval 领导者选举间隔
	LeaderElectionInterval = 30 * time.Second

	// LeaderLockTTL 领导者锁的TTL
	LeaderLockTTL = 60 * time.Second

	// CleanupTaskInterval 清理任务执行间隔
	CleanupTaskInterval = 5 * time.Minute
)

// Cleaner 网关实例清理器
type Cleaner struct {
	redis      *redisClient.RedisClient
	instanceID string
	isLeader   bool
	stopCh     chan struct{}

	// 选举相关
	electionTicker *time.Ticker

	// 清理任务相关
	cleanupTicker *time.Ticker
}

// NewCleaner 创建清理器
func NewCleaner(redis *redisClient.RedisClient, instanceID string) *Cleaner {
	return &Cleaner{
		redis:      redis,
		instanceID: instanceID,
		isLeader:   false,
		stopCh:     make(chan struct{}),
	}
}

// Start 启动清理器（包含领导者选举）
func (c *Cleaner) Start(ctx context.Context) {
	log.Printf("启动网关清理器，实例ID: %s", c.instanceID)

	// 启动领导者选举
	c.electionTicker = time.NewTicker(LeaderElectionInterval)
	go c.leaderElection(ctx)

	// 立即尝试一次选举
	go c.tryBecomeLeader(ctx)
}

// Stop 停止清理器
func (c *Cleaner) Stop() {
	log.Printf("停止网关清理器，实例ID: %s", c.instanceID)

	close(c.stopCh)

	if c.electionTicker != nil {
		c.electionTicker.Stop()
	}

	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}

	// 如果是领导者，释放锁
	if c.isLeader {
		ctx := context.Background()
		c.releaseLock(ctx)
	}
}

// leaderElection 领导者选举协程
func (c *Cleaner) leaderElection(ctx context.Context) {
	defer c.electionTicker.Stop()

	for {
		select {
		case <-c.electionTicker.C:
			c.tryBecomeLeader(ctx)
		case <-c.stopCh:
			return
		}
	}
}

// tryBecomeLeader 尝试成为领导者
func (c *Cleaner) tryBecomeLeader(ctx context.Context) {
	// 尝试获取领导者锁
	ok, err := c.redis.SetNX(ctx, LeaderLockKey, c.instanceID, LeaderLockTTL)
	if err != nil {
		log.Printf("领导者选举失败: %v", err)
		return
	}

	if ok {
		// 成功获取锁，成为领导者
		if !c.isLeader {
			log.Printf("成为领导者，开始执行清理任务")
			c.isLeader = true
			c.startCleanupTask(ctx)
		} else {
			// 已经是领导者，续期锁
			log.Printf("续期领导者锁")
		}
	} else {
		// 未能获取锁，检查当前领导者
		currentLeader, err := c.redis.Get(ctx, LeaderLockKey)
		if err != nil {
			log.Printf("获取当前领导者失败: %v", err)
		} else {
			if c.isLeader && currentLeader != c.instanceID {
				// 我之前是领导者，但现在不是了
				log.Printf("失去领导者身份，停止清理任务")
				c.isLeader = false
				c.stopCleanupTask()
			}
		}
	}
}

// startCleanupTask 启动清理任务
func (c *Cleaner) startCleanupTask(ctx context.Context) {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}

	c.cleanupTicker = time.NewTicker(CleanupTaskInterval)

	// 立即执行一次清理
	go c.executeCleanup(ctx)

	// 启动定期清理
	go func() {
		defer c.cleanupTicker.Stop()

		for {
			select {
			case <-c.cleanupTicker.C:
				if c.isLeader {
					c.executeCleanup(ctx)
				}
			case <-c.stopCh:
				return
			}
		}
	}()
}

// stopCleanupTask 停止清理任务
func (c *Cleaner) stopCleanupTask() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
		c.cleanupTicker = nil
	}
}

// executeCleanup 执行清理任务
func (c *Cleaner) executeCleanup(ctx context.Context) {
	log.Printf("执行网关实例清理任务...")

	// 计算过期时间戳
	expiredBefore := time.Now().Unix() - HeartbeatWindow

	// 1. 先获取要删除的实例数量（用于日志）
	expiredOpt := &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(expiredBefore, 10),
	}
	expiredInstances, err := c.redis.ZRangeByScore(ctx, ActiveGatewaysKey, expiredOpt)
	if err != nil {
		log.Printf("获取过期实例失败: %v", err)
		return
	}

	// 2. 清理ZSET中的过期实例
	if len(expiredInstances) > 0 {
		err = c.redis.ZRemRangeByScore(ctx, ActiveGatewaysKey, "0", strconv.FormatInt(expiredBefore, 10))
		if err != nil {
			log.Printf("清理ZSET过期实例失败: %v", err)
			return
		}
		log.Printf("从ZSET中清理了 %d 个过期网关实例: %v", len(expiredInstances), expiredInstances)
	}

	// 3. 清理孤儿Hash
	orphanedHashes, err := c.cleanupOrphanedHashes(ctx)
	if err != nil {
		log.Printf("清理孤儿Hash失败: %v", err)
	} else if orphanedHashes > 0 {
		log.Printf("清理了 %d 个孤儿Hash", orphanedHashes)
	}

	if len(expiredInstances) > 0 || orphanedHashes > 0 {
		log.Printf("过期实例清理完成: ZSET清理 %d 个, Hash清理 %d 个", len(expiredInstances), orphanedHashes)
	} else {
		log.Printf("清理任务完成，无过期实例")
	}
}

// cleanupOrphanedHashes 清理孤儿Hash
func (c *Cleaner) cleanupOrphanedHashes(ctx context.Context) (int, error) {
	// 获取所有gateway_instances:*的keys
	pattern := fmt.Sprintf(GatewayInstanceHashKeyFmt, "*")
	hashKeys, err := c.redis.Keys(ctx, pattern)
	if err != nil {
		return 0, fmt.Errorf("获取Hash keys失败: %v", err)
	}

	if len(hashKeys) == 0 {
		return 0, nil
	}

	// 获取ZSET中的所有实例ID
	allOpt := &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}
	activeIDs, err := c.redis.ZRangeByScore(ctx, ActiveGatewaysKey, allOpt)
	if err != nil {
		return 0, fmt.Errorf("获取ZSET实例失败: %v", err)
	}

	// 构建活跃实例ID的map
	activeIDMap := make(map[string]bool)
	for _, id := range activeIDs {
		activeIDMap[id] = true
	}

	// 检查并删除孤儿Hash
	orphanedCount := 0
	for _, hashKey := range hashKeys {
		// 从key中提取instanceID
		instanceID := strings.TrimPrefix(hashKey, "gateway_instances:")

		// 如果ZSET中没有这个实例，则删除Hash
		if !activeIDMap[instanceID] {
			if err := c.redis.Del(ctx, hashKey); err != nil {
				log.Printf("删除孤儿Hash %s 失败: %v", hashKey, err)
			} else {
				orphanedCount++
				log.Printf("删除孤儿Hash: %s", hashKey)
			}
		}
	}

	return orphanedCount, nil
}

// releaseLock 释放领导者锁
func (c *Cleaner) releaseLock(ctx context.Context) {
	// 只有当前实例是锁的持有者时才释放
	currentLeader, err := c.redis.Get(ctx, LeaderLockKey)
	if err != nil {
		log.Printf("获取当前领导者失败: %v", err)
		return
	}

	if currentLeader == c.instanceID {
		if err := c.redis.Del(ctx, LeaderLockKey); err != nil {
			log.Printf("释放领导者锁失败: %v", err)
		} else {
			log.Printf("已释放领导者锁")
		}
	}
}

// IsLeader 检查是否为领导者
func (c *Cleaner) IsLeader() bool {
	return c.isLeader
}

// GetLeaderInfo 获取当前领导者信息
func (c *Cleaner) GetLeaderInfo(ctx context.Context) (string, error) {
	return c.redis.Get(ctx, LeaderLockKey)
}
