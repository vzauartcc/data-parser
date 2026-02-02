package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/vzauartcc/data-parser/config"
	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"github.com/vzauartcc/data-parser/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

var redisClient *redis.Client
var mongoDB *mongo.Database

var multiSpace = regexp.MustCompile(`\s+`)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("Application starting. . . .")

	mongoURI, err := connstring.ParseAndValidate(os.Getenv("MONGO_URI"))
	if err != nil {
		log.Fatalln("Invalid MongoDB URI", err)
	}

	mongoClient, err := database.NewMongoClient(os.Getenv("MONGO_URI"))
	if err != nil {
		log.Fatalln("Error connecting to MongoDB", err)
	}

	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(shutCtx)
	}()

	mongoDB = mongoClient.Database(mongoURI.Database)

	redisClient = cache.NewRedisClient(os.Getenv("REDIS_URI"))

	defer func() {
		_ = redisClient.Close()
	}()

	err = redisClient.Ping(ctx).Err()
	if err != nil {
		panic(err)
	}

	runner := cron.New(cron.WithSeconds())

	_, err = runner.AddFunc("*/15 * * * * *", func() {
		runParser(ctx)
	})
	if err != nil {
		panic(err)
	}

	log.Println("Waiting for next run. . . .")

	runner.Start()

	<-ctx.Done()

	log.Println("Shutting down. . . .")

	stopCtx := runner.Stop()

	<-stopCtx.Done()

	log.Println("Bye!")
}

func runParser(ctx context.Context) {
	if redisClient == nil {
		log.Fatalln("Redis client is not set up.")
	}

	if mongoDB == nil {
		log.Fatalln("MongoDB is not set up.")
	}

	log.Println("Performing task. . . .")

	go doVnasFeed(ctx)

	fmt.Println()

	go doVatsimFeed(ctx)

	fmt.Println()

	go doPirepFeed(ctx)
}

func doVnasFeed(ctx context.Context) {
	data, err := datafeed.FetchVnasFeed(ctx)
	if err != nil {
		log.Printf("Error during VNAS fetch: %v", err)
		return
	}

	err = processControllers(ctx, data.Controllers)
	if err != nil {
		log.Printf("Error processing controllers: %v\n", err)
	}
}

func processControllers(ctx context.Context, controllers []datafeed.VnasController) error {
	if redisClient == nil {
		log.Fatalln("Redis client is not set up.")
	}

	if mongoDB == nil {
		log.Fatalln("MongoDB is not set up.")
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
			log.Printf(
				"Processing ZAU Controller %s (%s) working %s\n",
				controller.VatsimData.RealName,
				controller.VatsimData.CID,
				controller.VatsimData.Callsign,
			)

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

			logSession(ctx, controller)

			_ = redisClient.Expire(ctx, "CONTROLLER:"+controller.VatsimData.Callsign, 5*time.Minute)
			_ = redisClient.Publish(ctx, "CONTROLLER:UPDATE", controller.VatsimData.Callsign)
		} else if slices.Contains(neighbors, controller.ArtccID) {
			log.Printf("Setting neighbor online %s\n", controller.ArtccID)
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

func logSession(ctx context.Context, controller datafeed.VnasController) {
	var session *models.ControllerHours

	_ = mongoDB.Collection("controllerHours").
		FindOne(ctx, bson.M{
			"cid":       controller.VatsimData.CID,
			"timeStart": controller.LoginTime,
		}).
		Decode(&session)

	if session == nil {
		if controller.IsActive || controller.IsObserver {
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

		_ = mongoDB.Collection("controllerHours").FindOneAndUpdate(
			ctx,
			bson.M{
				"_id": session.ID},
			bson.M{
				"timeEnd":   session.TimeEnd,
				"timeStart": session.TimeStart,
			})
	}
}

func doVatsimFeed(ctx context.Context) {
	data, err := datafeed.FetchVatsimDatafeed(ctx)
	if err != nil {
		log.Printf("Error during VATSIM fetch: %v\n", err)
		return
	}

	err2 := processPilots(ctx, data.Pilots)
	if err2 != nil {
		log.Printf("Error processing pilots: %v\n", err2)
	}

	err3 := processAtiss(ctx, data.ATISs)
	if err3 != nil {
		log.Printf("Error processing ATISs: %v\n", err3)
	}
}

func processPilots(ctx context.Context, pilots []datafeed.VatsimPilot) error {
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
			isPointInAirspace(pilot.Latitude, pilot.Longitude) {
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

func processAtiss(ctx context.Context, atiss []datafeed.VatsimATIS) error {
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

func doPirepFeed(ctx context.Context) {
	data, err := datafeed.FetchPirepFeed(ctx)
	if err != nil {
		log.Printf("Error during PIREP fetch: %v", err)
		return
	}

	err = processPireps(ctx, data)
	if err != nil {
		log.Printf("Error processing PIREPs: %v\n", err)
	}
}

func processPireps(ctx context.Context, pireps []datafeed.Pirep) error {
	for _, pirep := range pireps {
		if pirep.AircraftType == "" || pirep.RawObservation[:3] == "" ||
			pirep.ObservationTime == 0 ||
			pirep.FlightLevel == 0 {
			continue
		}

		if !isPointInAirspace(pirep.Latitude, pirep.Longitude) ||
			(pirep.PirepType != "PIREP" && pirep.PirepType != "Urgent PIREP") {
			continue
		}

		log.Printf("Processing PIREP: %s\n", pirep.RawObservation)

		windSpd := ""
		if pirep.WindSpeed != nil {
			windSpd = "@" + windSpd
		}

		windDir := ""
		if pirep.WindDirection != nil {
			windDir = fmt.Sprintf("%d", pirep.WindDirection)
		}

		wind := windDir + "" + windSpd

		icing := ""
		if pirep.IcingInterval1 != "" {
			icing = pirep.IcingInterval1 + " "
		}

		if pirep.IcingType1 != "" {
			icing += pirep.IcingType1
		}

		icing = strings.TrimSpace(multiSpace.ReplaceAllString(icing, " "))

		skyCond := ""

		turb := ""

		if pirep.Clouds != nil && len((*pirep.Clouds)) > 0 {
			base := fmt.Sprintf("000%d", (*pirep.Clouds)[0].Base)

			tops := ""
			if (*pirep.Clouds)[0].Tops != 0 {
				tops = fmt.Sprintf("000%d", (*pirep.Clouds)[0].Tops)
			}

			skyCond = fmt.Sprintf(
				"%s %s %s",
				(*pirep.Clouds)[0].Cover,
				base[len(base)-3:],
				tops[len(base)-3:],
			)

			if pirep.TurbulenceInterval1 != "" {
				turb = pirep.TurbulenceInterval1 + " "
			}

			if pirep.TurbulenceFrequency1 != "" {
				turb = turb + pirep.TurbulenceFrequency1 + " "
			}

			if pirep.TurbulenceType1 != "" {
				turb += pirep.TurbulenceType1
			}

			turb = strings.TrimSpace(multiSpace.ReplaceAllString(turb, " "))
		}

		_, _ = mongoDB.Collection("pireps").InsertOne(ctx, bson.M{
			"reportTime":  time.UnixMilli(int64(pirep.ObservationTime * 1000)),
			"location":    pirep.RawObservation[:3],
			"aircraft":    pirep.AircraftType,
			"flightLevel": pirep.FlightLevel,
			"skyCond":     skyCond,
			"turbulence":  turb,
			"icing":       icing,
			"vis":         pirep.Visibility,
			"temp":        pirep.Temperature,
			"wind":        wind,
			"urgent":      pirep.PirepType == "Urgent PIREP",
			"raw":         pirep.RawObservation,
			"manual":      false,
		})
	}

	return nil
}

func isPointInAirspace(lat float64, lon float64) bool {
	inside := false

	verticies := len(config.Airspace)
	if verticies < 3 {
		return false
	}

	// Iterate through each edge of airspace.
	j := verticies - 1
	for i := range verticies {
		xi, yi := config.Airspace[i][0], config.Airspace[i][1]
		xj, yj := config.Airspace[j][0], config.Airspace[j][1]

		// Check if the point's Longitude coordinate is within the edge's
		// Longitude range and if the ray casting to the right intersects the edge
		intersect := ((yi > lon) != (yj > lon)) &&
			(lat < (xj-xi)*(lon-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}

		j = i
	}

	return inside
}
