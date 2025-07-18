// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"math"
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

// LeakyBucketLimiter implements the leaky bucket algorithm using minimal Storage API (Get/Set only).
// State is stored in two separate keys per user: water level and last leak timestamp.
type LeakyBucketLimiter struct {
	limit    int           // maximum water capacity
	window   time.Duration // leak window duration
	store    storage.Storage
	prefix   string     // key prefix, e.g. "gorl:lb"
	mu       sync.Mutex // ensure atomicity in-memory; Redis backend handles atomic ops
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
func (l *LeakyBucketLimiter) Allow(key string) (bool, error) {
	start := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UnixNano()
	waterKey := l.prefix + ":water:" + key
	leakKey := l.prefix + ":leak:" + key

	// Load current state
	waterVal, err := l.store.Get(waterKey)
	if allowed, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics); done {
		return allowed, retErr
	}

	waterLevel := int(waterVal)
	lastLeakVal, err := l.store.Get(leakKey)
	if allowed, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics); done {
		return allowed, retErr
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
			// Advance lastLeak based on consumed time
			lastLeak += int64(math.Floor(float64(leaked) / tokensPerNano))
		}
	}

	// Determine allowance and update water level
	allowed := waterLevel < l.limit
	if allowed {
		waterLevel++
	}

	// Persist updated state
	err = l.store.Set(waterKey, float64(waterLevel), l.window)
	if allowed, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics); done {
		return allowed, retErr
	}
	err = l.store.Set(leakKey, float64(lastLeak), l.window)
	if allowed, retErr, done := failOpenHandler(start, err, l.failOpen, l.metrics); done {
		return allowed, retErr
	}

	l.metrics.ObserveLatency(time.Since(start))
	if allowed {
		l.metrics.IncAllow()
	} else {
		l.metrics.IncDeny()
	}

	return allowed, nil
}
