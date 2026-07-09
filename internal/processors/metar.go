package processors

import (
	"context"
	"encoding/json"
	"log"
	"time"

	metartafparser "github.com/ryansavara/go-metar-taf-parser"
	"github.com/vzauartcc/data-parser/internal/cache"
)

func MetarFeed(ctx context.Context, data []metartafparser.Metar, redis cache.RedisClient) error {
	pipe := redis.Pipeline()

	for _, metar := range data {
		station := metar.Station
		if len(station) != 4 {
			log.Printf("Skipping invalid METAR: %s\n", metar.Message)
			continue
		}

		jsonData, err := json.Marshal(metar)
		if err != nil {
			log.Printf("Failed to marshal METAR: %s\n", err)
			continue
		}

		pipe.Set(ctx, "METAR:"+station, jsonData, 10*time.Minute)
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
