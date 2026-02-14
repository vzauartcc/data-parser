package models

import "go.mongodb.org/mongo-driver/v2/bson"

type AtisOnline struct {
	ID       bson.ObjectID `bson:"_id,omitempty"`
	CID      int           `bson:"cid"`
	Airport  string        `bson:"airport"`
	Callsign string        `bson:"callsign"`
	Code     string        `bson:"code"`
	Text     string        `bson:"text"`
}
