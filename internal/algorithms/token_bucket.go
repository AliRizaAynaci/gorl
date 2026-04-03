// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"context"
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage"
)

// TokenBucketLimiter implements the token bucket algorithm using a minimal Storage API (Get/Set only).
// State is stored in two separate keys per user: tokens and last refill timestamp.
type TokenBucketLimiter struct {
	limit        int
	window       time.Duration
	store        storage.Storage
	prefix       string
	mu           sync.Mutex
	metrics      core.MetricsCollector
	timePerToken int64
	failOpen     bool
}

// NewTokenBucketLimiter constructs a new TokenBucketLimiter.
func NewTokenBucketLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	tpt := cfg.Window.Nanoseconds() / int64(cfg.Limit)
	if tpt <= 0 {
		tpt = 1
	}
	return &TokenBucketLimiter{
		limit:        cfg.Limit,
		window:       cfg.Window,
		store:        store,
		prefix:       "gorl:tb",
		metrics:      cfg.Metrics,
		timePerToken: tpt,
		failOpen:     cfg.FailOpen,
	}
}

// Allow checks token availability and consumes one token if allowed.
func (t *TokenBucketLimiter) Allow(ctx context.Context, key string) (core.Result, error) {
	start := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now().UnixNano()
	tokensKey := t.prefix + ":tokens:" + key
	refillKey := t.prefix + ":refill:" + key

	// Load current token count
	tokenVal, err := t.store.Get(ctx, tokensKey)
	if res, retErr, done := failOpenHandler(start, err, t.failOpen, t.metrics, t.limit); done {
		return res, retErr
	}
	tokens := int64(tokenVal)

	// Load last refill timestamp
	lastRefillVal, err := t.store.Get(ctx, refillKey)
	if res, retErr, done := failOpenHandler(start, err, t.failOpen, t.metrics, t.limit); done {
		return res, retErr
	}
	lastRefill := int64(lastRefillVal)

	// Initialize on first request
	if lastRefill == 0 {
		tokens = int64(t.limit)
		lastRefill = now
	} else {
		// Refill tokens based on elapsed time
		elapsed := now - lastRefill
		newTokens := elapsed / t.timePerToken
		if newTokens > 0 {
			tokens += newTokens
			if tokens > int64(t.limit) {
				tokens = int64(t.limit)
			}
			lastRefill += newTokens * t.timePerToken
		}
	}

	// Check and consume
	allowed := tokens > 0
	if allowed {
		tokens--
	}

	elapsedSinceRefill := now - lastRefill
	nextTokenDelay := clampDuration(time.Duration(t.timePerToken-elapsedSinceRefill) * time.Nanosecond)
	missingTokens := int64(t.limit) - tokens
	reset := time.Duration(0)
	if missingTokens > 0 {
		reset = clampDuration(time.Duration(missingTokens*t.timePerToken-elapsedSinceRefill) * time.Nanosecond)
	}

	// Persist updated values
	err = t.store.Set(ctx, tokensKey, float64(tokens), t.window)
	if res, retErr, done := failOpenHandler(start, err, t.failOpen, t.metrics, t.limit); done {
		return res, retErr
	}
	err = t.store.Set(ctx, refillKey, float64(lastRefill), t.window)
	if res, retErr, done := failOpenHandler(start, err, t.failOpen, t.metrics, t.limit); done {
		return res, retErr
	}

	t.metrics.ObserveLatency(time.Since(start))

	res := core.Result{
		Allowed:   allowed,
		Limit:     t.limit,
		Remaining: int(tokens),
		Reset:     reset,
	}

	if allowed {
		t.metrics.IncAllow()
	} else {
		t.metrics.IncDeny()
		res.RetryAfter = nextTokenDelay
	}

	return res, nil
}

// Close releases resources held by the limiter.
func (t *TokenBucketLimiter) Close() error {
	return t.store.Close()
}
