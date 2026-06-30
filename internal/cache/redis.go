package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
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
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
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

	opt.ClientName = "data-parser"
	opt.MaintNotificationsConfig = &maintnotifications.Config{
		Mode: maintnotifications.ModeDisabled,
	}

	return redis.NewClient(opt)
}
