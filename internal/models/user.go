package models

import "go.mongodb.org/mongo-driver/v2/bson"

type User struct {
	ID        bson.ObjectID `bson:"_id"`
	CID       int           `bson:"cid"`
	FirstName string        `bson:"fname"`
	LastName  string        `bson:"lname"`
}
