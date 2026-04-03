package algorithms

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
)

type redisScriptRunner interface {
	EvalScript(ctx context.Context, name string, keys []string, args ...int64) ([]int64, error)
}

const (
	redisScriptSlidingWindow = "sliding_window"
	redisScriptTokenBucket   = "token_bucket"
	redisScriptLeakyBucket   = "leaky_bucket"
)

func clampDuration(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

func durationToMilliseconds(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	ms := d / time.Millisecond
	if ms <= 0 {
		return 1
	}
	return int64(ms)
}

func durationToMicros(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	us := d / time.Microsecond
	if us <= 0 {
		return 1
	}
	return int64(us)
}

func microsToDuration(us int64) time.Duration {
	if us <= 0 {
		return 0
	}
	return time.Duration(us) * time.Microsecond
}

func buildRedisScriptResult(limit int, values []int64) (core.Result, error) {
	if len(values) != 4 {
		return core.Result{}, fmt.Errorf("unexpected redis script result length: %d", len(values))
	}
	return core.Result{
		Allowed:    values[0] == 1,
		Limit:      limit,
		Remaining:  int(values[1]),
		Reset:      microsToDuration(values[2]),
		RetryAfter: microsToDuration(values[3]),
	}, nil
}

// failOpenHandler centralizes fail-open logic.
//   - start: timestamp when Allow began (for latency metrics)
//   - err: storage/algorithm error
//   - failOpen: cfg.FailOpen flag
//   - m: metrics collector
//
// Returns (result, done):
//   - done=true: caller should return immediately with (result, retErr)
//   - done=false: no error, continue normal flow
func failOpenHandler(start time.Time, err error, failOpen bool, m core.MetricsCollector, limit int) (core.Result, error, bool) {
	if err == nil {
		return core.Result{}, nil, false
	}
	if failOpen {
		m.ObserveLatency(time.Since(start))
		m.IncAllow()
		return core.Result{Allowed: true, Limit: limit}, nil, true
	}
	return core.Result{Allowed: false, Limit: limit}, err, true
}
