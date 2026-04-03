// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"context"
	"fmt"
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
	if runner, ok := s.store.(redisScriptRunner); ok {
		return s.allowRedis(ctx, start, runner, key)
	}

	return s.allowGeneric(ctx, start, key)
}

func (s *SlidingWindowLimiter) allowGeneric(ctx context.Context, start time.Time, key string) (core.Result, error) {
	now := time.Now().UnixNano()

	tsKey := fmt.Sprintf("%s:{%s}:ts", s.prefix, key)
	currKey := fmt.Sprintf("%s:{%s}:curr", s.prefix, key)
	prevKey := fmt.Sprintf("%s:{%s}:prev", s.prefix, key)

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

func (s *SlidingWindowLimiter) allowRedis(ctx context.Context, start time.Time, runner redisScriptRunner, key string) (core.Result, error) {
	keys := []string{
		fmt.Sprintf("%s:{%s}:ts", s.prefix, key),
		fmt.Sprintf("%s:{%s}:curr", s.prefix, key),
		fmt.Sprintf("%s:{%s}:prev", s.prefix, key),
	}

	values, err := runner.EvalScript(
		ctx,
		redisScriptSlidingWindow,
		keys,
		int64(s.limit),
		time.Now().UnixMicro(),
		durationToMicros(s.window),
		durationToMilliseconds(s.stateTTL),
	)
	if res, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
		return res, retErr
	}

	res, err := buildRedisScriptResult(s.limit, values)
	if res2, retErr, done := failOpenHandler(start, err, s.failOpen, s.metrics, s.limit); done {
		return res2, retErr
	}

	s.metrics.ObserveLatency(time.Since(start))
	if res.Allowed {
		s.metrics.IncAllow()
	} else {
		s.metrics.IncDeny()
	}

	return res, nil
}

// Close releases resources held by the limiter.
func (s *SlidingWindowLimiter) Close() error {
	return s.store.Close()
}
