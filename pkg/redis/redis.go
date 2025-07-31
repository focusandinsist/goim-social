package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient Redis客户端
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(addr string) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // 如果有密码，在这里设置
		DB:       0,  // 使用默认数据库
	})

	return &RedisClient{
		client: client,
	}
}

// GetClient 获取Redis客户端
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Set 设置键值对
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// SetNX 设置键值对（仅当键不存在时）
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// Get 获取值
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Del 删除键
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Expire 设置过期时间
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// HMSet 批量设置 hash 字段
func (r *RedisClient) HMSet(ctx context.Context, key string, fields map[string]interface{}) error {
	return r.client.HSet(ctx, key, fields).Err()
}

// HSet 设置单个 hash 字段
func (r *RedisClient) HSet(ctx context.Context, key, field string, value interface{}) error {
	return r.client.HSet(ctx, key, field, value).Err()
}

// Keys 按 pattern 查找 key
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

// SAdd 将成员添加到 set
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}

// SRem 从 set 移除成员
func (r *RedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SRem(ctx, key, members...).Err()
}

// SIsMember 检查成员是否在 set 中
func (r *RedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.client.SIsMember(ctx, key, member).Result()
}

// SMembers 获取 set 中的所有成员
func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// HGetAll 获取 hash 中的所有字段和值
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// Publish 发布消息到频道
func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Subscribe 订阅频道
func (r *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

// ZAdd 添加成员到有序集合
func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	return r.client.ZAdd(ctx, key, members...).Err()
}

// ZRangeByScore 根据分数范围获取有序集合成员
func (r *RedisClient) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return r.client.ZRangeByScore(ctx, key, opt).Result()
}

// ZRemRangeByScore 根据分数范围移除有序集合成员
func (r *RedisClient) ZRemRangeByScore(ctx context.Context, key string, min, max string) error {
	return r.client.ZRemRangeByScore(ctx, key, min, max).Err()
}

// ZRem 从有序集合移除成员
func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.ZRem(ctx, key, members...).Err()
}

// ZCard 获取有序集合成员数量
func (r *RedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	return r.client.ZCard(ctx, key).Result()
}

// Close 关闭连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}
