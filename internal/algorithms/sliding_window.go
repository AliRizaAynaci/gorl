// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"context"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage"
)

// SlidingWindowLimiter implements an approximate sliding window algorithm using minimal Storage API.
// It keeps two counters (current and previous window) and a timestamp of the window start.
type SlidingWindowLimiter struct {
	limit    int
	window   time.Duration
	store    storage.Storage
	prefix   string
	metrics  core.MetricsCollector
	failOpen bool
}

// NewSlidingWindowLimiter constructs a new SlidingWindowLimiter.
func NewSlidingWindowLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &SlidingWindowLimiter{
		limit:    cfg.Limit,
		window:   cfg.Window,
		store:    store,
		prefix:   "gorl:sw",
		metrics:  cfg.Metrics,
		failOpen: cfg.FailOpen,
	}
}

// Allow checks whether a request is allowed under a sliding window.
func (s *SlidingWindowLimiter) Allow(ctx context.Context, key string) (core.Result, error) {
	start := time.Now()
	now := time.Now().UnixNano()

	tsKey := s.prefix + ":ts:" + key
	currKey := s.prefix + ":curr:" + key
	prevKey := s.prefix + ":prev:" + key

	// Load last window start
	tsVal, err := s.store.Get(ctx, tsKey)
	if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
		return res, retErr
	}

	var windowStart int64
	if tsVal == 0 {
		// First request: initialize
		windowStart = now
		if err := s.store.Set(ctx, tsKey, float64(windowStart), s.window); err != nil {
			if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
				return res, retErr
			}
		}
		if err := s.store.Set(ctx, currKey, 0, s.window); err != nil {
			if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
				return res, retErr
			}
		}
		if err := s.store.Set(ctx, prevKey, 0, s.window); err != nil {
			if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
				return res, retErr
			}
		}
	} else {
		windowStart = int64(tsVal)
		elapsed := now - windowStart

		if elapsed >= int64(s.window) {
			intervals := elapsed / int64(s.window)

			currCount, err := s.store.Get(ctx, currKey)
			if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
				return res, retErr
			}
			if err := s.store.Set(ctx, prevKey, currCount, s.window); err != nil {
				if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
					return res, retErr
				}
			}
			if err := s.store.Set(ctx, currKey, 0, s.window); err != nil {
				if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
					return res, retErr
				}
			}

			windowStart += intervals * int64(s.window)
			if err := s.store.Set(ctx, tsKey, float64(windowStart), s.window); err != nil {
				if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
					return res, retErr
				}
			}
		}
	}

	// Calculate interpolation ratio within the current window
	since := now - windowStart
	ratio := float64(since) / float64(s.window)

	// Load counts
	prevCount, err := s.store.Get(ctx, prevKey)
	if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
		return res, retErr
	}
	currCount, err := s.store.Get(ctx, currKey)
	if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
		return res, retErr
	}

	// Approximate total in sliding window
	slidingCount := prevCount*(1-ratio) + currCount
	allowed := slidingCount < float64(s.limit)

	if allowed {
		_, err := s.store.Incr(ctx, currKey, s.window)
		if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
			return res, retErr
		}
	}

	s.metrics.ObserveLatency(time.Since(start))

	remaining := s.limit - int(slidingCount)
	if remaining < 0 {
		remaining = 0
	}

	// Reset for sliding window is a bit fuzzy, but we can say when the CURRENT window ends
	reset := time.Duration(windowStart+int64(s.window)-now) * time.Nanosecond

	res := core.Result{
		Allowed:   allowed,
		Limit:     s.limit,
		Remaining: remaining,
		Reset:     reset,
	}

	if allowed {
		s.metrics.IncAllow()
	} else {
		s.metrics.IncDeny()
		res.RetryAfter = reset // In sliding window, wait until next window start might be too long, but it's a safe bet
	}
	return res, nil
}

// Close releases resources held by the limiter.
func (s *SlidingWindowLimiter) Close() error {
	return s.store.Close()
}
