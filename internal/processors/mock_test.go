package processors

import (
	"testing"
	"time"

	"github.com/vzauartcc/data-parser/internal/database"
	"github.com/vzauartcc/data-parser/internal/models"
)

type ControllerMockConfig struct {
	CID             int
	UserFirstName   string
	UserLastName    string
	Position        string
	CollectionATC   string
	CollectionUsers string
	CollectionHours string
}

type PilotMockConfig struct {
	CollectionPilots string
}

// SetupControllerMongoMock creates a MongoDB mock for controller tests.
func SetupControllerMongoMock(
	t *testing.T,
	cfg ControllerMockConfig,
) (*database.MockMongoDatabase, *database.MockCollection) {
	t.Helper()

	now := time.Now()
	user := models.User{CID: cfg.CID, FirstName: cfg.UserFirstName, LastName: cfg.UserLastName}
	session := models.ControllerHours{
		CID:          cfg.CID,
		TimeStart:    now.Add(-2 * time.Hour),
		TimeEnd:      now,
		Position:     cfg.Position,
		IsStudent:    false,
		IsInstructor: false,
		WentInactive: false,
	}

	mock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			cfg.CollectionATC:   {},
			cfg.CollectionUsers: {MockData: []any{user}},
			cfg.CollectionHours: {MockData: []any{session}},
		},
	}

	return mock, mock.Collections[cfg.CollectionATC]
}

// SetupPilotMongoMock creates a MongoDB mock for pilot tests.
func SetupPilotMongoMock(
	t *testing.T,
	cfg PilotMockConfig,
) (*database.MockMongoDatabase, *database.MockCollection) {
	t.Helper()

	mock := &database.MockMongoDatabase{
		Collections: map[string]*database.MockCollection{
			cfg.CollectionPilots: {},
		},
	}

	return mock, mock.Collections[cfg.CollectionPilots]
}
