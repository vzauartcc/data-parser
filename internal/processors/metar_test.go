package processors

import (
	"context"
	"testing"

	"github.com/vzauartcc/data-parser/internal/cache"
)

func TestMetarFeed_CustomMock(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		input          []string
		expectedCmds   []string
		expectedLength int
	}{
		{
			name:  "processes valid METARs and ignores short ones",
			input: []string{"KORD 121530Z", "BAD", "KMDW 121600Z"},
			expectedCmds: []string{
				"SET:METAR:KORD",
				"SET:METAR:KMDW",
			},
			expectedLength: 2,
		},
		{
			name:           "handles empty input",
			input:          []string{},
			expectedCmds:   nil,
			expectedLength: 0,
		},
		{
			name:           "skips all invalid input",
			input:          []string{"123", "A", ""},
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

func TestExtendMetarTTL(t *testing.T) {
	ctx := context.Background()

	mockData := map[string]string{
		"METAR:KORD": "data1",
		"METAR:KMDW": "data2",
		"OTHER:KEY":  "data3",
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
