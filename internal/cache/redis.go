package cache

import "github.com/redis/go-redis/v9"

func NewRedisClient(addr string) *redis.Client {
	opt, err := redis.ParseURL(addr)
	if err != nil {
		panic(err)
	}

	return redis.NewClient(opt)
}
