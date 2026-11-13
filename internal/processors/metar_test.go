package processors

import (
	"context"
	"testing"

	metartafparser "github.com/ryansavara/go-metar-taf-parser"
	"github.com/vzauartcc/data-parser/internal/cache"
)

const testStationKMDW = "KMDW"

func TestMetarFeed_CustomMock(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		input          []metartafparser.Metar
		expectedCmds   []string
		expectedLength int
	}{
		{
			name: "processes valid METARs and ignores short ones",
			input: []metartafparser.Metar{
				{Station: "KORD", Message: "KORD 121530Z"},
				{Station: "BAD", Message: "BAD"},
				{Station: testStationKMDW, Message: testStationKMDW + " 121600Z"},
			},
			expectedCmds: []string{
				"SET:METAR:KORD",
				"SET:METAR:BAD",
				"SET:METAR:" + testStationKMDW,
			},
			expectedLength: 3,
		},
		{
			name:           "handles empty input",
			input:          []metartafparser.Metar{},
			expectedCmds:   nil,
			expectedLength: 0,
		},
		{
			name: "skips all invalid input",
			input: []metartafparser.Metar{
				{Station: "123", Message: "123"},
				{Station: "A", Message: "A"},
				{Station: "", Message: ""},
			},
			expectedCmds:   nil,
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPipe := &cache.MockPipeline{
				Published: make(map[string]string),
			}
			mockRedis := &cache.MockRedis{
				Pipe: mockPipe,
				Data: make(map[string]string),
			}

			err := MetarFeed(ctx, tt.input, mockRedis)
			if err != nil {
				t.Fatalf("MetarFeed failed unexpectedly: %v", err)
			}

			if mockPipe.Len() != tt.expectedLength {
				t.Errorf("Expected %d commands, got %d", tt.expectedLength, mockPipe.Len())
			}

			for i, cmd := range tt.expectedCmds {
				if mockPipe.RecordedCmds[i] != cmd {
					t.Errorf("At index %d: expected command %s, got %s", i, cmd, mockPipe.RecordedCmds[i])
				}
			}
		})
	}
}

func TestMetarFeed_StationCodeLength(t *testing.T) {
	tests := []struct {
		name           string
		input          []metartafparser.Metar
		expectedLength int
	}{
		{
			name:           "3-char station code should be processed",
			input:          []metartafparser.Metar{{Station: "PHI", Message: "PHI 121530Z"}},
			expectedLength: 1,
		},
		{
			name:           "4-char station code",
			input:          []metartafparser.Metar{{Station: testStationKMDW, Message: testStationKMDW + " 121600Z"}},
			expectedLength: 1,
		},
		{
			name:           "too-short station code should be skipped",
			input:          []metartafparser.Metar{{Station: "AB", Message: "AB"}},
			expectedLength: 0,
		},
		{
			name:           "numeric station code should be skipped",
			input:          []metartafparser.Metar{{Station: "123", Message: "123"}},
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPipe := &cache.MockPipeline{
				Published: make(map[string]string),
			}
			mockRedis := &cache.MockRedis{
				Pipe: mockPipe,
				Data: make(map[string]string),
			}

			err := MetarFeed(context.Background(), tt.input, mockRedis)
			if err != nil {
				t.Fatalf("MetarFeed failed: %v", err)
			}

			if mockPipe.Len() != tt.expectedLength {
				t.Errorf("Expected %d commands, got %d", tt.expectedLength, mockPipe.Len())
			}
		})
	}
}

func TestExtendMetarTTL(t *testing.T) {
	ctx := context.Background()

	mockData := map[string]string{
		"METAR:KORD":               "data1",
		"METAR:" + testStationKMDW: "data2",
		"OTHER:KEY":                "data3",
	}

	mockPipe := &cache.MockPipeline{
		Published: make(map[string]string),
	}

	client := &cache.MockRedis{
		Data: mockData,
		Pipe: mockPipe,
	}

	err := ExtendMetarTTL(ctx, client)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedKeys := 2
	if len(mockPipe.ExpiredKeys) != expectedKeys {
		t.Errorf("expected %d expired keys, got %d", expectedKeys, len(mockPipe.ExpiredKeys))
	}
}
