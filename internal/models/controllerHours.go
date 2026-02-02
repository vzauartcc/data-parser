package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type ControllerHours struct {
	ID           bson.ObjectID `bson:"_id"`
	CID          int           `bson:"cid"`
	TimeStart    time.Time     `bson:"timeStart"`
	TimeEnd      time.Time     `bson:"timeEnd"`
	Position     string        `bson:"position"`
	IsStudent    bool          `bson:"isStudent"`
	IsInstructor bool          `bson:"isInstructor"`
}
