package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCmdable interface {
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Publish(ctx context.Context, channel string, message any) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	SAdd(ctx context.Context, key string, members ...any) *redis.IntCmd
	SDiff(ctx context.Context, keys ...string) *redis.StringSliceCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	Rename(ctx context.Context, key, newkey string) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
}

type RedisClient interface {
	RedisCmdable
	Pipeline() redis.Pipeliner
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

func NewRedisClient(addr string) *redis.Client {
	opt, err := redis.ParseURL(addr)
	if err != nil {
		panic(err)
	}

	return redis.NewClient(opt)
}
