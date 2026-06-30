package datafeed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchMetarFeed(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantErr        bool
	}{
		{
			name:           "Success - returns split metar strings",
			serverResponse: "KORD 112151Z 24008KT\nKMDW 112150Z 27010KT",
			serverStatus:   http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "Error - 500 Internal Server Error",
			serverResponse: "Internal Server Error",
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "Error - 404 Not Found",
			serverResponse: "Not Found",
			serverStatus:   http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			oldURL := metarURL
			metarURL = server.URL + "/"

			defer func() { metarURL = oldURL }()

			got, err := FetchMetarFeed(context.Background(), server.Client(), 0*time.Second)

			if (err != nil) != tt.wantErr {
				t.Errorf("FetchMetarFeed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) != 2 {
					t.Errorf("expected 2 metar strings, got %d", len(got))
				}

				if got[0] != "KORD 112151Z 24008KT" {
					t.Errorf("expected KORD metar first, got %s", got[0])
				}
			}
		})
	}
}
