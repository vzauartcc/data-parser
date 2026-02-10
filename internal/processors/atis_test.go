package processors

import (
	"context"
	"testing"

	"github.com/vzauartcc/data-parser/internal/cache"
	"github.com/vzauartcc/data-parser/internal/datafeed"
)

func TestAtisFeed(t *testing.T) {
	mockPipe := &cache.MockPipeline{
		Published:    make(map[string]string),
		DiffResponse: []string{"KMDW"},
	}
	client := &cache.MockRedis{Pipe: mockPipe}

	data := []datafeed.VatsimATIS{
		{Callsign: "KORD_ATIS", ATISCode: "A"},
	}

	err := AtisFeed(context.Background(), data, client)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mockPipe.Published["ATIS:KORD"] != "A" {
		t.Errorf("expected KORD code A, got %s", mockPipe.Published["ATIS:KORD"])
	}

	foundDelete := false

	for _, k := range mockPipe.ExpiredKeys {
		if k == "ATIS:KMDW" {
			foundDelete = true
		}
	}

	if !foundDelete {
		t.Error("expected KMDW to be expired/cleaned up")
	}
}
