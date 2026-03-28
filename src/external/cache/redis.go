package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

type redisCache struct {
	client *redis.Client
}

// NewRedisCache creates a Redis-backed CacheService.
func NewRedisCache(opts *redis.Options) (port.CacheService, *redis.Client) {
	client := redis.NewClient(opts)
	return &redisCache{client: client}, client
}

func (c *redisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *redisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("cache miss: %w", apperror.ErrCacheMiss)
	}
	if err != nil {
		return fmt.Errorf("redis get: %w", err)
	}
	return json.Unmarshal(data, dest)
}

func (c *redisCache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

func (c *redisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("marshal: %w", err)
	}
	return c.client.SetNX(ctx, key, data, ttl).Result()
}
