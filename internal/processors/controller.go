package processors

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"github.com/vzauartcc/data-parser/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func ControllerFeed(ctx context.Context, controllers []datafeed.VnasController, mongoDB *mongo.Database, redisClient *redis.Client) error {
	_, _ = mongoDB.Collection("atcOnline").DeleteMany(ctx, bson.M{})

	dataControllers := make([]string, 0)

	redisData, err := redisClient.Get(ctx, "controllers").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	redisControllers := make([]string, 0)
	if len(redisData) > 0 {
		redisControllers = strings.Split(redisData, "|")
	}

	neighbors := []string{
		"ZOB",
		"ZID",
		"ZKC",
		"ZMP",
	}
	dataNeighbors := make(map[string]struct{})

	for _, controller := range controllers {
		if controller.ArtccID == "ZAU" {
			cid, err := strconv.Atoi(controller.VatsimData.CID)
			if err != nil {
				continue
			}

			var user models.User

			_ = mongoDB.Collection("users").FindOne(ctx, bson.M{"cid": cid}).Decode(&user)

			controllerName := controller.VatsimData.RealName
			if user.FirstName != "" {
				controllerName = user.FirstName + " " + user.LastName
			}

			if controller.IsActive {
				_, _ = mongoDB.Collection("atcOnline").InsertOne(ctx, bson.M{
					"cid":       controller.VatsimData.CID,
					"name":      controllerName,
					"rating":    datafeed.GetRating(controller.VatsimData.RequestedRating),
					"pos":       controller.VatsimData.Callsign,
					"timeStart": controller.LoginTime,
					"atis":      controller.VatsimData.ControllerInfo,
					"frequency": controller.VatsimData.PrimaryFrequency,
				})

				dataControllers = append(dataControllers, controller.VatsimData.Callsign)
			}

			logSession(ctx, controller, mongoDB)

			_ = redisClient.Expire(ctx, "CONTROLLER:"+controller.VatsimData.Callsign, 5*time.Minute)
			_ = redisClient.Publish(ctx, "CONTROLLER:UPDATE", controller.VatsimData.Callsign)
		} else if slices.Contains(neighbors, controller.ArtccID) {
			dataNeighbors[controller.ArtccID] = struct{}{}
		}
	}

	for _, atc := range redisControllers {
		if !slices.Contains(dataControllers, atc) {
			data, err := json.Marshal(atc)
			if err != nil {
				_ = redisClient.LPush(ctx, "1231231231231231231231", string(data))
			}

			_ = redisClient.Publish(ctx, "CONTROLLER:DELETE", atc)
		}
	}

	_ = redisClient.Set(ctx, "controllers", strings.Join(dataControllers, "|"), 65*time.Second)

	neighs := make([]string, 0)
	for key := range dataNeighbors {
		neighs = append(neighs, key)
	}

	_ = redisClient.Set(ctx, "neighbors", strings.Join(neighs, "|"), 0*time.Second)

	return nil
}

func logSession(ctx context.Context, controller datafeed.VnasController, mongoDB *mongo.Database) {
	var session *models.ControllerHours

	_ = mongoDB.Collection("controllerHours").
		FindOne(ctx, bson.M{
			"cid":       controller.VatsimData.CID,
			"timeStart": controller.LoginTime,
		}).
		Decode(&session)

	if session == nil {
		if controller.IsActive || controller.IsObserver {
			log.Printf("Creating new session for %s on %s\n", controller.VatsimData.CID, controller.VatsimData.Callsign)
			_, _ = mongoDB.Collection("controllerHours").InsertOne(ctx, bson.M{
				"cid":          controller.VatsimData.CID,
				"timeStart":    controller.LoginTime,
				"timeEnd":      time.Now(),
				"position":     controller.VatsimData.Callsign,
				"isStudent":    controller.Role == "Student",
				"isInstructor": controller.Role == "Instructor",
			})
		}
	} else {
		session.TimeEnd = time.Now()
		if !controller.IsActive && !controller.IsObserver {
			session.TimeStart = session.TimeStart.Add(-1 * time.Second)
		}

		_, err := mongoDB.Collection("controllerHours").UpdateByID(ctx, session.ID, bson.M{
			"$set": bson.M{
				"timeEnd":   session.TimeEnd,
				"timeStart": session.TimeStart,
			},
		})

		if err != nil {
			log.Printf("failed to update session for %s: %+v", controller.VatsimData.CID, err)

		}

	}
}
