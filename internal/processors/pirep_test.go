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

const (
	aircraftB738       = "B738"
	collectionPireps   = "pireps"
	rawObservationKORD = "KORD 123456Z"
	pirepTypePIREP     = "PIREP"
	locationKORD       = "KORD"
	rawObservationAB   = "AB"
)

func TestPirepFeed(t *testing.T) {
	pirepColl := &database.MockCollection{}

	mockDB := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			collectionPireps: pirepColl,
		},
	}

	ctx := context.Background()

	pireps := []datafeed.Pirep{
		{
			// Valid PIREP
			AircraftType:    aircraftB738,
			RawObservation:  rawObservationKORD,
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     340,
			Longitude:       -87.9,
			Latitude:        41.9,
			PirepType:       pirepTypePIREP,
		},
		{
			// Invalid PIREP: Missing FlightLevel (should be skipped)
			AircraftType:    "C172",
			RawObservation:  "KSFO 123456Z",
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     0,
			Longitude:       -122.3,
			Latitude:        37.6,
			PirepType:       pirepTypePIREP,
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

			if doc["location"] != locationKORD {
				t.Errorf("Expected location 'KORD', got %v", doc["location"])
			}

			if doc["manual"] != false {
				t.Error("Expected manual flag to be false")
			}
		}
	}

	// Verify DeletedFilter is based on expiration time, not the $or condition
	filter, ok := pirepColl.DeletedFilter.(bson.M)
	if !ok {
		t.Errorf("Expected DeletedFilter to be bson.M, got %T", pirepColl.DeletedFilter)
	} else {
		if filter["$or"] != nil {
			t.Errorf("Expected DeletedFilter to not contain $or, got %v", filter)
		}

		if filter["reportTime"] == nil {
			t.Errorf("Expected DeletedFilter to contain reportTime condition")
		}
	}
}

func TestPirepFeed_CleanupFilter(t *testing.T) {
	ctx := context.Background()

	pirepColl := &database.MockCollection{}

	mockDB := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			collectionPireps: pirepColl,
		},
	}

	// Run PirepFeed with empty pireps to just trigger cleanup
	err := PirepFeed(ctx, []datafeed.Pirep{}, mockDB)
	if err != nil {
		t.Fatalf("PirepFeed returned an unexpected error: %v", err)
	}

	filter, filterOk := pirepColl.DeletedFilter.(bson.M)
	if !filterOk {
		t.Fatalf("Expected DeletedFilter to be bson.M, got %T", pirepColl.DeletedFilter)
	}

	// Should use expiration-based filter, not $or with manual: false
	if filter["$or"] != nil {
		t.Errorf("Expected DeletedFilter to not contain $or, got %v", filter)
	}

	reportTimeFilter, filterOk := filter["reportTime"].(bson.M)
	if !filterOk {
		t.Errorf("Expected reportTime filter, got %v", filter["reportTime"])
	}

	if reportTimeFilter["$lte"] == nil {
		t.Errorf("Expected $lte condition in reportTime filter")
	}
}

func TestPirepModelDeserialization(t *testing.T) {
	flightLevel := 340
	visibility := 10
	temp := 25

	doc := bson.M{"reportTime": time.Now(), "location": locationKORD, "aircraft": aircraftB738,
		"flightLevel": flightLevel, "skyCond": "CLR", "turbulence": "MODERATE",
		"icing": "LI", "vis": &visibility, "temp": &temp, "wind": "27010", "urgent": false,
		"raw": rawObservationKORD, "manual": false}

	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal doc: %v", err)
	}

	var pirepModel models.Pirep

	err = bson.Unmarshal(data, &pirepModel)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if pirepModel.Temperature == nil || *pirepModel.Temperature != temp {
		t.Errorf("Expected Temperature to be %d, got %v", temp, pirepModel.Temperature)
	}

	if pirepModel.FlightLevel != flightLevel {
		t.Errorf("Expected FlightLevel to be %d, got %d", flightLevel, pirepModel.FlightLevel)
	}
}

func TestPirepModelRoundTrip(t *testing.T) {
	flightLevel := 340
	visibility := 10
	temp := 25

	raw, err := bson.Marshal(bson.M{
		"reportTime":  time.Now(),
		"location":    locationKORD,
		"aircraft":    aircraftB738,
		"flightLevel": flightLevel,
		"skyCond":     "CLR",
		"turbulence":  "",
		"icing":       "",
		"vis":         &visibility,
		"temp":        &temp,
		"wind":        "27010",
		"urgent":      false,
		"raw":         rawObservationKORD,
		"manual":      false,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var pirep models.Pirep

	err = bson.Unmarshal(raw, &pirep)
	if err != nil {
		t.Fatalf("unmarshal into models.Pirep: %v", err)
	}

	if pirep.FlightLevel != flightLevel {
		t.Errorf("Expected flightLevel %d, got %d", flightLevel, pirep.FlightLevel)
	}

	if pirep.Temperature == nil || *pirep.Temperature != temp {
		t.Errorf("Expected temp %d, got %v", temp, pirep.Temperature)
	}
}

func intPtr(i int) *int { return &i }

func TestPirepWindSpeed(t *testing.T) {
	pirepColl := &database.MockCollection{}

	mockDB := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			collectionPireps: pirepColl,
		},
	}

	ctx := context.Background()

	windSpeed := 10
	windDir := 270
	pireps := []datafeed.Pirep{
		{
			// Valid PIREP with wind
			AircraftType:    aircraftB738,
			RawObservation:  rawObservationKORD,
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     340,
			Longitude:       -87.9,
			Latitude:        41.9,
			WindSpeed:       intPtr(windSpeed),
			WindDirection:   intPtr(windDir),
			PirepType:       pirepTypePIREP,
		},
		{
			// Valid PIREP without wind
			AircraftType:    "C172",
			RawObservation:  "KMDW 123456Z",
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     340,
			Longitude:       -87.9,
			Latitude:        41.9,
			PirepType:       pirepTypePIREP,
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

	insertModel1, insertOK := model1.(*mongo.InsertOneModel)
	if !insertOK {
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

	insertModel2, insertOK2 := model2.(*mongo.InsertOneModel)
	if !insertOK2 {
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
		Collections: map[string]*database.MockCollection{collectionPireps: pirepColl},
	}

	pireps := []datafeed.Pirep{
		{
			AircraftType:    aircraftB738,
			RawObservation:  rawObservationAB,
			ObservationTime: int(time.Now().Unix()),
			FlightLevel:     340,
			Longitude:       -87.9,
			Latitude:        41.9,
			PirepType:       "PIREP",
		},
	}

	err := PirepFeed(context.Background(), pireps, mockDB)
	if err != nil {
		t.Fatalf("PirepFeed returned an unexpected error: %v", err)
	}

	if len(pirepColl.BulkModels) != 0 {
		t.Errorf("Expected no inserts for a short raw observation, got %d", len(pirepColl.BulkModels))
	}
}
