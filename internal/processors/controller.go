package processors

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"github.com/vzauartcc/data-parser/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ControllerFeed(ctx context.Context, controllers []datafeed.VnasUser, mongoDB *mongo.Database, redisClient *redis.Client) error {
	_, err := mongoDB.Collection("atcOnline").DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Printf("Error cleaning atc online collection: %v\n", err)
	}

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
			var user models.User

			err = mongoDB.Collection("users").FindOne(ctx, bson.M{"cid": controller.CID}).Decode(&user)
			if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
				log.Printf("Failed to decode user for %d: %v\n", controller.CID, err)
			}

			controllerName := controller.VatsimData.RealName
			if user.FirstName != "" {
				controllerName = user.FirstName + " " + user.LastName
			}

			if controller.IsActive {
				_, err = mongoDB.Collection("atcOnline").InsertOne(ctx, bson.M{
					"cid":       controller.CID,
					"name":      controllerName,
					"rating":    datafeed.GetRating(controller.VatsimData.RequestedRating),
					"pos":       controller.VatsimData.Callsign,
					"timeStart": controller.LoginTime,
					"atis":      controller.VatsimData.ControllerInfo,
					"frequency": controller.VatsimData.PrimaryFrequency,
				})
				if err != nil {
					log.Printf("Error inserting online atc for %d: %v\n", controller.CID, err)
				}

				dataControllers = append(dataControllers, controller.VatsimData.Callsign)
			}

			logSession(ctx, controller, mongoDB)

			_, err = redisClient.Expire(ctx, "CONTROLLER:"+controller.VatsimData.Callsign, 5*time.Minute).Result()
			if err != nil {
				log.Printf("Error expiring old controller info for %s: %v\n", controller.VatsimData.Callsign, err)
			}

			_, err = redisClient.Publish(ctx, "CONTROLLER:UPDATE", controller.VatsimData.Callsign).Result()
			if err != nil {
				log.Printf("Error publishing controller info for %s: %v\n", controller.VatsimData.Callsign, err)
			}
		} else if slices.Contains(neighbors, controller.ArtccID) {
			dataNeighbors[controller.ArtccID] = struct{}{}
		}
	}

	for _, atc := range redisControllers {
		if !slices.Contains(dataControllers, atc) {
			data, err := json.Marshal(atc)
			if err != nil {
				_, err2 := redisClient.LPush(ctx, "1231231231231231231231", string(data)).Result()
				if err2 != nil {
					log.Printf("Error queuing controller %s: %v\n", atc, err2)
				}
			}

			_, err = redisClient.Publish(ctx, "CONTROLLER:DELETE", atc).Result()
			if err != nil {
				log.Printf("Error publishing CONTROLLER:DELETE for %s: %v\n", atc, err)
			}
		}
	}

	_, err = redisClient.Set(ctx, "controllers", strings.Join(dataControllers, "|"), 65*time.Second).Result()
	if err != nil {
		log.Printf("Error setting online controllers: %v\n", err)
	}

	neighs := make([]string, 0)
	for key := range dataNeighbors {
		neighs = append(neighs, key)
	}

	_, err = redisClient.Set(ctx, "neighbors", strings.Join(neighs, "|"), 0*time.Second).Result()
	if err != nil {
		log.Printf("Error setting online neighbors: %v\n", err)
	}

	return nil
}

func logSession(ctx context.Context, controller datafeed.VnasUser, mongoDB *mongo.Database) {
	var session *models.ControllerHours

	err := mongoDB.Collection("controllerHours").
		FindOne(ctx, bson.M{
			"cid": controller.CID,
			"timeStart": bson.M{
				"$gte": controller.LoginTime,
				"$lt":  time.Now(),
			},
		}, options.FindOne().SetSort(bson.M{"timeStart": -1})).
		Decode(&session)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		log.Printf("Failed to decode session for %d: %v\n", controller.CID, err)
	}

	if session == nil || session.WentInactive {
		if controller.IsActive || controller.IsObserver {
			log.Printf("Creating new session for %d on %s\n", controller.CID, controller.VatsimData.Callsign)

			startTime := controller.LoginTime
			if session != nil && session.WentInactive {
				startTime = time.Now().Add(-1 * time.Second)
			}

			_, err = mongoDB.Collection("controllerHours").InsertOne(ctx, bson.M{
				"cid":          controller.CID,
				"timeStart":    startTime,
				"timeEnd":      time.Now(),
				"position":     controller.VatsimData.Callsign,
				"isStudent":    controller.Role == "Student",
				"isInstructor": controller.Role == "Instructor",
				"wentInactive": false,
			})
			if err != nil {
				log.Printf("Failed to insert session for %d: %v\n", controller.CID, err)
			}
		}
	} else {
		session.TimeEnd = time.Now()

		if !controller.IsActive && !controller.IsObserver {
			log.Printf("%d's session went inactive, updating record\n", controller.CID)

			session.WentInactive = true
		}

		_, err := mongoDB.Collection("controllerHours").UpdateByID(ctx, session.ID, bson.M{
			"$set": bson.M{
				"timeEnd":      session.TimeEnd,
				"timeStart":    session.TimeStart,
				"wentInactive": session.WentInactive,
			},
		})
		if err != nil {
			log.Printf("failed to update session for %d: %+v\n", controller.CID, err)
		}
	}
}
