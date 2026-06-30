package processors

import (
	"context"
	"log"

	"github.com/vzauartcc/data-parser/internal/database"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func Cleanup(ctx context.Context, mongoDB database.MongoDatabase) {
	_, err := mongoDB.Collection("pilotsOnline").DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Printf("Error cleaning up pilotsOnline collection: %v\n", err)
	}

	_, err = mongoDB.Collection("atcOnline").DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Printf("Error cleaning up atcOnline collection: %v\n", err)
	}
}
