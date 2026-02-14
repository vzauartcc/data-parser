package processors

import (
	"context"
	"log"
	"time"

	"github.com/vzauartcc/data-parser/internal/cache"
)

func MetarFeed(ctx context.Context, data []string, redis cache.RedisClient) error {
	pipe := redis.Pipeline()

	for _, metar := range data {
		if len(metar) > 4 {
			pipe.Set(ctx, "METAR:"+metar[0:4], metar, 10*time.Minute)
		} else {
			log.Printf("Skipping invalid METAR: %s\n", metar)
		}
	}

	_, err := pipe.Exec(ctx)

	return err
}

func ExtendMetarTTL(ctx context.Context, redis cache.RedisClient) error {
	log.Println("Extending METAR TTL")

	var cursor uint64

	for {
		keys, nextCursor, err := redis.Scan(ctx, cursor, "METAR:*", 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			pipe := redis.Pipeline()
			for _, key := range keys {
				pipe.Expire(ctx, key, 10*time.Minute)
			}

			_, err := pipe.Exec(ctx)
			if err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}
