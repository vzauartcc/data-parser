package datafeed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchVatsimDatafeed(t *testing.T) {
	mockFeed := VatsimFeed{
		Pilots: []VatsimPilot{
			{
				CID:         123456,
				Name:        "John Doe",
				Callsign:    "DAL123",
				Latitude:    33.64,
				Longitude:   -84.42,
				Altitude:    35000,
				Groundspeed: 450,
				FlightPlan: &VatsimFlightPlan{
					AircraftFAA: "B738",
					Departure:   "KORD",
					Arrival:     "KJFK",
					Route:       "DCT",
				},
			},
		},
		ATISs: []VatsimATIS{
			{
				Callsign: "KMKE_ATIS",
				ATISCode: "R",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(mockFeed)
	}))
	defer server.Close()

	originalURL := vatsimURL
	vatsimURL = server.URL

	defer func() { vatsimURL = originalURL }()

	ctx := context.Background()

	feed, err := FetchVatsimDatafeed(ctx, server.Client())
	if err != nil {
		t.Fatalf("FetchVatsimDatafeed failed: %v", err)
	}

	if len(feed.Pilots) != 1 {
		t.Fatalf("expected 1 pilot, got %d", len(feed.Pilots))
	}

	pilot := feed.Pilots[0]
	if pilot.Callsign != "DAL123" {
		t.Errorf("expected callsign DAL123, got %s", pilot.Callsign)
	}

	if pilot.FlightPlan == nil {
		t.Fatal("expected flight plan to be populated, got nil")
	}

	if pilot.FlightPlan.Departure != "KORD" {
		t.Errorf("expected departure KORD, got %s", pilot.FlightPlan.Departure)
	}

	if len(feed.ATISs) != 1 || feed.ATISs[0].ATISCode != "R" {
		t.Errorf("ATIS data mismatch: %+v", feed.ATISs)
	}
}

func TestFetchVatsimDatafeed_Failures(t *testing.T) {
	t.Run("Server Returns 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		oldURL := vatsimURL
		vatsimURL = server.URL

		defer func() { vatsimURL = oldURL }()

		feed, err := FetchVatsimDatafeed(context.Background(), server.Client())
		if err == nil {
			t.Error("expected error on 404 Status Not Found, but got nil")
		}

		if len(feed.Pilots) != 0 {
			t.Errorf("expected empty Pilots slice on error, got %d", len(feed.Pilots))
		}
	})

	t.Run("Malformed JSON Structure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"pilots": "this-should-be-an-array-not-a-string"}`))
		}))
		defer server.Close()

		oldURL := vatsimURL
		vatsimURL = server.URL

		defer func() { vatsimURL = oldURL }()

		_, err := FetchVatsimDatafeed(context.Background(), server.Client())
		if err == nil {
			t.Error("expected JSON unmarshal error, but got nil")
		}
	})

	t.Run("Context Deadline Exceeded", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(100 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				return
			}
		}))
		defer server.Close()

		oldURL := vatsimURL
		vatsimURL = server.URL

		defer func() { vatsimURL = oldURL }()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := FetchVatsimDatafeed(ctx, server.Client())
		if err == nil {
			t.Error("expected error for context timeout, but got nil")
		}
	})
}
