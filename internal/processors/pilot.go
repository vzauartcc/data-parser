package processors

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vzauartcc/data-parser/config"
	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func PilotFeed(ctx context.Context, pilots []datafeed.VatsimPilot, mongoDB database.MongoDatabase, redisClient cache.RedisClient) error {
	redisData, err := redisClient.Get(ctx, "pilots").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	var (
		dataPilotsSlice = make([]string, 0)
		dataPilotsMap   = make(map[string]struct{})
		redisPipe       = redisClient.Pipeline()
		coll            = mongoDB.Collection("pilotsOnline")
		upsertModels    = make([]mongo.WriteModel, 0)
	)

	redisPilots := make([]string, 0)
	if len(redisData) > 0 {
		redisPilots = strings.Split(redisData, "|")
	}

	for _, pilot := range pilots {
		if pilot.FlightPlan == nil ||
			strings.TrimSpace(pilot.FlightPlan.AircraftFAA) == "" ||
			strings.TrimSpace(pilot.FlightPlan.RequestedAltitude) == "" {
			continue
		}

		if config.Airports[pilot.FlightPlan.Departure] ||
			config.Airports[pilot.FlightPlan.Arrival] ||
			config.IsPointInAirspace(pilot.Latitude, pilot.Longitude) {
			plannedCruise := formatCruiseAltitude(pilot.FlightPlan.RequestedAltitude)

			toSave := bson.M{"$set": bson.M{
				"cid":           pilot.CID,
				"name":          pilot.Name,
				"callsign":      pilot.Callsign,
				"aircraft":      pilot.FlightPlan.AircraftFAA,
				"dep":           pilot.FlightPlan.Departure,
				"dest":          pilot.FlightPlan.Arrival,
				"code":          pilot.Transponder,
				"lat":           pilot.Latitude,
				"lng":           pilot.Longitude,
				"altitude":      pilot.Altitude,
				"heading":       pilot.Heading,
				"speed":         pilot.Groundspeed,
				"plannedCruise": plannedCruise,
				"route":         pilot.FlightPlan.Route,
				"remarks":       pilot.FlightPlan.Remarks,
			}}

			upsertModels = append(upsertModels, mongo.NewUpdateOneModel().SetFilter(bson.M{"callsign": pilot.Callsign}).SetUpdate(toSave).SetUpsert(true))

			dataPilotsMap[pilot.Callsign] = struct{}{}
			dataPilotsSlice = append(dataPilotsSlice, pilot.Callsign)

			key := "PILOT:" + pilot.Callsign

			redisPipe.HSet(ctx, key,
				"callsign", pilot.Callsign,
				"lat", pilot.Latitude,
				"lng", pilot.Longitude,
				"speed", pilot.Groundspeed,
				"heading", pilot.Heading,
				"altitude", pilot.Altitude,
				"cruise", plannedCruise,
				"destination", pilot.FlightPlan.Arrival)

			redisPipe.Expire(ctx, key, 5*time.Minute)
			redisPipe.Publish(ctx, "PILOT:UPDATE", pilot.Callsign)
		}
	}

	if len(dataPilotsSlice) > 0 {
		_, err = redisPipe.Exec(ctx)
		if err != nil {
			log.Printf("Error executing pilots pipeline: %v\n", err)
		}
	}

	if len(upsertModels) > 0 {
		_, err = coll.BulkWrite(ctx, upsertModels, options.BulkWrite().SetOrdered(false))
		if err != nil {
			log.Printf("Error saving pilots to MongoDB: %v\n", err)
		}
	}

	deletePipeline := redisClient.Pipeline()
	toDelete := []string{}

	for _, pilot := range redisPilots {
		_, ok := dataPilotsMap[pilot]
		if pilot != "" && !ok {
			toDelete = append(toDelete, pilot)
			deletePipeline.Publish(ctx, "PILOT:DELETE", pilot)
		}
	}

	if len(toDelete) > 0 {
		_, err = coll.DeleteMany(ctx, bson.M{"callsign": bson.M{"$in": toDelete}})
		if err != nil {
			log.Printf("Error deleting offline pilots: %v\n", err)
		}
	}

	deletePipeline.Set(ctx, "pilots", strings.Join(dataPilotsSlice, "|"), 65*time.Second)

	_, err = deletePipeline.Exec(ctx)
	if err != nil {
		log.Printf("Error publish pilot changes: %v\n", err)
	}

	return nil
}

func formatCruiseAltitude(alt string) string {
	if strings.Contains(alt, "FL") {
		return strings.ReplaceAll(alt, "FL", "") + "00"
	}

	return alt
}
