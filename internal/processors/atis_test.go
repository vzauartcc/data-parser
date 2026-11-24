package processors

import (
	"context"
	"testing"
	"time"

	"github.com/vzauartcc/data-parser/internal/datafeed"
)

const (
	testAirportKMDW      = "KMDW"
	testAirportThreeChar = "05C"
)

func TestAtisFeed(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	// Pre-seed: KMDW is in the main set (simulating previously active).
	_, err := helper.Mr.SAdd("atis", testAirportKMDW)
	if err != nil {
		t.Fatalf("failed to seed atis: %v", err)
	}

	// Pre-seed ATIS:KMDW hash so we can verify it gets deleted.
	helper.Mr.HSet("ATIS:"+testAirportKMDW, "code", "A")

	// Pre-seed ATIS:KORD key so EXPIRE in AtisFeed sets TTL on an existing key.
	helper.Mr.HSet("ATIS:KORD", "code", "B")

	data := []datafeed.VatsimATIS{
		{Callsign: "KORD_ATIS", ATISCode: "A"},
	}

	// Subscribe BEFORE calling AtisFeed to catch the publish.
	sub := client.Subscribe(ctx, "ATIS:KORD")

	err = AtisFeed(ctx, data, client)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify ATIS:KORD was published.
	payload := waitForChannel(t, sub, "ATIS:KORD")
	if payload != "A" {
		t.Errorf("expected ATIS:KORD publish payload 'A', got %s", payload)
	}

	sub.Close()

	// Verify KMDW was cleaned up (deleted from main set).
	members, err := client.SMembers(ctx, "atis").Result()
	if err != nil {
		t.Fatalf("failed to get atis members: %v", err)
	}

	for _, m := range members {
		if m == testAirportKMDW {
			t.Errorf("expected KMDW to be removed from atis set, but it's still there")
		}
	}

	// Verify ATIS:KMDW hash was deleted (the cleanup pipeline deletes it).
	if helper.Mr.Exists("ATIS:" + testAirportKMDW) {
		t.Errorf("expected ATIS:KMDW hash to be deleted")
	}

	// Verify ATIS:KORD key has 65s TTL (set by pipe.Expire).
	if got := helper.Mr.TTL("ATIS:KORD"); got != 65*time.Second {
		t.Errorf("expected 65s TTL for ATIS:KORD, got %v", got)
	}
}

func TestAtisFeed_ExpiredAirports(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	// Pre-seed: KMDW is in the main set.
	_, err := helper.Mr.SAdd("atis", testAirportKMDW)
	if err != nil {
		t.Fatalf("failed to seed atis: %v", err)
	}

	// Pre-seed ATIS:KMDW hash.
	helper.Mr.HSet("ATIS:"+testAirportKMDW, "code", "A")

	data := []datafeed.VatsimATIS{
		{Callsign: "KORD_ATIS", ATISCode: "A"},
	}

	// Subscribe to ATIS:DELETE BEFORE calling AtisFeed.
	sub := client.Subscribe(ctx, "ATIS:DELETE")

	err = AtisFeed(ctx, data, client)
	if err != nil {
		t.Fatalf("AtisFeed failed: %v", err)
	}

	// Verify ATIS:DELETE was published for the expired airport.
	payload := waitForChannel(t, sub, "ATIS:DELETE")
	if payload != testAirportKMDW {
		t.Errorf("expected ATIS:DELETE payload '%s', got '%s'", testAirportKMDW, payload)
	}

	sub.Close()

	// Verify KMDW was cleaned up from main set.
	members, err := client.SMembers(ctx, "atis").Result()
	if err != nil {
		t.Fatalf("failed to get atis members: %v", err)
	}

	for _, m := range members {
		if m == testAirportKMDW {
			t.Errorf("Expected KMDW to be identified as expired and removed")
		}
	}

	// Verify ATIS:KMDW hash was deleted.
	if helper.Mr.Exists("ATIS:" + testAirportKMDW) {
		t.Errorf("expected ATIS:KMDW hash to be deleted")
	}
}

func TestAtisFeed_ThreeCharAirport(t *testing.T) {
	ctx := context.Background()
	helper := NewRedisTestHelper(t)
	client := helper.Client

	// Pre-seed ATIS:05C key so EXPIRE in AtisFeed sets TTL.
	helper.Mr.HSet("ATIS:"+testAirportThreeChar, "code", "B")

	data := []datafeed.VatsimATIS{
		{Callsign: testAirportThreeChar + "_ATIS", ATISCode: "A"},
	}

	// Subscribe BEFORE calling AtisFeed to catch the publish.
	sub := client.Subscribe(ctx, "ATIS:"+testAirportThreeChar)

	err := AtisFeed(ctx, data, client)
	if err != nil {
		t.Fatalf("AtisFeed failed: %v", err)
	}

	// Verify ATIS:05C was published.
	payload := waitForChannel(t, sub, "ATIS:"+testAirportThreeChar)
	if payload != "A" {
		t.Errorf("Expected ATIS:%s to be published with value 'A', got %v", testAirportThreeChar, payload)
	}

	sub.Close()

	// Verify ATIS:05C key has 65s TTL.
	if got := helper.Mr.TTL("ATIS:" + testAirportThreeChar); got != 65*time.Second {
		t.Errorf("expected 65s TTL for ATIS:%s, got %v", testAirportThreeChar, got)
	}
}
