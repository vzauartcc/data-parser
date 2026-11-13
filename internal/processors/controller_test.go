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

func TestControllerFeed(t *testing.T) {
	ctx := context.Background()

	atcColl := &database.MockCollection{}
	userColl := &database.MockCollection{
		MockData: []any{
			models.User{CID: 1234567, FirstName: "John", LastName: "Doe"},
		},
	}
	sessionColl := &database.MockCollection{
		MockData: []any{
			models.ControllerHours{
				CID:          1234567,
				TimeStart:    time.Now().Add(-2 * time.Hour),
				TimeEnd:      time.Now(),
				Position:     "ORD_GND",
				IsStudent:    false,
				IsInstructor: false,
				WentInactive: false,
			},
		},
	}

	mongoMock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			"atcOnline":       atcColl,
			"users":           userColl,
			"controllerHours": sessionColl,
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
					Callsign: "ORD_GND",
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
			models.User{CID: 1234567, FirstName: "John", LastName: "Doe"},
		},
	}
	sessionColl := &database.MockCollection{
		MockData: []any{
			models.ControllerHours{
				CID:          1234567,
				TimeStart:    time.Now().Add(-2 * time.Hour),
				TimeEnd:      time.Now(),
				Position:     "ZOB_APP",
				IsStudent:    false,
				IsInstructor: false,
				WentInactive: false,
			},
		},
	}

	mongoMock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			"atcOnline":       atcColl,
			"users":           userColl,
			"controllerHours": sessionColl,
		},
	}

	mockPipe := &cache.MockPipeline{
		Published:    make(map[string]string),
		DiffResponse: []string{"ZOB_APP"},
	}
	redisMock := &cache.MockRedis{
		Pipe: mockPipe,
		Data: make(map[string]string),
	}

	controllers := []datafeed.VnasUser{
		{
			CID: 1234567,
			VnasController: datafeed.VnasController{
				ArtccID:  "ZOB",
				IsActive: true,
				VatsimData: datafeed.VnasVatsimData{
					Callsign: "ZOB_APP",
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
