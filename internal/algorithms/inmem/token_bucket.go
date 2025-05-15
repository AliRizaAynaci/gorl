package algorithms

import (
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

// bucket holds the token count and last refill time for a single rate limit key.
type bucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex // Per-bucket locking
}

type tokenBucketLimiter struct {
	limit   int
	window  time.Duration
	buckets sync.Map // key: string, value: *bucket
}

func (t *tokenBucketLimiter) Allow(key string) (bool, error) {
	now := time.Now()

	var bkt *bucket
	val, ok := t.buckets.Load(key)
	if !ok {
		bkt = &bucket{
			tokens:     float64(t.limit),
			lastRefill: now,
		}
		t.buckets.Store(key, bkt)
	} else {
		bkt = val.(*bucket)
	}

	bkt.mu.Lock()
	defer bkt.mu.Unlock()

	elapsed := now.Sub(bkt.lastRefill).Seconds()
	refillRate := float64(t.limit) / t.window.Seconds()
	refilled := elapsed * refillRate

	bkt.tokens = minFloat(bkt.tokens+refilled, float64(t.limit))
	bkt.lastRefill = now

	if bkt.tokens >= 1 {
		bkt.tokens -= 1
		return true, nil
	}
	return false, nil
}

func NewTokenBucketLimiter(cfg core.Config) core.Limiter {
	return &tokenBucketLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
