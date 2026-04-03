// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"context"
	"math"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage"
)

// SlidingWindowLimiter implements an approximate sliding window algorithm using minimal Storage API.
// It keeps two counters (current and previous window) and a timestamp of the window start.
type SlidingWindowLimiter struct {
	limit    int
	window   time.Duration
	stateTTL time.Duration
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
		stateTTL: 2 * cfg.Window,
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
		if err := s.store.Set(ctx, tsKey, float64(windowStart), s.stateTTL); err != nil {
			if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
				return res, retErr
			}
		}
		if err := s.store.Set(ctx, currKey, 0, s.stateTTL); err != nil {
			if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
				return res, retErr
			}
		}
		if err := s.store.Set(ctx, prevKey, 0, s.stateTTL); err != nil {
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
			nextPrevCount := 0.0
			if intervals == 1 {
				nextPrevCount = currCount
			}
			if err := s.store.Set(ctx, prevKey, nextPrevCount, s.stateTTL); err != nil {
				if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
					return res, retErr
				}
			}
			if err := s.store.Set(ctx, currKey, 0, s.stateTTL); err != nil {
				if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
					return res, retErr
				}
			}

			windowStart += intervals * int64(s.window)
			if err := s.store.Set(ctx, tsKey, float64(windowStart), s.stateTTL); err != nil {
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

	// Approximate total in sliding window before handling the current request.
	slidingCount := prevCount*(1-ratio) + currCount
	allowed := slidingCount < float64(s.limit)
	currCountAfter := currCount
	slidingCountAfter := slidingCount

	if allowed {
		_, err := s.store.Incr(ctx, currKey, s.stateTTL)
		if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
			return res, retErr
		}
		currCountAfter++
		slidingCountAfter++
	}

	s.metrics.ObserveLatency(time.Since(start))

	remaining := int(float64(s.limit) - slidingCountAfter)
	if remaining < 0 {
		remaining = 0
	}

	windowUntilBoundary := clampDuration(time.Duration(int64(s.window)-since) * time.Nanosecond)
	reset := time.Duration(0)
	switch {
	case currCountAfter > 0:
		reset = clampDuration(time.Duration(2*int64(s.window)-since) * time.Nanosecond)
	case prevCount > 0:
		reset = windowUntilBoundary
	}

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
		switch {
		case currCount >= float64(s.limit):
			res.RetryAfter = windowUntilBoundary
		case prevCount > 0:
			requiredRatio := 1 - (float64(s.limit)-currCount)/prevCount
			delayRatio := requiredRatio - ratio
			if delayRatio < 0 {
				delayRatio = 0
			}
			res.RetryAfter = clampDuration(time.Duration(math.Ceil(delayRatio*float64(s.window.Nanoseconds()))) * time.Nanosecond)
			if res.RetryAfter == 0 {
				res.RetryAfter = time.Nanosecond
			}
			if res.RetryAfter > windowUntilBoundary {
				res.RetryAfter = windowUntilBoundary
			}
		default:
			res.RetryAfter = windowUntilBoundary
		}
	}
	return res, nil
}

// Close releases resources held by the limiter.
func (s *SlidingWindowLimiter) Close() error {
	return s.store.Close()
}
