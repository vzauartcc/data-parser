package processors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"github.com/vzauartcc/data-parser/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const aircraftB738 = "B738"

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
			AircraftType:    aircraftB738,
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
			if doc["aircraft"] != aircraftB738 {
				t.Errorf("Expected aircraft 'B738', got %v", doc["aircraft"])
			}

			if doc["manual"] != false {
				t.Error("Expected manual flag to be false")
			}
		}
	}
}

func TestPirepModelDeserialization(t *testing.T) {
	doc := bson.M{"reportTime": time.Now(), "location": "KORD", "aircraft": "B738",
		"flightLevel": "340", "skyCond": "CLR", "turbulence": "MODERATE",
		"icing": "LI", "vis": "10", "temp": "25", "wind": "27010", "urgent": false,
		"raw": "KORD 123456Z", "manual": false}

	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal doc: %v", err)
	}

	var p models.Pirep
	err = bson.Unmarshal(data, &p)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if p.Temperature == nil || *p.Temperature != "25" {
		t.Errorf("Expected Temperature to be '25', got %v", p.Temperature)
	}
}

func intPtr(i int) *int { return &i }

func TestPirepWindSpeed(t *testing.T) {
	pirepColl := &database.MockCollection{}

	mockDB := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			"pireps": pirepColl,
		},
	}

	ctx := context.Background()

	windSpeed := 10
	windDir := 270
	pireps := []datafeed.Pirep{
		{
			// Valid PIREP with wind
			AircraftType:    "B738",
			RawObservation:  "KORD 123456Z",
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     340,
			Longitude:       -87.9,
			Latitude:        41.9,
			WindSpeed:       intPtr(windSpeed),
			WindDirection:   intPtr(windDir),
			PirepType:       "PIREP",
		},
		{
			// Valid PIREP without wind
			AircraftType:    "C172",
			RawObservation:  "KMDW 123456Z",
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     340,
			Longitude:       -87.9,
			Latitude:        41.9,
			PirepType:       "PIREP",
		},
	}

	err := PirepFeed(ctx, pireps, mockDB)
	if err != nil {
		t.Fatalf("PirepFeed returned an unexpected error: %v", err)
	}

	if len(pirepColl.BulkModels) != 2 {
		t.Fatalf("Expected 2 PIREPs to be inserted, got %d", len(pirepColl.BulkModels))
	}

	// Check first pirep (with wind)
	model1 := pirepColl.BulkModels[0]
	insertModel1, ok := model1.(*mongo.InsertOneModel)
	if !ok {
		t.Errorf("Expected model type *mongo.InsertOneModel, got %T", model1)
	} else {
		doc1 := insertModel1.Document.(bson.M)
		expectedWind := fmt.Sprintf("%d@%d", windDir, windSpeed)
		if doc1["wind"] != expectedWind {
			t.Errorf("Expected wind '%s', got %v", expectedWind, doc1["wind"])
		}
	}

	// Check second pirep (without wind)
	model2 := pirepColl.BulkModels[1]
	insertModel2, ok := model2.(*mongo.InsertOneModel)
	if !ok {
		t.Errorf("Expected model type *mongo.InsertOneModel, got %T", model2)
	} else {
		doc2 := insertModel2.Document.(bson.M)
		if doc2["wind"] != "" {
			t.Errorf("Expected empty wind, got %v", doc2["wind"])
		}
	}
}

func TestPirepFeed_ShortRawObservation(t *testing.T) {
	pirepColl := &database.MockCollection{}
	mockDB := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{"pireps": pirepColl},
	}

	pireps := []datafeed.Pirep{
		{
			AircraftType:    "B738",
			RawObservation:  "AB",
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     340,
			Longitude:       -87.9,
			Latitude:        41.9,
			PirepType:       "PIREP",
		},
	}

	if err := PirepFeed(context.Background(), pireps, mockDB); err != nil {
		t.Fatalf("PirepFeed returned an unexpected error: %v", err)
	}

	if len(pirepColl.BulkModels) != 0 {
		t.Errorf("Expected no inserts for a short raw observation, got %d", len(pirepColl.BulkModels))
	}
}
