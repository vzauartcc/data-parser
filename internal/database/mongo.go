package database

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func NewMongoClient(uri string) (*mongo.Client, error) {
	return mongo.Connect(
		options.
			Client().
			ApplyURI(uri).
			SetConnectTimeout(5 * time.Second))
}
