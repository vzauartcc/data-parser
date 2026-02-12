package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"github.com/vzauartcc/data-parser/internal/processors"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

type App struct {
	ctx     context.Context
	mongoDB *database.MongoRepo
	redisDB cache.RedisClient
}

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

	mongoRepo := &database.MongoRepo{Db: mongoClient.Database(mongoURI.Database)}

	redisClient := cache.NewRedisClient(os.Getenv("REDIS_URI"))

	defer func() {
		_ = redisClient.Close()
	}()

	err = redisClient.Ping(ctx).Err()
	if err != nil {
		panic(err)
	}

	app := &App{
		ctx:     ctx,
		mongoDB: mongoRepo,
		redisDB: redisClient,
	}

	runner := cron.New(cron.WithSeconds())

	_, err = runner.AddFunc("*/15 * * * * *", app.doVatsimFeed)
	if err != nil {
		log.Println("Failed to add cron job for VATSIM data")
		panic(err)
	}

	_, err = runner.AddFunc("0 * * * * *", app.doMetarFeed)
	if err != nil {
		log.Println("Failed to add cron job for METAR data")
		panic(err)
	}

	_, err = runner.AddFunc("*/15 * * * * *", app.doVnasFeed)
	if err != nil {
		log.Println("Failed to add cron job for vNAS data")
		panic(err)
	}

	_, err = runner.AddFunc("0 */5 * * * *", app.doPirepFeed)
	if err != nil {
		log.Println("Failed to add cron job for PIREPs")
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

func (app *App) doVatsimFeed() {
	if app.redisDB == nil {
		log.Println("Redis client is not set up.")
		return
	}

	if app.mongoDB == nil {
		log.Println("MongoDB is not set up.")
		return
	}

	data, err := datafeed.FetchVatsimDatafeed(app.ctx)
	if err != nil {
		log.Printf("Error during VATSIM fetch: %v\n", err)
		return
	}

	err2 := processors.PilotFeed(app.ctx, data.Pilots, app.mongoDB, app.redisDB)
	if err2 != nil {
		log.Printf("Error processing pilots: %v\n", err2)
	}

	err3 := processors.AtisFeed(app.ctx, data.ATISs, app.redisDB)
	if err3 != nil {
		log.Printf("Error processing ATISs: %v\n", err3)
	}
}

func (app *App) doVnasFeed() {
	data, err := datafeed.FetchVnasFeed(app.ctx)
	if err != nil {
		log.Printf("Error during VNAS fetch: %v", err)
		return
	}

	err = processors.ControllerFeed(app.ctx, data.Controllers, app.mongoDB, app.redisDB)
	if err != nil {
		log.Printf("Error processing controllers: %v\n", err)
	}
}

func (app *App) doPirepFeed() {
	data, err := datafeed.FetchPirepFeed(app.ctx)
	if err != nil {
		log.Printf("Error during PIREP fetch: %v", err)
		return
	}

	err = processors.PirepFeed(app.ctx, data, app.mongoDB)
	if err != nil {
		log.Printf("Error processing PIREPs: %v\n", err)
	}
}

func (app *App) doMetarFeed() {
	data, err := datafeed.FetchMetarFeed(app.ctx)
	if err != nil {
		log.Printf("Error during METAR fetch: %v", err)
		return
	}

	err = processors.MetarFeed(app.ctx, data, app.redisDB)
	if err != nil {
		log.Printf("Error processing METARs: %v\n", err)
	}
}
