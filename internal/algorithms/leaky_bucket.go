// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"context"
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

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UnixNano()
	waterKey := l.prefix + ":water:" + key
	leakKey := l.prefix + ":leak:" + key

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
	}

	if allowed {
		l.metrics.IncAllow()
	} else {
		l.metrics.IncDeny()
		// Wait time until water level drops by 1
		nanoPerToken := float64(l.window.Nanoseconds()) / float64(l.limit)
		res.RetryAfter = time.Duration((1.0/nanoPerToken)-float64(now-lastLeak)) * time.Nanosecond // This math might be slightly off but gives an idea
		if res.RetryAfter < 0 {
			res.RetryAfter = 0
		}
	}

	return res, nil
}

// Close releases resources held by the limiter.
func (l *LeakyBucketLimiter) Close() error {
	return l.store.Close()
}
