package processors

import (
	"context"
	"testing"
	"time"

	metartafparser "github.com/ryansavara/go-metar-taf-parser"
)

const testStationKMDW = "KMDW"

func TestMetarFeed_CustomMock(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		input          []metartafparser.Metar
		expectedLength int
	}{
		{
			name: "processes valid METARs and ignores short ones",
			input: []metartafparser.Metar{
				{Station: "KORD", Message: "KORD 121530Z"},
				{Station: "BAD", Message: "BAD"},
				{Station: testStationKMDW, Message: testStationKMDW + " 121600Z"},
			},
			expectedLength: 3,
		},
		{
			name:           "handles empty input",
			input:          []metartafparser.Metar{},
			expectedLength: 0,
		},
		{
			name: "skips all invalid input",
			input: []metartafparser.Metar{
				{Station: "123", Message: "123"},
				{Station: "A", Message: "A"},
				{Station: "", Message: ""},
			},
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := NewRedisTestHelper(t)
			client := helper.Client

			err := MetarFeed(ctx, tt.input, client)
			if err != nil {
				t.Fatalf("MetarFeed failed unexpectedly: %v", err)
			}

			// Check the actual keys in Redis.
			keys, err := client.Keys(ctx, "METAR:*").Result()
			if err != nil {
				t.Fatalf("failed to get keys: %v", err)
			}

			if len(keys) != tt.expectedLength {
				t.Errorf("Expected %d METAR keys, got %d: %v", tt.expectedLength, len(keys), keys)
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
			ctx := context.Background()
			helper := NewRedisTestHelper(t)
			client := helper.Client

			err := MetarFeed(ctx, tt.input, client)
			if err != nil {
				t.Fatalf("MetarFeed failed: %v", err)
			}

			keys, err := client.Keys(ctx, "METAR:*").Result()
			if err != nil {
				t.Fatalf("failed to get keys: %v", err)
			}

			if len(keys) != tt.expectedLength {
				t.Errorf("Expected %d METAR keys, got %d: %v", tt.expectedLength, len(keys), keys)
			}
		})
	}
}

func TestExtendMetarTTL(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	// Pre-seed some METAR keys.
	err := helper.Mr.Set("METAR:KORD", "data1")
	if err != nil {
		t.Fatalf("failed to seed METAR:KORD: %v", err)
	}

	err = helper.Mr.Set("METAR:"+testStationKMDW, "data2")
	if err != nil {
		t.Fatalf("failed to seed METAR:%s: %v", testStationKMDW, err)
	}

	err = helper.Mr.Set("OTHER:KEY", "data3")
	if err != nil {
		t.Fatalf("failed to seed OTHER:KEY: %v", err)
	}

	err = ExtendMetarTTL(ctx, client)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify METAR keys have TTL set to 10 minutes.
	expectedTTL := 10 * time.Minute
	if got := helper.Mr.TTL("METAR:KORD"); got != expectedTTL {
		t.Errorf("expected 10m TTL for METAR:KORD, got %v", got)
	}

	if got := helper.Mr.TTL("METAR:" + testStationKMDW); got != expectedTTL {
		t.Errorf("expected 10m TTL for METAR:%s, got %v", testStationKMDW, got)
	}

	// Verify OTHER:KEY TTL was not changed (miniredis returns 0 for keys without TTL).
	if got := helper.Mr.TTL("OTHER:KEY"); got != 0 {
		t.Errorf("OTHER:KEY TTL changed: %v", got)
	}

	// Verify OTHER:KEY value was not touched.
	val, err := client.Get(ctx, "OTHER:KEY").Result()
	if err != nil {
		t.Errorf("OTHER:KEY should still exist: %v", err)
	}

	if val != "data3" {
		t.Errorf("OTHER:KEY value changed: %s", val)
	}
}

func TestExtendMetarTTL_MultiPageScan(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	// Seed 150 METAR keys (batch size is 100 in production, so this tests multi-page scan).
	for i := range 150 {
		key := "METAR:K" + string(rune('A'+i%26)) + string(rune('0'+i/26%10))

		err := helper.Mr.Set(key, "data")
		if err != nil {
			t.Fatalf("failed to seed %s: %v", key, err)
		}
	}

	err := ExtendMetarTTL(ctx, client)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify all 150 keys have TTL set to 10 minutes.
	expectedTTL := 10 * time.Minute

	keys, err := client.Keys(ctx, "METAR:*").Result()
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}

	if len(keys) != 150 {
		t.Fatalf("expected 150 keys, got %d", len(keys))
	}

	for _, key := range keys {
		if got := helper.Mr.TTL(key); got != expectedTTL {
			t.Errorf("key %s: expected 10m TTL, got %v", key, got)
		}
	}
}
