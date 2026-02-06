package processors

import (
	"context"
	"errors"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vzauartcc/data-parser/config"
	"github.com/vzauartcc/data-parser/internal/datafeed"
)

func AtisFeed(ctx context.Context, atiss []datafeed.VatsimATIS, redisClient *redis.Client) error {
	dataAtis := make([]string, 0)

	redisData, err := redisClient.Get(ctx, "atis").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	redisAtis := make([]string, 0)
	if len(redisData) > 0 {
		redisAtis = strings.Split(redisData, "|")
	}

	for _, atis := range atiss {
		airport := atis.Callsign[0:4]
		if slices.Contains(config.Airports, airport) {
			dataAtis = append(dataAtis, airport)

			_, err = redisClient.Expire(ctx, "ATIS:"+airport, 65*time.Second).Result()
			if err != nil {
				log.Printf("Error setting expiry for ATIS:%s: %v\n", airport, err)
			}

			_, err = redisClient.Publish(ctx, "ATIS:"+airport, atis.ATISCode).Result()
			if err != nil {
				log.Printf("Error publishing ATIS:%s: %v\n", airport, err)
			}
		}
	}

	for _, atis := range redisAtis {
		if !slices.Contains(dataAtis, atis) {
			_, err = redisClient.Publish(ctx, "ATIS:DELETE", atis).Result()
			if err != nil {
				log.Printf("Error publishing  ATIS:DELETE for %s: %v\n", atis, err)
			}

			_, err = redisClient.Del(ctx, "ATIS:"+atis).Result()
			if err != nil {
				log.Printf("Error deleting ATIS:%s: %v\n", atis, err)
			}
		}
	}

	_, err = redisClient.Set(ctx, "atis", strings.Join(dataAtis, "|"), 65*time.Second).Result()
	if err != nil {
		log.Printf("Error setting list of online ATISs: %v\n", err)
	}

	return nil
}
