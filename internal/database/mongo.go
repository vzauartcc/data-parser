package database

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Collections struct {
	AtcOnline       *mongo.Collection
	AtisOnline      *mongo.Collection
	PilotOnline     *mongo.Collection
	Pirep           *mongo.Collection
	ControllerHours *mongo.Collection
	Users           *mongo.Collection
}

func NewMongoClient(uri string) (*mongo.Client, error) {
	return mongo.Connect(
		options.
			Client().
			ApplyURI(uri).
			SetConnectTimeout(5 * time.Second))
}
