package processors

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vzauartcc/data-parser/config"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func PilotFeed(ctx context.Context, pilots []datafeed.VatsimPilot, mongoDB *mongo.Database, redisClient *redis.Client) error {
	_, _ = mongoDB.Collection("pilotsOnline").DeleteMany(ctx, bson.M{})

	dataPilots := make([]string, 0)

	redisData, err := redisClient.Get(ctx, "pilots").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

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

		if slices.Contains(config.Airports, pilot.FlightPlan.Departure) ||
			slices.Contains(config.Airports, pilot.FlightPlan.Arrival) ||
			config.IsPointInAirspace(pilot.Latitude, pilot.Longitude) {
			plannedCruise := pilot.FlightPlan.RequestedAltitude
			if strings.Contains(pilot.FlightPlan.RequestedAltitude, "FL") {
				plannedCruise = strings.ReplaceAll(plannedCruise, "FL", "") + "00"
			}

			_, _ = mongoDB.Collection("pilotsOnline").InsertOne(ctx, bson.M{
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
			})

			dataPilots = append(dataPilots, pilot.Callsign)

			_ = redisClient.HMSet(ctx, "PILOT:"+pilot.Callsign,
				"callsign", pilot.Callsign,
				"lat", fmt.Sprintf("%f", pilot.Latitude),
				"lng", fmt.Sprintf("%f", pilot.Longitude),
				"speed", strconv.Itoa(pilot.Groundspeed),
				"heading", strconv.Itoa(pilot.Heading),
				"altitude", strconv.Itoa(pilot.Altitude),
				"cruise", plannedCruise,
				"destination", pilot.FlightPlan.Arrival,
			)

			_ = redisClient.Expire(ctx, "PILOT:"+pilot.Callsign, 5*time.Minute)
			_ = redisClient.Publish(ctx, "PILOT:UPDATE", pilot.Callsign)
		}
	}

	for _, pilot := range redisPilots {
		if !slices.Contains(dataPilots, pilot) {
			_ = redisClient.Publish(ctx, "PILOT:DELETE", pilot)
		}
	}

	_ = redisClient.Set(ctx, "pilots", strings.Join(dataPilots, "|"), 65*time.Second)

	return nil
}
