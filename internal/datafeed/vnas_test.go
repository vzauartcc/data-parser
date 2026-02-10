package datafeed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchVnasFeed(t *testing.T) {
	now := time.Now().Round(time.Second)

	mockResponse := ControllerFeed{
		UpdatedAt: now,
		Controllers: []VnasController{
			{
				ArtccID: "ZAU",
				Role:    "CTR",
				VatsimData: VnasVatsimData{
					CID:      "1234567",
					Callsign: "CHI_CTR",
				},
			},
			{
				ArtccID: "ZNY",
				VatsimData: VnasVatsimData{
					CID: "invalid",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	oldURL := vNasURL
	vNasURL = server.URL

	defer func() { vNasURL = oldURL }()

	result, err := FetchVnasFeed(context.Background())
	if err != nil {
		t.Fatalf("FetchVnasFeed failed: %v", err)
	}

	if !result.UpdatedAt.Equal(now) {
		t.Errorf("expected time %v, got %v", now, result.UpdatedAt)
	}

	if len(result.Controllers) != 1 {
		t.Fatalf("expected 1 controller after filtering, got %d", len(result.Controllers))
	}

	if result.Controllers[0].CID != 1234567 {
		t.Errorf("expected CID as int 1234567, got %d", result.Controllers[0].CID)
	}

	if result.Controllers[0].ArtccID != "ZAU" {
		t.Errorf("embedded VnasController data missing, got %s", result.Controllers[0].ArtccID)
	}
}

func TestFetchVnasFeed_Failures(t *testing.T) {
	t.Run("Server Error 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		oldURL := vNasURL
		vNasURL = server.URL

		defer func() { vNasURL = oldURL }()

		_, err := FetchVnasFeed(context.Background())
		if err == nil {
			t.Error("expected error on 500 status code, but got nil")
		}
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{ "controllers": [ { "vatsimData": { "cid": `))
		}))
		defer server.Close()

		oldURL := vNasURL
		vNasURL = server.URL

		defer func() { vNasURL = oldURL }()

		_, err := FetchVnasFeed(context.Background())
		if err == nil {
			t.Error("expected error on malformed JSON, but got nil")
		}
	})

	t.Run("Context Timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		oldURL := vNasURL
		vNasURL = server.URL

		defer func() { vNasURL = oldURL }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err := FetchVnasFeed(ctx)
		if err == nil {
			t.Error("expected error due to context timeout, but got nil")
		}
	})
}

func TestGetRating(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"Observer", 1},
		{"Student3", 4},
		{"Controller2", 6},
		{"Supervisor", 11},
		{"Administrator", 12},
		{"Unknown", 0},
		{"", 0},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := GetRating(tc.input)
			if got != tc.expected {
				t.Errorf("GetRating(%q) = %d; want %d", tc.input, got, tc.expected)
			}
		})
	}
}
