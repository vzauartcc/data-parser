package processors

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/datafeed"
	"github.com/vzauartcc/data-parser/internal/models"
)

const (
	testCallsignZOBApp  = "ZOB_APP"
	testCallsignORDGnd  = "ORD_GND"
	testUserFirstName   = "John"
	testUserLastName    = "Doe"
	testCollectionATC   = "atcOnline"
	testCollectionUsers = "users"
	testCollectionHours = "controllerHours"
	testArtccZOB        = "ZOB"
)

func TestControllerFeed(t *testing.T) {
	ctx := context.Background()

	atcColl := &database.MockCollection{}
	userColl := &database.MockCollection{
		MockData: []any{
			models.User{CID: 1234567, FirstName: testUserFirstName, LastName: testUserLastName},
		},
	}
	sessionColl := &database.MockCollection{
		MockData: []any{
			models.ControllerHours{
				CID:          1234567,
				TimeStart:    time.Now().Add(-2 * time.Hour),
				TimeEnd:      time.Now(),
				Position:     testCallsignORDGnd,
				IsStudent:    false,
				IsInstructor: false,
				WentInactive: false,
			},
		},
	}

	mongoMock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			testCollectionATC:   atcColl,
			testCollectionUsers: userColl,
			testCollectionHours: sessionColl,
		},
	}

	mockPipe := &cache.MockPipeline{
		Published:    make(map[string]string),
		DiffResponse: []string{"ORD_APP"},
	}
	redisMock := &cache.MockRedis{
		Pipe: mockPipe,
	}

	controllers := []datafeed.VnasUser{
		{
			CID: 1234567,
			VnasController: datafeed.VnasController{
				ArtccID:  "ZAU",
				IsActive: true,
				VatsimData: datafeed.VnasVatsimData{
					Callsign: testCallsignORDGnd,
				},
			},
		},
	}

	err := ControllerFeed(ctx, controllers, mongoMock, redisMock)
	if err != nil {
		t.Fatalf("ControllerFeed failed: %v", err)
	}

	foundDelete := slices.Contains(mockPipe.RecordedCmds, "PUBLISH:CONTROLLER:DELETE:ORD_APP")

	if !foundDelete {
		t.Errorf("Expected delete publish for ORD_APP, but it wasn't found in: %v", mockPipe.RecordedCmds)
	}

	if atcColl.DeletedFilter == nil {
		t.Error("Mongo DeleteMany was not called")
	}
}

func TestControllerFeed_Neighbors(t *testing.T) {
	ctx := context.Background()

	atcColl := &database.MockCollection{}
	userColl := &database.MockCollection{
		MockData: []any{
			models.User{CID: 1234567, FirstName: "John", LastName: testUserLastName},
		},
	}
	sessionColl := &database.MockCollection{
		MockData: []any{
			models.ControllerHours{
				CID:          1234567,
				TimeStart:    time.Now().Add(-2 * time.Hour),
				TimeEnd:      time.Now(),
				Position:     testCallsignZOBApp,
				IsStudent:    false,
				IsInstructor: false,
				WentInactive: false,
			},
		},
	}

	mongoMock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			testCollectionATC:   atcColl,
			testCollectionUsers: userColl,
			testCollectionHours: sessionColl,
		},
	}

	mockPipe := &cache.MockPipeline{
		Published:    make(map[string]string),
		DiffResponse: []string{testCallsignZOBApp},
	}
	redisMock := &cache.MockRedis{
		Pipe: mockPipe,
		Data: make(map[string]string),
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

	err := ControllerFeed(ctx, controllers, mongoMock, redisMock)
	if err != nil {
		t.Fatalf("ControllerFeed failed: %v", err)
	}

	neighbors, err := redisMock.Get(ctx, "neighbors").Result()
	if err != nil || neighbors != "ZOB" {
		t.Errorf("Expected neighbors to be 'ZOB', got '%s'", neighbors)
	}
}

func TestControllerFeed_NeighborsCleared(t *testing.T) {
	ctx := context.Background()

	atcColl := &database.MockCollection{}
	userColl := &database.MockCollection{
		MockData: []any{
			models.User{CID: 1234567, FirstName: testUserFirstName, LastName: testUserLastName},
		},
	}
	sessionColl := &database.MockCollection{
		MockData: []any{
			models.ControllerHours{
				CID:          1234567,
				TimeStart:    time.Now().Add(-2 * time.Hour),
				TimeEnd:      time.Now(),
				Position:     testCallsignZOBApp,
				IsStudent:    false,
				IsInstructor: false,
				WentInactive: false,
			},
		},
	}

	mongoMock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			testCollectionATC:   atcColl,
			testCollectionUsers: userColl,
			testCollectionHours: sessionColl,
		},
	}

	mockPipe := &cache.MockPipeline{
		Published:    make(map[string]string),
		DiffResponse: []string{testCallsignZOBApp},
	}
	redisMock := &cache.MockRedis{
		Pipe: mockPipe,
		Data: make(map[string]string),
	}

	// First feed: has a ZOB controller
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

	err := ControllerFeed(ctx, controllers, mongoMock, redisMock)
	if err != nil {
		t.Fatalf("ControllerFeed failed: %v", err)
	}

	// Verify neighbors was set to ZOB
	if redisMock.SetCalls["neighbors"] != "ZOB" {
		t.Errorf("Expected neighbors to be 'ZOB', got '%s'", redisMock.SetCalls["neighbors"])
	}

	// Second feed: only ZAU controller (no neighbor)
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

	err = ControllerFeed(ctx, controllers2, mongoMock, redisMock)
	if err != nil {
		t.Fatalf("ControllerFeed failed: %v", err)
	}

	// Verify neighbors was cleared (Del was called)
	found := slices.Contains(redisMock.DelCalls, "neighbors")
	if !found {
		t.Errorf("Expected neighbors to be deleted, DelCalls: %v", redisMock.DelCalls)
	}
}
