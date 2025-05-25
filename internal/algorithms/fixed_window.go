// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

// FixedWindowLimiter implements the fixed window rate limiting algorithm.
// It allows a certain number of requests within a fixed time window.
type FixedWindowLimiter struct {
	limit   int
	window  time.Duration
	store   storage.Storage
	prefix  string
	metrics core.MetricsCollector
}

// NewFixedWindowLimiter creates a new FixedWindowLimiter.
func NewFixedWindowLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &FixedWindowLimiter{
		limit:   cfg.Limit,
		window:  cfg.Window,
		store:   store,
		prefix:  "gorl:fw",
		metrics: cfg.Metrics,
	}
}

// Allow checks if a request with the given key is allowed under the fixed window policy.
func (f *FixedWindowLimiter) Allow(key string) (bool, error) {
	start := time.Now()
	storageKey := f.prefix + ":" + key

	count, err := f.store.Incr(storageKey, f.window)
	if err != nil {
		return false, err
	}
	allowed := count <= float64(f.limit)
	f.metrics.ObserveLatency(time.Since(start))

	if allowed {
		f.metrics.IncAllow()
	} else {
		f.metrics.IncDeny()
	}
	return allowed, nil
}
