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
	limit  int
	window time.Duration
	store  storage.Storage
	prefix string
}

// NewFixedWindowLimiter creates a new FixedWindowLimiter.
func NewFixedWindowLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &FixedWindowLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  store,
		prefix: "gorl:fw",
	}
}

// Allow checks if a request with the given key is allowed under the fixed window policy.
func (f *FixedWindowLimiter) Allow(key string) (bool, error) {
	storageKey := f.prefix + ":" + key

	count, err := f.store.Incr(storageKey, f.window)
	if err != nil {
		return false, err
	}
	if count > float64(f.limit) {
		return false, nil
	}
	return true, nil
}
