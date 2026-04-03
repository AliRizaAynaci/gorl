// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage"
)

// LeakyBucketLimiter implements the leaky bucket algorithm using minimal Storage API (Get/Set only).
// State is stored in two separate keys per user: water level and last leak timestamp.
type LeakyBucketLimiter struct {
	limit    int
	window   time.Duration
	store    storage.Storage
	prefix   string
	mu       sync.Mutex
	metrics  core.MetricsCollector
	failOpen bool
}

// NewLeakyBucketLimiter constructs a new LeakyBucketLimiter.
func NewLeakyBucketLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &LeakyBucketLimiter{
		limit:    cfg.Limit,
		window:   cfg.Window,
		store:    store,
		prefix:   "gorl:lb",
		metrics:  cfg.Metrics,
		failOpen: cfg.FailOpen,
	}
}

// Allow checks and updates water level, allowing requests at a steady rate.
func (l *LeakyBucketLimiter) Allow(ctx context.Context, key string) (core.Result, error) {
	start := time.Now()
	if runner, ok := l.store.(redisScriptRunner); ok {
		return l.allowRedis(ctx, start, runner, key)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	return l.allowGeneric(ctx, start, key)
}

func (l *LeakyBucketLimiter) allowGeneric(ctx context.Context, start time.Time, key string) (core.Result, error) {
	now := time.Now().UnixNano()
	waterKey := fmt.Sprintf("%s:{%s}:water", l.prefix, key)
	leakKey := fmt.Sprintf("%s:{%s}:leak", l.prefix, key)

	// Load current state
	waterVal, err := l.store.Get(ctx, waterKey)
	if res, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics, l.limit); done {
		return res, retErr
	}

	waterLevel := int(waterVal)
	lastLeakVal, err := l.store.Get(ctx, leakKey)
	if res, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics, l.limit); done {
		return res, retErr
	}
	lastLeak := int64(lastLeakVal)

	// Initialize if first run
	if lastLeak == 0 {
		waterLevel = 0
		lastLeak = now
	} else {
		// Compute leaked tokens since last leak
		elapsed := now - lastLeak
		tokensPerNano := float64(l.limit) / float64(l.window.Nanoseconds())
		leaked := int64(math.Floor(float64(elapsed) * tokensPerNano))
		if leaked > 0 {
			waterLevel -= int(leaked)
			if waterLevel < 0 {
				waterLevel = 0
			}
			lastLeak += int64(math.Floor(float64(leaked) / tokensPerNano))
		}
	}

	// Determine allowance and update water level
	allowed := waterLevel < l.limit
	if allowed {
		waterLevel++
	}

	nanoPerToken := float64(l.window.Nanoseconds()) / float64(l.limit)
	elapsedSinceLeak := float64(now - lastLeak)
	nextLeak := clampDuration(time.Duration(nanoPerToken-elapsedSinceLeak) * time.Nanosecond)
	reset := time.Duration(0)
	if waterLevel > 0 {
		reset = clampDuration(time.Duration(float64(waterLevel)*nanoPerToken-elapsedSinceLeak) * time.Nanosecond)
	}

	// Persist updated state
	err = l.store.Set(ctx, waterKey, float64(waterLevel), l.window)
	if res, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics, l.limit); done {
		return res, retErr
	}
	err = l.store.Set(ctx, leakKey, float64(lastLeak), l.window)
	if res, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics, l.limit); done {
		return res, retErr
	}

	l.metrics.ObserveLatency(time.Since(start))

	res := core.Result{
		Allowed:   allowed,
		Limit:     l.limit,
		Remaining: l.limit - waterLevel,
		Reset:     reset,
	}

	if allowed {
		l.metrics.IncAllow()
	} else {
		l.metrics.IncDeny()
		res.RetryAfter = nextLeak
	}

	return res, nil
}

func (l *LeakyBucketLimiter) allowRedis(ctx context.Context, start time.Time, runner redisScriptRunner, key string) (core.Result, error) {
	keys := []string{
		fmt.Sprintf("%s:{%s}:water", l.prefix, key),
		fmt.Sprintf("%s:{%s}:leak", l.prefix, key),
	}

	values, err := runner.EvalScript(
		ctx,
		redisScriptLeakyBucket,
		keys,
		int64(l.limit),
		time.Now().UnixMicro(),
		durationToMicros(l.window),
		durationToMilliseconds(l.window),
	)
	if res, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics, l.limit); done {
		return res, retErr
	}

	res, err := buildRedisScriptResult(l.limit, values)
	if res2, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics, l.limit); done {
		return res2, retErr
	}

	l.metrics.ObserveLatency(time.Since(start))
	if res.Allowed {
		l.metrics.IncAllow()
	} else {
		l.metrics.IncDeny()
	}

	return res, nil
}

// Close releases resources held by the limiter.
func (l *LeakyBucketLimiter) Close() error {
	return l.store.Close()
}
