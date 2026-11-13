package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Pirep struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	ReportTime    time.Time     `bson:"reportTime"`
	Location      string        `bson:"location"`
	Aircraft      string        `bson:"aircraft"`
	FlightLevel   int           `bson:"flightLevel"`
	SkyConditions *string       `bson:"skyCond"`
	Turbulence    *string       `bson:"turbulence"`
	Icing         *string       `bson:"icing"`
	Visibility    *int          `bson:"vis"`
	Temperature   *int          `bson:"temp"`
	Wind          *string       `bson:"wind"`
	Urgent        bool          `bson:"urgent"`
	Raw           string        `bson:"raw"`
	Manual        bool          `bson:"manual"`
}
