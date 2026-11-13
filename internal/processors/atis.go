package processors

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/vzauartcc/data-parser/config"
	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/datafeed"
)

func AtisFeed(ctx context.Context, atiss []datafeed.VatsimATIS, redisClient cache.RedisClient) error {
	const mainKey = "atis"

	const tempKey = "atis:temp"

	const expiration = 70 * time.Second

	activeAirports := make([]any, 0)
	pipe := redisClient.Pipeline()

	for _, atis := range atiss {
		airport, _, _ := strings.Cut(atis.Callsign, "_")
		if airport == "" || strings.TrimSpace(atis.ATISCode) == "" || !config.Airports[airport] {
			continue
		}

		activeAirports = append(activeAirports, airport)

		pipe.Expire(ctx, "ATIS:"+airport, 65*time.Second)
		pipe.Publish(ctx, "ATIS:"+airport, atis.ATISCode)
	}

	var expiredAirports []string

	if len(activeAirports) > 0 {
		redisClient.SAdd(ctx, tempKey, activeAirports...)
		redisClient.Expire(ctx, tempKey, expiration)

		var err error

		expiredAirports, err = redisClient.SDiff(ctx, mainKey, tempKey).Result()
		if err != nil {
			log.Printf("Error computing expired ATIS airports: %v\n", err)

			expiredAirports = nil
		}

		err = redisClient.Rename(ctx, tempKey, mainKey).Err()
		if err != nil {
			log.Printf("Error renaming atis key: %v\n", err)
		}
	} else {
		var err error

		expiredAirports, err = redisClient.SMembers(ctx, mainKey).Result()
		if err != nil {
			log.Printf("Error getting ATIS airports: %v\n", err)

			expiredAirports = nil
		}

		err = redisClient.Del(ctx, mainKey).Err()
		if err != nil {
			log.Printf("Error deleting atis key: %v\n", err)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Error executing ATIS pipeline: %v\n", err)
	}

	if len(expiredAirports) > 0 {
		deletePipe := redisClient.Pipeline()

		for _, airport := range expiredAirports {
			deletePipe.Publish(ctx, "ATIS:DELETE", airport)
			deletePipe.Del(ctx, "ATIS:"+airport)
		}

		_, err = deletePipe.Exec(ctx)
		if err != nil {
			log.Printf("Error cleaning up ATIS: %v\n", err)
		}
	}

	return nil
}
