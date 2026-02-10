package processors

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	_, err := mongoDB.Collection("pilotsOnline").DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Printf("Error cleaning pilotsOnline collection: %v\n", err)
	}

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

		if config.Airports[pilot.FlightPlan.Departure] ||
			config.Airports[pilot.FlightPlan.Arrival] ||
			config.IsPointInAirspace(pilot.Latitude, pilot.Longitude) {
			plannedCruise := pilot.FlightPlan.RequestedAltitude
			if strings.Contains(pilot.FlightPlan.RequestedAltitude, "FL") {
				plannedCruise = strings.ReplaceAll(plannedCruise, "FL", "") + "00"
			}

			_, err = mongoDB.Collection("pilotsOnline").InsertOne(ctx, bson.M{
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
			if err != nil {
				log.Printf("Error inserting pilot %s: %v\n", pilot.Callsign, err)
			}

			dataPilots = append(dataPilots, pilot.Callsign)

			_, err = redisClient.HMSet(ctx, "PILOT:"+pilot.Callsign,
				"callsign", pilot.Callsign,
				"lat", fmt.Sprintf("%f", pilot.Latitude),
				"lng", fmt.Sprintf("%f", pilot.Longitude),
				"speed", strconv.Itoa(pilot.Groundspeed),
				"heading", strconv.Itoa(pilot.Heading),
				"altitude", strconv.Itoa(pilot.Altitude),
				"cruise", plannedCruise,
				"destination", pilot.FlightPlan.Arrival,
			).Result()
			if err != nil {
				log.Printf("Error setting pilot %s in redis: %v\n", pilot.Callsign, err)
			}

			_, err = redisClient.Expire(ctx, "PILOT:"+pilot.Callsign, 5*time.Minute).Result()
			if err != nil {
				log.Printf("Error expiring pilot %s: %v\n", pilot.Callsign, err)
			}

			_, err = redisClient.Publish(ctx, "PILOT:UPDATE", pilot.Callsign).Result()
			if err != nil {
				log.Printf("Error updating pilot %s: %v\n", pilot.Callsign, err)
			}
		}
	}

	for _, pilot := range redisPilots {
		if !slices.Contains(dataPilots, pilot) {
			_, err = redisClient.Publish(ctx, "PILOT:DELETE", pilot).Result()
			if err != nil {
				log.Printf("Error deleting pilot %s: %v\n", pilot, err)
			}
		}
	}

	_, err = redisClient.Set(ctx, "pilots", strings.Join(dataPilots, "|"), 65*time.Second).Result()
	if err != nil {
		log.Printf("Error setting online pilots: %v\n", err)
	}

	return nil
}
