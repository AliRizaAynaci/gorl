package algorithms

import (
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

type TokenBucketLimiter struct {
	limit  int
	window time.Duration
	store  storage.Storage
	prefix string
}

func NewTokenBucketLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &TokenBucketLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  store,
	}
}

func (t *TokenBucketLimiter) Allow(key string) (bool, error) {
	storageKey := t.prefix + ":" + key
	now := float64(time.Now().UnixNano())
	state, err := t.store.HMGet(storageKey, "tokens", "last_refill")

	var tokens, lastRefill float64
	if err != nil || state["last_refill"] == 0 {
		tokens = float64(t.limit)
		lastRefill = now
	} else {
		tokens = state["tokens"]
		lastRefill = state["last_refill"]

		elapsed := now - lastRefill
		refillRate := float64(t.limit) / float64(t.window.Nanoseconds())
		refilled := elapsed * refillRate

		if refilled > 0 {
			tokens = minFloat(tokens+refilled, float64(t.limit))
			lastRefill = lastRefill + refilled/refillRate
		}
	}

	if tokens >= 1 {
		tokens--
		fields := map[string]float64{
			"tokens":      tokens,
			"last_refill": lastRefill,
		}
		_ = t.store.HMSet(storageKey, fields, t.window)
		return true, nil
	}

	fields := map[string]float64{
		"tokens":      tokens,
		"last_refill": lastRefill,
	}
	_ = t.store.HMSet(storageKey, fields, t.window)
	return false, nil
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
