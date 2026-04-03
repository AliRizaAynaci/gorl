// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage"
)

// FixedWindowLimiter implements the fixed window rate limiting algorithm.
// It allows a certain number of requests within a fixed time window.
type FixedWindowLimiter struct {
	limit    int
	window   time.Duration
	store    storage.Storage
	prefix   string
	metrics  core.MetricsCollector
	failOpen bool
}

// NewFixedWindowLimiter creates a new FixedWindowLimiter.
func NewFixedWindowLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &FixedWindowLimiter{
		limit:    cfg.Limit,
		window:   cfg.Window,
		store:    store,
		prefix:   "gorl:fw",
		metrics:  cfg.Metrics,
		failOpen: cfg.FailOpen,
	}
}

// Allow checks if a request with the given key is allowed under the fixed window policy.
func (f *FixedWindowLimiter) Allow(ctx context.Context, key string) (core.Result, error) {
	start := time.Now()
	bucket := start.UnixNano() / int64(f.window)
	storageKey := fmt.Sprintf("%s:%s:%d", f.prefix, key, bucket)

	count, err := f.store.Incr(ctx, storageKey, f.window)
	if res, retErr, done := failOpenHandler(start, err, f.failOpen, f.metrics, f.limit); done {
		return res, retErr
	}

	nextBucketStart := time.Unix(0, (bucket+1)*int64(f.window))
	reset := clampDuration(time.Until(nextBucketStart))
	remaining := f.limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	allowed := count <= float64(f.limit)
	f.metrics.ObserveLatency(time.Since(start))

	res := core.Result{
		Allowed:   allowed,
		Limit:     f.limit,
		Remaining: remaining,
		Reset:     reset,
	}

	if allowed {
		f.metrics.IncAllow()
	} else {
		f.metrics.IncDeny()
		res.RetryAfter = reset
	}
	return res, nil
}

// Close releases resources held by the limiter.
func (f *FixedWindowLimiter) Close() error {
	return f.store.Close()
}
