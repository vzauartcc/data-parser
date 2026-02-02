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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func AtisFeed(ctx context.Context, atiss []datafeed.VatsimATIS, mongoDB *mongo.Database, redisClient *redis.Client) error {
	_, _ = mongoDB.Collection("atisOnline").DeleteMany(ctx, bson.M{})

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
			log.Printf("Saving ATIS code %s for %s\n", atis.ATISCode, atis.Callsign)

			dataAtis = append(dataAtis, airport)
			_ = redisClient.Expire(ctx, "ATIS:"+airport, 65*time.Second)
			_ = redisClient.Publish(ctx, "ATIS:"+airport, atis.ATISCode)
		}
	}

	for _, atis := range redisAtis {
		if !slices.Contains(dataAtis, atis) {
			_ = redisClient.Publish(ctx, "ATIS:DELETE", atis)
			_ = redisClient.Del(ctx, "ATIS:"+atis)
		}
	}

	_ = redisClient.Set(ctx, "atis", strings.Join(dataAtis, "|"), 65*time.Second)

	return nil
}
