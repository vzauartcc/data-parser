package processors

import (
	"context"
	"testing"
	"time"

	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestPirepFeed(t *testing.T) {
	pirepColl := &database.MockCollection{}

	mockDB := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			"pireps": pirepColl,
		},
	}

	ctx := context.Background()

	pireps := []datafeed.Pirep{
		{
			// Valid PIREP
			AircraftType:    "B738",
			RawObservation:  "KORD 123456Z",
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     340,
			Longitude:       -87.9,
			Latitude:        41.9,
			PirepType:       "PIREP",
		},
		{
			// Invalid PIREP: Missing FlightLevel (should be skipped)
			AircraftType:    "C172",
			RawObservation:  "KSFO 123456Z",
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     0,
			Longitude:       -122.3,
			Latitude:        37.6,
			PirepType:       "PIREP",
		},
		{
			// Invalid PIREP: Not a PIREP or Urgent PIREP type (should be skipped)
			AircraftType:    "A320",
			RawObservation:  "KATL 123456Z",
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     200,
			PirepType:       "AIREP",
		},
	}

	err := PirepFeed(ctx, pireps, mockDB)
	if err != nil {
		t.Fatalf("PirepFeed returned an unexpected error: %v", err)
	}

	if pirepColl.DeletedFilter == nil {
		t.Error("Expected DeleteMany to be called for collection cleanup, but DeletedFilter is nil")
	}

	expectedInserts := 1
	actualInserts := len(pirepColl.BulkModels)

	if actualInserts != expectedInserts {
		t.Errorf("Expected %d PIREPs to be queued for BulkWrite, but found %d", expectedInserts, actualInserts)
	}

	if actualInserts > 0 {
		model := pirepColl.BulkModels[0]
		insertModel, ok := model.(*mongo.InsertOneModel)

		if !ok {
			t.Errorf("Expected model type *mongo.InsertOneModel, got %T", model)
		} else {
			doc := insertModel.Document.(bson.M)
			if doc["aircraft"] != "B738" {
				t.Errorf("Expected aircraft 'B738', got %v", doc["aircraft"])
			}

			if doc["manual"] != false {
				t.Error("Expected manual flag to be false")
			}
		}
	}
}
