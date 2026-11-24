package processors

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/vzauartcc/data-parser/internal/datafeed"
)

const (
	testCallsignZOBApp  = "ZOB_APP"
	testCallsignORDGnd  = "ORD_GND"
	testCallsignORDApp  = "ORD_APP"
	testCallsignORDTwr  = "ORD_TWR"
	testArtccZOB        = "ZOB"
	testUserFirstName   = "John"
	testUserLastName    = "Doe"
	testCollectionATC   = "atcOnline"
	testCollectionUsers = "users"
	testCollectionHours = "controllerHours"
)

func TestControllerFeed(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	mongoMock, atcColl := SetupControllerMongoMock(t, ControllerMockConfig{
		CID:             1234567,
		UserFirstName:   testUserFirstName,
		UserLastName:    testUserLastName,
		Position:        testCallsignORDGnd,
		CollectionATC:   testCollectionATC,
		CollectionUsers: testCollectionUsers,
		CollectionHours: testCollectionHours,
	})

	// Pre-seed: ORD_GND is in the main controllers set (simulating previously active).
	// The feed will have ORD_APP active, so ORD_GND should be expired/cleaned up.
	_, err := helper.Mr.SAdd("controllers", testCallsignORDGnd)
	if err != nil {
		t.Fatalf("failed to seed controllers: %v", err)
	}

	// Pre-seed the CONTROLLER:ORD_GND hash so we can verify it gets deleted.
	helper.Mr.HSet("CONTROLLER:"+testCallsignORDGnd, "cid", "1234567")

	controllers := []datafeed.VnasUser{
		{
			CID: 1234567,
			VnasController: datafeed.VnasController{
				ArtccID:  "ZAU",
				IsActive: true,
				VatsimData: datafeed.VnasVatsimData{
					Callsign: testCallsignORDApp, // ORD_APP is active, ORD_GND should be cleaned up
				},
			},
		},
	}

	// Subscribe to CONTROLLER:UPDATE BEFORE calling ControllerFeed.
	sub := client.Subscribe(ctx, "CONTROLLER:UPDATE")

	err = ControllerFeed(ctx, controllers, mongoMock, client)
	if err != nil {
		t.Fatalf("ControllerFeed failed: %v", err)
	}

	// Verify CONTROLLER:UPDATE was published for the active controller.
	if got := waitForChannel(t, sub, "CONTROLLER:UPDATE"); got != testCallsignORDApp {
		t.Errorf("expected CONTROLLER:UPDATE payload %q, got %q", testCallsignORDApp, got)
	}

	sub.Close()

	// Verify ORD_GND was deleted (expired).
	members, err := client.SMembers(ctx, "controllers").Result()
	if err != nil {
		t.Fatalf("failed to get controllers members: %v", err)
	}

	if slices.Contains(members, testCallsignORDGnd) {
		t.Errorf("Expected %s to be removed from controllers set", testCallsignORDGnd)
	}

	// Verify CONTROLLER:ORD_GND key was deleted.
	if helper.Mr.Exists("CONTROLLER:" + testCallsignORDGnd) {
		t.Errorf("Expected CONTROLLER:%s hash to be deleted", testCallsignORDGnd)
	}

	// Verify MongoDB DeleteMany was called.
	if atcColl.DeletedFilter == nil {
		t.Error("Mongo DeleteMany was not called")
	}

	// Verify MongoDB upsert was called for the active controller.
	if got := len(atcColl.BulkModels); got != 1 {
		t.Errorf("expected 1 Mongo upsert model, got %d", got)
	}
}

func TestControllerFeed_Neighbors(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	mongoMock, _ := SetupControllerMongoMock(t, ControllerMockConfig{
		CID:             1234567,
		UserFirstName:   testUserFirstName,
		UserLastName:    testUserLastName,
		Position:        testCallsignORDGnd,
		CollectionATC:   testCollectionATC,
		CollectionUsers: testCollectionUsers,
		CollectionHours: testCollectionHours,
	})

	// Pre-seed: a ZAU controller (ORD_TWR) in the controllers set.
	_, err := helper.Mr.SAdd("controllers", testCallsignORDTwr)
	if err != nil {
		t.Fatalf("failed to seed controllers: %v", err)
	}

	controllers := []datafeed.VnasUser{
		{
			CID: 1234567,
			VnasController: datafeed.VnasController{
				ArtccID:  testArtccZOB,
				IsActive: true,
				VatsimData: datafeed.VnasVatsimData{
					Callsign: testCallsignZOBApp,
				},
			},
		},
	}

	err = ControllerFeed(ctx, controllers, mongoMock, client)
	if err != nil {
		t.Fatalf("ControllerFeed failed: %v", err)
	}

	// Verify neighbors was set to ZOB.
	neighbors, err := client.Get(ctx, "neighbors").Result()
	if err != nil || neighbors != "ZOB" {
		t.Errorf("Expected neighbors to be 'ZOB', got '%s'", neighbors)
	}
}

func TestControllerFeed_NeighborsCleared(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	mongoMock, _ := SetupControllerMongoMock(t, ControllerMockConfig{
		CID:             1234567,
		UserFirstName:   testUserFirstName,
		UserLastName:    testUserLastName,
		Position:        testCallsignORDGnd,
		CollectionATC:   testCollectionATC,
		CollectionUsers: testCollectionUsers,
		CollectionHours: testCollectionHours,
	})

	// Pre-seed: a ZAU controller (ORD_TWR) in the controllers set.
	_, err := helper.Mr.SAdd("controllers", testCallsignORDTwr)
	if err != nil {
		t.Fatalf("failed to seed controllers: %v", err)
	}

	// First feed: has a ZOB controller.
	controllers := []datafeed.VnasUser{
		{
			CID: 1234567,
			VnasController: datafeed.VnasController{
				ArtccID:  testArtccZOB,
				IsActive: true,
				VatsimData: datafeed.VnasVatsimData{
					Callsign: testCallsignZOBApp,
				},
			},
		},
	}

	err = ControllerFeed(ctx, controllers, mongoMock, client)
	if err != nil {
		t.Fatalf("ControllerFeed failed: %v", err)
	}

	// Verify neighbors was set to ZOB.
	if val, _ := client.Get(ctx, "neighbors").Result(); val != "ZOB" {
		t.Errorf("Expected neighbors to be 'ZOB', got '%s'", val)
	}

	// Second feed: only ZAU controller (no neighbor).
	controllers2 := []datafeed.VnasUser{
		{
			CID: 1234568,
			VnasController: datafeed.VnasController{
				ArtccID:  "ZAU",
				IsActive: true,
				VatsimData: datafeed.VnasVatsimData{
					Callsign: testCallsignORDGnd,
				},
			},
		},
	}

	err = ControllerFeed(ctx, controllers2, mongoMock, client)
	if err != nil {
		t.Fatalf("ControllerFeed failed: %v", err)
	}

	// Verify neighbors was cleared (Del was called).
	_, err = client.Get(ctx, "neighbors").Result()
	if !errors.Is(err, redis.Nil) {
		t.Errorf("Expected neighbors to be deleted, got: %v", err)
	}
}
