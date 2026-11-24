package processors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vzauartcc/data-parser/internal/datafeed"
)

const (
	pilotCallsignORDGnd  = "ORD_GND"
	testCollectionPilots = "pilotsOnline"
	pilotAircraftB738    = "B738"
)

func TestPilotFeed_Full(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	mongoMock, pilotColl := SetupPilotMongoMock(t, PilotMockConfig{
		CollectionPilots: testCollectionPilots,
	})

	// Pre-seed: DAL200 is in the pilots set (simulating previously active).
	err := helper.Mr.Set("pilots", "DAL200")
	if err != nil {
		t.Fatalf("failed to seed pilots: %v", err)
	}

	// Pre-seed PILOT:DAL200 hash so we can verify it gets deleted.
	helper.Mr.HSet("PILOT:DAL200", "callsign", "DAL200")

	pilots := []datafeed.VatsimPilot{
		{
			CID: 1234567, Callsign: "AAL100", Latitude: 30.0, Longitude: -80.0,
			FlightPlan: &datafeed.VatsimFlightPlan{
				Departure: "KORD", Arrival: "KLAX",
				AircraftFAA: pilotAircraftB738, RequestedAltitude: "34000",
			},
		},
	}

	err = PilotFeed(ctx, pilots, mongoMock, client)
	if err != nil {
		t.Fatalf("PilotFeed returned unexpected error: %v", err)
	}

	if len(pilotColl.BulkModels) != 1 {
		t.Errorf("Expected 1 Mongo BulkWrite model, got %d", len(pilotColl.BulkModels))
	}

	if pilotColl.DeletedFilter == nil {
		t.Error("Expected Mongo DeleteMany to be called for offline pilot DAL200, but it wasn't")
	}

	// Check that PILOT:AAL100 hash was created.
	exists, err := client.Exists(ctx, "PILOT:AAL100").Result()
	if err != nil || exists == 0 {
		t.Error("Expected PILOT:AAL100 hash to be created")
	}

	// Check that PILOT:DAL200 was deleted.
	if helper.Mr.Exists("PILOT:DAL200") {
		t.Errorf("Expected PILOT:DAL200 hash to be deleted")
	}

	// Check pilots key was updated to only contain AAL100.
	pilotsKey, err := client.Get(ctx, "pilots").Result()
	if err != nil {
		t.Fatalf("failed to get pilots key: %v", err)
	}

	if pilotsKey != "AAL100" {
		t.Errorf("Expected pilots key to be 'AAL100', got '%s'", pilotsKey)
	}
}

func TestPilotFeed_MissingRedisKey(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	mongoMock, _ := SetupPilotMongoMock(t, PilotMockConfig{
		CollectionPilots: testCollectionPilots,
	})

	// No pre-seed for "pilots" key.

	pilots := []datafeed.VatsimPilot{}

	err := PilotFeed(ctx, pilots, mongoMock, client)
	if err != nil {
		t.Fatalf("PilotFeed failed with missing Redis key: %v", err)
	}

	// Verify pilots key was NOT deleted when it didn't exist.
	if mongoMock.Collections[testCollectionPilots].DeletedFilter != nil {
		t.Error("Expected no pilots to be deleted when Redis key is missing")
	}

	// Production still writes pilots = "" with 65s TTL.
	pilotsKey, err := client.Get(ctx, "pilots").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		t.Fatalf("failed to get pilots key: %v", err)
	}

	if pilotsKey != "" {
		t.Errorf("Expected pilots key to be empty string, got '%s'", pilotsKey)
	}

	// Verify pilots key has 65s TTL.
	if got := helper.Mr.TTL("pilots"); got != 65*time.Second {
		t.Errorf("expected 65s TTL for pilots key, got %v", got)
	}
}

func TestPilotFeed_NoPilotsInAirspace(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	mongoMock, _ := SetupPilotMongoMock(t, PilotMockConfig{
		CollectionPilots: testCollectionPilots,
	})

	// Pre-seed: DAL200 and AAL100 are in the pilots set.
	err := helper.Mr.Set("pilots", "DAL200|AAL100")
	if err != nil {
		t.Fatalf("failed to seed pilots: %v", err)
	}

	// Pre-seed PILOT hashes so we can verify they get deleted.
	helper.Mr.HSet("PILOT:DAL200", "callsign", "DAL200")
	helper.Mr.HSet("PILOT:AAL100", "callsign", "AAL100")

	// Pilot with coordinates outside ZAU airspace.
	pilots := []datafeed.VatsimPilot{
		{
			CID: 1234567, Callsign: "XYZ999", Latitude: 0.0, Longitude: 0.0,
			FlightPlan: &datafeed.VatsimFlightPlan{
				Departure: "KJFK", Arrival: "KLAX",
				AircraftFAA: pilotAircraftB738, RequestedAltitude: "34000",
			},
		},
	}

	err = PilotFeed(ctx, pilots, mongoMock, client)
	if err != nil {
		t.Fatalf("PilotFeed failed: %v", err)
	}

	// Since DAL200 and AAL100 are not in the current batch, they should be deleted.
	// Check PILOT:DAL200 was deleted.
	if helper.Mr.Exists("PILOT:DAL200") {
		t.Errorf("Expected PILOT:DAL200 hash to be deleted")
	}

	// Check PILOT:AAL100 was deleted.
	if helper.Mr.Exists("PILOT:AAL100") {
		t.Errorf("Expected PILOT:AAL100 hash to be deleted")
	}

	// Check pilots key is now empty (no pilots in airspace).
	pilotsKey, err := client.Get(ctx, "pilots").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		t.Fatalf("failed to get pilots key: %v", err)
	}

	if pilotsKey != "" {
		t.Errorf("Expected pilots key to be empty, got '%s'", pilotsKey)
	}

	// Verify pilots key has 65s TTL.
	if got := helper.Mr.TTL("pilots"); got != 65*time.Second {
		t.Errorf("expected 65s TTL for pilots key, got %v", got)
	}
}
