package processors

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// RedisTestHelper holds a miniredis instance and a redis client.
type RedisTestHelper struct {
	Client *redis.Client
	Mr     *miniredis.Miniredis
}

// NewRedisTestHelper creates a new miniredis instance and redis client for testing.
func NewRedisTestHelper(t *testing.T) *RedisTestHelper {
	t.Helper()
	miniredisInstance := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: miniredisInstance.Addr(),
	})

	t.Cleanup(func() {
		_ = client.Close()
	})

	return &RedisTestHelper{
		Client: client,
		Mr:     miniredisInstance,
	}
}

// waitForChannel waits for a single message on the given pubsub channel.
// Uses a 5s timeout (configurable via parameter if needed in future).
func waitForChannel(
	t *testing.T,
	sub *redis.PubSub,
	channel string,
) string {
	t.Helper()

	ch := sub.Channel()
	select {
	case msg := <-ch:
		if msg.Channel != channel {
			t.Fatalf("unexpected channel: got %s, want %s", msg.Channel, channel)
		}

		return msg.Payload
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for publish on %s", channel)

		return ""
	}
}
