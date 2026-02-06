package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type AtcOnline struct {
	ID        bson.ObjectID `bson:"_id"`
	CID       int           `bson:"cid"`
	Name      string        `bson:"name"`
	Rating    int           `bson:"rating"`
	Position  string        `bson:"pos"`
	TimeStart time.Time     `bson:"timeStart"`
	Atis      *string       `bson:"atis"`
	Frequency int           `bson:"frequency"`
}
