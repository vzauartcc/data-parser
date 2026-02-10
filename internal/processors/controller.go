package processors

import (
	"context"
	"errors"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"github.com/vzauartcc/data-parser/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var neighbors = []string{
	"ZOB",
	"ZID",
	"ZKC",
	"ZMP",
}

const (
	expiration = 65 * time.Second
	mainKey    = "controllers"
	tempKey    = "controllers:temp"
)

func ControllerFeed(ctx context.Context, controllers []datafeed.VnasUser, mongoDB database.MongoDatabase, redisClient cache.RedisClient) error {
	var (
		redisPipe         = redisClient.Pipeline()
		coll              = mongoDB.Collection("atcOnline")
		upsertModels      = make([]mongo.WriteModel, 0)
		sessionModels     = make([]mongo.WriteModel, 0)
		activeControllers = make([]any, 0)
		dataNeighbors     = make(map[string]struct{})
	)

	cids := make([]int, len(controllers))
	for i, c := range controllers {
		cids[i] = c.CID
	}

	userMap := make(map[int]models.User)

	cursor, _ := mongoDB.Collection("users").Find(ctx, bson.M{"cid": bson.M{"$in": cids}})
	if cursor != nil {
		var users []models.User

		_ = cursor.All(ctx, &users)
		for _, u := range users {
			userMap[u.CID] = u
		}

		cursor.Close(ctx)
	}

	for _, controller := range controllers {
		if controller.ArtccID == "ZAU" {
			user := userMap[controller.CID]

			controllerName := controller.VatsimData.RealName
			if user.FirstName != "" {
				controllerName = user.FirstName + " " + user.LastName
			}

			if controller.IsActive {
				toSave := bson.M{"$set": bson.M{
					"cid":       controller.CID,
					"name":      controllerName,
					"rating":    datafeed.GetRating(controller.VatsimData.RequestedRating),
					"pos":       controller.VatsimData.Callsign,
					"timeStart": controller.LoginTime,
					"atis":      controller.VatsimData.ControllerInfo,
					"frequency": controller.VatsimData.PrimaryFrequency,
				}}

				model := mongo.NewUpdateOneModel().
					SetFilter(bson.M{"callsign": controller.VatsimData.Callsign}).
					SetUpdate(toSave).
					SetUpsert(true)

				upsertModels = append(upsertModels, model)
				activeControllers = append(activeControllers, controller.VatsimData.Callsign)
			}

			sessionDoc := logSession(ctx, controller, mongoDB)
			if sessionDoc != nil {
				sessionModels = append(sessionModels, sessionDoc)
			}

			key := "CONTROLLER:" + controller.VatsimData.Callsign

			redisPipe.Expire(ctx, key, 5*time.Minute)
			redisPipe.Publish(ctx, "CONTROLLER:UPDATE", controller.VatsimData.Callsign)
		} else if slices.Contains(neighbors, controller.ArtccID) {
			dataNeighbors[controller.ArtccID] = struct{}{}
		}
	}

	if len(sessionModels) > 0 {
		_, err := mongoDB.Collection("controllerHours").BulkWrite(ctx, sessionModels, options.BulkWrite().SetOrdered(false))
		if err != nil {
			log.Printf("Error saving sessions: %v\n", err)
		}
	}

	var expiredControllers []string

	if len(activeControllers) > 0 {
		redisClient.SAdd(ctx, tempKey, activeControllers...)
		redisClient.Expire(ctx, tempKey, expiration)

		expiredControllers, _ = redisClient.SDiff(ctx, mainKey, tempKey).Result()

		err := redisClient.Rename(ctx, tempKey, mainKey).Err()
		if err != nil {
			log.Printf("Error renaming controllers key: %v\n", err)
		}

		redisClient.Expire(ctx, mainKey, expiration)
	} else {
		expiredControllers, _ = redisClient.SMembers(ctx, mainKey).Result()

		_, err := redisClient.Del(ctx, mainKey).Result()
		if err != nil {
			log.Printf("Error deleting controllers key: %v\n", err)
		}
	}

	_, err := redisPipe.Exec(ctx)
	if err != nil {
		log.Printf("Error executing controller pipe: %v\n", err)
	}

	if len(expiredControllers) > 0 {
		deletePipe := redisClient.Pipeline()

		for _, controller := range expiredControllers {
			deletePipe.LPush(ctx, "1231231231231231231231", controller)
			deletePipe.Publish(ctx, "CONTROLLER:DELETE", controller)
		}

		_, _ = deletePipe.Exec(ctx)

		_, err := coll.DeleteMany(ctx, bson.M{"callsign": bson.M{"$in": expiredControllers}})
		if err != nil {
			log.Printf("Error deleting offline controllers: %v\n", err)
		}
	}

	if len(upsertModels) > 0 {
		_, err := coll.BulkWrite(ctx, upsertModels, options.BulkWrite().SetOrdered(false))
		if err != nil {
			log.Printf("Error saving controllers to MongoDB: %v\n", err)
		}
	}

	neighs := make([]string, 0)
	for key := range dataNeighbors {
		neighs = append(neighs, key)
	}

	if len(neighs) > 0 {
		_, err = redisClient.Set(ctx, "neighbors", strings.Join(neighs, "|"), 0*time.Second).Result()
		if err != nil {
			log.Printf("Error publishing neighbors changes: %v\n", err)
		}
	}

	return nil
}

func logSession(ctx context.Context, controller datafeed.VnasUser, mongoDB database.MongoDatabase) mongo.WriteModel {
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

			return mongo.NewInsertOneModel().
				SetDocument(bson.M{"$set": bson.M{
					"cid":          controller.CID,
					"timeStart":    startTime,
					"timeEnd":      time.Now(),
					"position":     controller.VatsimData.Callsign,
					"isStudent":    controller.Role == "Student",
					"isInstructor": controller.Role == "Instructor",
					"wentInactive": false,
				}})
		}
	} else {
		session.TimeEnd = time.Now()

		if !controller.IsActive && !controller.IsObserver {
			log.Printf("%d's session went inactive, updating record\n", controller.CID)

			session.WentInactive = true
		}

		return mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": session.ID}).SetUpdate(bson.M{"$set": bson.M{
			"timeEnd":      session.TimeEnd,
			"timeStart":    session.TimeStart,
			"wentInactive": session.WentInactive,
		}}).SetUpsert(true)
	}

	return nil
}
