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
			pipe.Set(ctx, "METAR:"+metar[0:4], metar, 5*time.Minute)
		} else {
			log.Printf("Skipping invalid METAR: %s\n", metar)
		}
	}

	_, err := pipe.Exec(ctx)

	return err
}
