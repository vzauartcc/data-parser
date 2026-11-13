package processors

import (
	"context"
	"slices"
	"testing"

	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
)

func TestPilotFeed_Full(t *testing.T) {
	ctx := context.Background()

	mockPipe := &cache.MockPipeline{}
	redisMock := &cache.MockRedis{
		Pipe: mockPipe,
		Data: map[string]string{"pilots": "DAL200"},
	}

	pilotColl := &database.MockCollection{}
	mongoMock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			"pilotsOnline": pilotColl,
		},
	}

	pilots := []datafeed.VatsimPilot{
		{
			CID: 1234567, Callsign: "AAL100", Latitude: 30.0, Longitude: -80.0,
			FlightPlan: &datafeed.VatsimFlightPlan{
				Departure: "KORD", Arrival: "KLAX",
				AircraftFAA: aircraftB738, RequestedAltitude: "34000",
			},
		},
	}

	err := PilotFeed(ctx, pilots, mongoMock, redisMock)
	if err != nil {
		t.Fatalf("PilotFeed returned unexpected error: %v", err)
	}

	if len(pilotColl.BulkModels) != 1 {
		t.Errorf("Expected 1 Mongo BulkWrite model, got %d", len(pilotColl.BulkModels))
	}

	if pilotColl.DeletedFilter == nil {
		t.Error("Expected Mongo DeleteMany to be called for offline pilot DAL200, but it wasn't")
	}

	var hasDeletePublish, hasUpdatePublish bool

	for _, cmd := range mockPipe.RecordedCmds {
		if cmd == "PUBLISH:PILOT:DELETE:DAL200" {
			hasDeletePublish = true
		}

		if cmd == "PUBLISH:PILOT:UPDATE:AAL100" {
			hasUpdatePublish = true
		}
	}

	if !hasUpdatePublish {
		t.Error("Missing Redis Publish for updated pilot AAL100")
	}

	if !hasDeletePublish {
		t.Error("Missing Redis Publish for deleted pilot DAL200")
	}

	foundDel := slices.Contains(mockPipe.ExpiredKeys, "PILOT:DAL200")
	if !foundDel {
		t.Errorf("Expected PILOT:DAL200 hash to be deleted, got %v", mockPipe.ExpiredKeys)
	}
}

func TestPilotFeed_MissingRedisKey(t *testing.T) {
	ctx := context.Background()

	redisMock := &cache.MockRedis{
		Pipe: &cache.MockPipeline{Published: make(map[string]string)},
		Data: make(map[string]string),
	}

	pilotColl := &database.MockCollection{}
	mongoMock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			"pilotsOnline": pilotColl,
		},
	}

	pilots := []datafeed.VatsimPilot{}

	err := PilotFeed(ctx, pilots, mongoMock, redisMock)
	if err != nil {
		t.Fatalf("PilotFeed failed with missing Redis key: %v", err)
	}

	if pilotColl.DeletedFilter != nil {
		t.Error("Expected no pilots to be deleted when Redis key is missing")
	}
}
