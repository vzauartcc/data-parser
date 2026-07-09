package datafeed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	metartafparser "github.com/ryansavara/go-metar-taf-parser"
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
					t.Errorf("expected 2 metar structs, got %d", len(got))
				}

				expected, err := metartafparser.ParseMetar("KORD 112151Z 24008KT", nil)
				if err != nil {
					t.Fatalf("failed to parse expected metar: %v", err)
				}

				if got[0].Station != expected.Station {
					t.Errorf("expected station %s, got %s", expected.Station, got[0].Station)
				}

				if got[0].Message != expected.Message {
					t.Errorf("expected message %s, got %s", expected.Message, got[0].Message)
				}
			}
		})
	}
}
