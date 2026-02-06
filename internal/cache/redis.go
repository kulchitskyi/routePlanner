package cache

import (
	"context"
	"encoding/json"
	"time"
	
	"github.com/redis/go-redis/v9"
)

type RedisCache[T any] struct {
	client *redis.Client
}

func NewRedisCache[T any](client *redis.Client) *RedisCache[T] {
	return &RedisCache[T]{client: client}
}

func (r *RedisCache[T]) Get(ctx context.Context, key string) (T, error) {
	var result T
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(val, &result)
	return result, err
}

func (r *RedisCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}
