// Package algorithms implements various rate limiting algorithms.
package algorithms

import (
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

// TokenBucketLimiter implements the token bucket algorithm using a minimal Storage API (Get/Set only).
// State is stored in two separate keys per user: tokens and last refill timestamp.
type TokenBucketLimiter struct {
	limit        int           // maximum tokens
	window       time.Duration // refill window duration
	store        storage.Storage
	prefix       string     // key prefix, e.g. "gorl:tb"
	mu           sync.Mutex // ensure atomicity in-memory; Redis backend handles atomic Incr
	metrics      core.MetricsCollector
	timePerToken int64 // nanoseconds per token refill interval
	failOpen     bool
}

// NewTokenBucketLimiter constructs a new TokenBucketLimiter.
// Calculates timePerToken = window.Nanoseconds() / limit, with a minimum of 1.
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
// It reloads state with Get, recalculates tokens, and persists with Set.
func (t *TokenBucketLimiter) Allow(key string) (bool, error) {
	start := time.Now()
	// lock for in-memory safety; Redis backend operations are atomic.
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now().UnixNano()
	tokensKey := t.prefix + ":tokens:" + key
	refillKey := t.prefix + ":refill:" + key

	// Load current token count
	tokenVal, err := t.store.Get(tokensKey)
	if allowed, retErr, done := failOpenHandler(start, err, t.failOpen, t.metrics); done {
		return allowed, retErr
	}
	tokens := int64(tokenVal)

	// Load last refill timestamp
	lastRefillVal, err := t.store.Get(refillKey)
	if allowed, retErr, done := failOpenHandler(start, err, t.failOpen, t.metrics); done {
		return allowed, retErr
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
			// advance lastRefill
			lastRefill += newTokens * t.timePerToken
		}
	}

	// Check and consume
	allowed := tokens > 0
	if allowed {
		tokens--
	}

	// Persist updated values
	err = t.store.Set(tokensKey, float64(tokens), t.window)
	if allowed, retErr, done := failOpenHandler(start, err, t.failOpen, t.metrics); done {
		return allowed, retErr
	}
	err = t.store.Set(refillKey, float64(lastRefill), t.window)
	if allowed, retErr, done := failOpenHandler(start, err, t.failOpen, t.metrics); done {
		return allowed, retErr
	}

	t.metrics.ObserveLatency(time.Since(start))
	if allowed {
		t.metrics.IncAllow()
	} else {
		t.metrics.IncDeny()
	}

	return allowed, nil
}
