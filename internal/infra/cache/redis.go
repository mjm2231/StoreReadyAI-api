package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"storeready_ai/internal/config"

	"github.com/redis/go-redis/v9"
)

// Cache 定义缓存的最小能力集合。
//
// 说明：
//   - 以 bytes 作为核心数据类型，避免在基础设施层强行绑定某种序列化协议。
//   - JSON 便捷方法仅用于常见业务场景，底层仍走 bytes。
type Cache interface {
	GetBytes(ctx context.Context, key string) ([]byte, error)
	SetBytes(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
	Exists(ctx context.Context, keys ...string) (int64, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	IncrBy(ctx context.Context, key string, delta int64) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Ping(ctx context.Context) error
}

// RedisCache 基于 go-redis/v9 的缓存实现。
//
// 设计点：
//   - KeyPrefix：用于多环境/多模块隔离 key
//   - DefaultTTL：调用方不传 ttl 或 ttl<=0 时使用默认值（0 表示不过期）
//
// 注意：
//   - 复杂的分布式锁/一致性/多级缓存不放在这里，避免 infra 层过重。
//   - 如需 lock，可在业务侧或单独 package 扩展。
type RedisCache struct {
	Client     redis.UniversalClient
	KeyPrefix  string
	DefaultTTL time.Duration
}

// NewRedis 创建一个 Redis 客户端，并返回关闭函数。
//
// 说明：
//   - 当前按单机模式初始化（addr 形如 127.0.0.1:6379）。
//   - 若未来需要支持集群/哨兵，可扩展 config 并在此处分支。
func NewRedis(cfg config.RedisConfig) (redis.UniversalClient, func() error, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, nil, errors.New("redis: addr 不能为空")
	}

	opt := &redis.Options{
		Addr:         addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	c := redis.NewClient(opt)

	// 启动时做一次轻量连通性检测
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, nil, fmt.Errorf("redis: ping 失败: %w", err)
	}

	closeFn := func() error {
		return c.Close()
	}
	return c, closeFn, nil
}

// NewRedisCache 创建 RedisCache。
func NewRedisCache(client redis.UniversalClient, keyPrefix string, defaultTTL time.Duration) *RedisCache {
	return &RedisCache{Client: client, KeyPrefix: strings.TrimSpace(keyPrefix), DefaultTTL: defaultTTL}
}

// Ping 检测 Redis 可用性。
func (r *RedisCache) Ping(ctx context.Context) error {
	if r == nil || r.Client == nil {
		return errors.New("redis cache: client 为空")
	}
	return r.Client.Ping(ctx).Err()
}

// GetBytes 获取缓存（bytes）。
func (r *RedisCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("redis cache: client 为空")
	}
	k := r.k(key)
	b, err := r.Client.Get(ctx, k).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, redis.Nil
		}
		return nil, err
	}
	return b, nil
}

// SetBytes 写入缓存（bytes）。
//
// ttl<=0 时使用 DefaultTTL（DefaultTTL=0 表示不过期）。
func (r *RedisCache) SetBytes(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	if r == nil || r.Client == nil {
		return errors.New("redis cache: client 为空")
	}
	k := r.k(key)
	useTTL := ttl
	if useTTL <= 0 {
		useTTL = r.DefaultTTL
	}
	return r.Client.Set(ctx, k, val, useTTL).Err()
}

// Del 删除 key。
func (r *RedisCache) Del(ctx context.Context, keys ...string) (int64, error) {
	if r == nil || r.Client == nil {
		return 0, errors.New("redis cache: client 为空")
	}
	ks := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		ks = append(ks, r.k(k))
	}
	if len(ks) == 0 {
		return 0, nil
	}
	return r.Client.Del(ctx, ks...).Result()
}

// Exists 判断 key 是否存在。
func (r *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	if r == nil || r.Client == nil {
		return 0, errors.New("redis cache: client 为空")
	}
	ks := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		ks = append(ks, r.k(k))
	}
	if len(ks) == 0 {
		return 0, nil
	}
	return r.Client.Exists(ctx, ks...).Result()
}

// TTL 获取剩余过期时间。
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if r == nil || r.Client == nil {
		return 0, errors.New("redis cache: client 为空")
	}
	k := r.k(key)
	return r.Client.TTL(ctx, k).Result()
}

// IncrBy 自增（常用于计数/限流/滑动窗口）。
func (r *RedisCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	if r == nil || r.Client == nil {
		return 0, errors.New("redis cache: client 为空")
	}
	k := r.k(key)
	return r.Client.IncrBy(ctx, k, delta).Result()
}

// Expire 设置过期时间。
func (r *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if r == nil || r.Client == nil {
		return false, errors.New("redis cache: client 为空")
	}
	k := r.k(key)
	return r.Client.Expire(ctx, k, ttl).Result()
}

// SetJSON 便捷方法：将对象序列化为 JSON 写入缓存。
func (r *RedisCache) SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return r.SetBytes(ctx, key, b, ttl)
}

// GetJSON 便捷方法：读取 JSON 并反序列化到 out（指针）。
func (r *RedisCache) GetJSON(ctx context.Context, key string, out any) error {
	b, err := r.GetBytes(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// k 规范化 key：自动拼接前缀，避免重复冒号。
func (r *RedisCache) k(key string) string {
	key = strings.TrimSpace(key)
	p := strings.TrimSpace(r.KeyPrefix)
	if p == "" {
		return key
	}
	p = strings.TrimSuffix(p, ":")
	key = strings.TrimPrefix(key, ":")
	return p + ":" + key
}
