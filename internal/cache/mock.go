package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type MockPipeline struct {
	redis.Pipeliner

	Published    map[string]string
	ExpiredKeys  []string
	DiffResponse []string
	RecordedCmds []string
}

func (m *MockPipeline) Publish(_ context.Context, channel string, message any) *redis.IntCmd {
	msgStr := fmt.Sprintf("%v", message)
	m.RecordedCmds = append(m.RecordedCmds, "PUBLISH:"+channel+":"+msgStr)

	if m.Published == nil {
		m.Published = make(map[string]string)
	}

	m.Published[channel] = msgStr

	return redis.NewIntResult(1, nil)
}

func (m *MockPipeline) HSet(_ context.Context, key string, values ...any) *redis.IntCmd {
	if len(values) >= 2 {
		if m.Published == nil {
			m.Published = make(map[string]string)
		}

		m.Published[key] = fmt.Sprint(values[1])
	}

	return redis.NewIntResult(1, nil)
}

func (m *MockPipeline) Expire(_ context.Context, _ string, _ time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(true, nil)
}

func (m *MockPipeline) Set(_ context.Context, key string, _ any, _ time.Duration) *redis.StatusCmd {
	m.RecordedCmds = append(m.RecordedCmds, "SET:"+key)
	return redis.NewStatusResult("OK", nil)
}

func (m *MockPipeline) Exec(_ context.Context) ([]redis.Cmder, error) {
	return nil, nil
}

func (m *MockPipeline) Rename(_ context.Context, _, _ string) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}

func (m *MockPipeline) Del(_ context.Context, keys ...string) *redis.IntCmd {
	m.ExpiredKeys = append(m.ExpiredKeys, keys...)
	return redis.NewIntResult(int64(len(keys)), nil)
}

func (m *MockPipeline) Cmds() []redis.Cmder {
	return nil
}

func (m *MockPipeline) Len() int {
	return len(m.RecordedCmds)
}

func (m *MockPipeline) Close() error { return nil }

func (m *MockPipeline) Discard() {}

type MockRedis struct {
	Pipe *MockPipeline
	Data map[string]string
}

func (m *MockRedis) Pipeline() redis.Pipeliner {
	if m.Pipe == nil {
		m.Pipe = &MockPipeline{
			Published: make(map[string]string),
		}
	}

	return m.Pipe
}

func (m *MockRedis) SAdd(_ context.Context, _ string, members ...any) *redis.IntCmd {
	return redis.NewIntResult(int64(len(members)), nil)
}

func (m *MockRedis) SDiff(_ context.Context, _ ...string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult(m.Pipe.DiffResponse, nil)
}

func (m *MockRedis) SMembers(_ context.Context, _ string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult(m.Pipe.DiffResponse, nil)
}

func (m *MockRedis) Expire(_ context.Context, _ string, _ time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(true, nil)
}

func (m *MockRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	return redis.NewIntResult(int64(len(keys)), nil)
}

func (m *MockRedis) Close() error {
	return nil
}

func (m *MockRedis) Ping(_ context.Context) *redis.StatusCmd {
	return redis.NewStatusResult("PONG", nil)
}

func (m *MockRedis) Publish(_ context.Context, channel string, message any) *redis.IntCmd {
	m.Pipe.Published[channel] = message.(string)
	return redis.NewIntResult(1, nil)
}

func (m *MockRedis) Rename(_ context.Context, _, _ string) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}

func (m *MockRedis) Get(_ context.Context, key string) *redis.StringCmd {
	val, exists := m.Data[key]
	if !exists {
		return redis.NewStringResult("", redis.Nil)
	}

	return redis.NewStringResult(val, nil)
}

func (m *MockRedis) Set(_ context.Context, _ string, _ any, _ time.Duration) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}

func (m *MockPipeline) LPush(_ context.Context, key string, values ...any) *redis.IntCmd {
	m.RecordedCmds = append(m.RecordedCmds, "LPUSH:"+key)
	return redis.NewIntResult(int64(len(values)), nil)
}
