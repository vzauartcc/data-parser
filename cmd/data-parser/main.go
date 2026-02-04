package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"github.com/vzauartcc/data-parser/internal/processors"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

var redisClient *redis.Client
var mongoDB *mongo.Database

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("data-parser starting. . . .")

	mongoURI, err := connstring.ParseAndValidate(os.Getenv("MONGO_URI"))
	if err != nil {
		log.Printf("Invalid MongoDB URI: %v", err)
		return
	}

	mongoClient, err := database.NewMongoClient(os.Getenv("MONGO_URI"))
	if err != nil {
		log.Printf("Error connecting to MongoDB: %v", err)
		return
	}

	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = mongoClient.Disconnect(shutCtx)
	}()

	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		panic(err)
	}

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

	runner.Start()

	log.Println("data-parser running. . . .")

	<-ctx.Done()

	log.Println("data-parser shutting down. . . .")

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

	go doVnasFeed(ctx)

	go doVatsimFeed(ctx)

	go doPirepFeed(ctx)
}

func doVnasFeed(ctx context.Context) {
	data, err := datafeed.FetchVnasFeed(ctx)
	if err != nil {
		log.Printf("Error during VNAS fetch: %v", err)
		return
	}

	err = processors.ControllerFeed(ctx, data.Controllers, mongoDB, redisClient)
	if err != nil {
		log.Printf("Error processing controllers: %v\n", err)
	}
}

func doVatsimFeed(ctx context.Context) {
	data, err := datafeed.FetchVatsimDatafeed(ctx)
	if err != nil {
		log.Printf("Error during VATSIM fetch: %v\n", err)
		return
	}

	err2 := processors.PilotFeed(ctx, data.Pilots, mongoDB, redisClient)
	if err2 != nil {
		log.Printf("Error processing pilots: %v\n", err2)
	}

	err3 := processors.AtisFeed(ctx, data.ATISs, mongoDB, redisClient)
	if err3 != nil {
		log.Printf("Error processing ATISs: %v\n", err3)
	}
}

func doPirepFeed(ctx context.Context) {
	data, err := datafeed.FetchPirepFeed(ctx)
	if err != nil {
		log.Printf("Error during PIREP fetch: %v", err)
		return
	}

	err = processors.PirepFeed(ctx, data, mongoDB)
	if err != nil {
		log.Printf("Error processing PIREPs: %v\n", err)
	}
}
