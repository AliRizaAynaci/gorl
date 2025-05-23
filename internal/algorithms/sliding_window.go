// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

// SlidingWindowLimiter implements an approximate sliding window algorithm using minimal Storage API (Get/Set/Incr).
// It keeps two counters (current and previous window) and a timestamp of the window start.
type SlidingWindowLimiter struct {
	limit  int           // maximum requests per window
	window time.Duration // window duration
	store  storage.Storage
	prefix string // key prefix, e.g. "gorl:sw"
}

// NewSlidingWindowLimiter constructs a new SlidingWindowLimiter.
func NewSlidingWindowLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &SlidingWindowLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  store,
		prefix: "gorl:sw",
	}
}

// Allow checks whether a request is allowed under a sliding window.
func (s *SlidingWindowLimiter) Allow(key string) (bool, error) {
	// Current timestamp in nanoseconds
	now := time.Now().UnixNano()

	// Define storage keys
	tsKey := s.prefix + ":ts:" + key     // window start timestamp
	currKey := s.prefix + ":curr:" + key // count in current window
	prevKey := s.prefix + ":prev:" + key // count in previous window

	// Load last window start
	tsVal, err := s.store.Get(tsKey)
	if err != nil {
		return false, err
	}

	var windowStart int64
	if tsVal == 0 {
		// First request: initialize
		windowStart = now
		_ = s.store.Set(tsKey, float64(windowStart), s.window)
		_ = s.store.Set(currKey, 0, s.window)
		_ = s.store.Set(prevKey, 0, s.window)
	} else {
		windowStart = int64(tsVal)
		elapsed := now - windowStart

		if elapsed >= int64(s.window) {
			// Move window forward by number of intervals passed
			intervals := elapsed / int64(s.window)

			// Shift current to previous
			currCount, err := s.store.Get(currKey)
			if err != nil {
				return false, err
			}
			_ = s.store.Set(prevKey, currCount, s.window)

			// Reset current counter
			_ = s.store.Set(currKey, 0, s.window)

			// Advance windowStart
			windowStart += intervals * int64(s.window)
			_ = s.store.Set(tsKey, float64(windowStart), s.window)
		}
	}

	// Calculate interpolation ratio within the current window
	since := now - windowStart
	ratio := float64(since) / float64(s.window)

	// Load counts
	prevCount, err := s.store.Get(prevKey)
	if err != nil {
		return false, err
	}
	currCount, err := s.store.Get(currKey)
	if err != nil {
		return false, err
	}

	// Approximate total in sliding window
	slidingCount := prevCount*(1-ratio) + currCount
	allowed := slidingCount < float64(s.limit)

	if allowed {
		// Increment current window counter
		_, err := s.store.Incr(currKey, s.window)
		if err != nil {
			return false, err
		}
	}

	return allowed, nil
}
