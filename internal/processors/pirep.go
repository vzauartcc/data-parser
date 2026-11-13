package processors

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vzauartcc/data-parser/config"
	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var multiSpace = regexp.MustCompile(`\s+`)

func PirepFeed(ctx context.Context, pireps []datafeed.Pirep, mongoDB database.MongoDatabase) error {
	expired := time.Now().Add(-2 * time.Hour)

	_, err := mongoDB.Collection("pireps").DeleteMany(ctx, bson.M{
		"reportTime": bson.M{"$lte": expired},
	})
	if err != nil {
		log.Printf("Error cleaning up pirep collection: %v\n", err)
	}

	upsertModels := make([]mongo.WriteModel, 0)

	for _, pirep := range pireps {
		if pirep.AircraftType == "" || len(pirep.RawObservation) < 3 ||
			pirep.ObservationTime == 0 ||
			pirep.FlightLevel == 0 {
			continue
		}

		if !config.IsPointInAirspace(pirep.Longitude, pirep.Latitude) ||
			(pirep.PirepType != "PIREP" && pirep.PirepType != "Urgent PIREP") {
			continue
		}

		windSpd := ""
		if pirep.WindSpeed != nil {
			windSpd = "@" + strconv.Itoa(*pirep.WindSpeed)
		}

		windDir := ""
		if pirep.WindDirection != nil {
			windDir = strconv.Itoa(*pirep.WindDirection)
		}

		wind := windDir + windSpd

		icing := ""
		if pirep.IcingInterval1 != "" {
			icing = pirep.IcingInterval1 + " "
		}

		if pirep.IcingType1 != "" {
			icing += pirep.IcingType1
		}

		icing = strings.TrimSpace(multiSpace.ReplaceAllString(icing, " "))

		skyCond := ""

		turb := ""

		if pirep.Clouds != nil && len((*pirep.Clouds)) > 0 {
			base := fmt.Sprintf("%03d", ((*pirep.Clouds)[0].Base)/100)

			tops := "000"
			if (*pirep.Clouds)[0].Tops != 0 {
				tops = fmt.Sprintf("%03d", ((*pirep.Clouds)[0].Tops / 100))
			}

			skyCond = fmt.Sprintf(
				"%s %s %s",
				(*pirep.Clouds)[0].Cover,
				base,
				tops,
			)

			if pirep.TurbulenceInterval1 != "" {
				turb = pirep.TurbulenceInterval1 + " "
			}

			if pirep.TurbulenceFrequency1 != "" {
				turb = turb + pirep.TurbulenceFrequency1 + " "
			}

			if pirep.TurbulenceType1 != "" {
				turb += pirep.TurbulenceType1
			}

			turb = strings.TrimSpace(multiSpace.ReplaceAllString(turb, " "))
		}

		location := ""
		if fields := strings.Fields(pirep.RawObservation); len(fields) > 0 {
			location = fields[0]
		}

		upsertModels = append(upsertModels, mongo.NewInsertOneModel().SetDocument(bson.M{
			"reportTime":  time.UnixMilli(int64(pirep.ObservationTime * 1000)),
			"location":    location,
			"aircraft":    pirep.AircraftType,
			"flightLevel": pirep.FlightLevel,
			"skyCond":     skyCond,
			"turbulence":  turb,
			"icing":       icing,
			"vis":         pirep.Visibility,
			"temp":        pirep.Temperature,
			"wind":        wind,
			"urgent":      pirep.PirepType == "Urgent PIREP",
			"raw":         pirep.RawObservation,
			"manual":      false,
		}))
	}

	_, err = mongoDB.Collection("pireps").BulkWrite(ctx, upsertModels, options.BulkWrite().SetOrdered(false))

	return err
}
