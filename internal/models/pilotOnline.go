package models

import "go.mongodb.org/mongo-driver/v2/bson"

type PilotOnline struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	CID           int           `bson:"cid"`
	Name          string        `bson:"name"`
	Callsign      string        `bson:"callsign"`
	AircraftType  string        `bson:"aircraft"`
	Departure     *string       `bson:"dep"`
	Destination   *string       `bson:"dest"`
	Code          *string       `bson:"code"`
	Latitude      float64       `bson:"lat"`
	Longitude     float64       `bson:"lng"`
	Altitude      int           `bson:"altitude"`
	Heading       int           `bson:"heading"`
	Speed         int           `bson:"speed"`
	PlannedCruise string        `bson:"planned_cruise"`
	Route         *string       `bson:"route"`
	Remarks       *string       `bson:"remarks"`
}
