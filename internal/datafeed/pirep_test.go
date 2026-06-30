package datafeed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func ptr[T any](v T) *T {
	return &v
}

func TestFetchPirepFeed(t *testing.T) {
	mockPireps := []Pirep{
		{
			ObservationTime: 1700000000,
			AircraftType:    "C172",
			Latitude:        42.21,
			Longitude:       -89.09,
			FlightLevel:     30,
			Clouds: &[]Cloud{
				{Cover: "OVC", Base: 2000, Tops: 4500},
			},
			Visibility:     ptr(10),
			RawObservation: "KRFD UA /OV KRFD/TM 1200/FL030/TP C172/SK OVC020-TOP045/WX FV10SM",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(mockPireps)
		if err != nil {
			t.Errorf("failed to encode mock data: %v", err)
		}
	}))
	defer server.Close()

	originalURL := pirepURL
	pirepURL = server.URL

	defer func() { pirepURL = originalURL }()

	ctx := context.Background()

	results, err := FetchPirepFeed(ctx, server.Client())
	if err != nil {
		t.Fatalf("FetchPirepFeed returned an error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 PIREP, got %d", len(results))
	}

	res := results[0]

	if res.AircraftType != "C172" {
		t.Errorf("expected acType C172, got %s", res.AircraftType)
	}

	if res.Visibility == nil || *res.Visibility != 10 {
		t.Errorf("expected visibility 10, got %v", res.Visibility)
	}

	if res.Clouds == nil || len(*res.Clouds) == 0 || (*res.Clouds)[0].Cover != "OVC" {
		t.Error("cloud data was not decoded correctly")
	}
}

func TestFetchPirepFeed_Failures(t *testing.T) {
	t.Run("API Server Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		oldURL := pirepURL
		pirepURL = server.URL

		defer func() { pirepURL = oldURL }()

		_, err := FetchPirepFeed(context.Background(), server.Client())
		if err == nil {
			t.Error("expected an error when the server returns 500, but got nil")
		}
	})

	t.Run("Malformed JSON Response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"obsTime": 123, "acType": "C172", "clouds": `))
		}))
		defer server.Close()

		oldURL := pirepURL
		pirepURL = server.URL

		defer func() { pirepURL = oldURL }()

		_, err := FetchPirepFeed(context.Background(), server.Client())
		if err == nil {
			t.Error("expected a JSON decoding error, but got nil")
		}
	})

	t.Run("Request Timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		oldURL := pirepURL
		pirepURL = server.URL

		defer func() { pirepURL = oldURL }()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := FetchPirepFeed(ctx, server.Client())
		if err == nil {
			t.Error("expected context deadline exceeded error, but got nil")
		}
	})
}
