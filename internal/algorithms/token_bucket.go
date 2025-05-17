package algorithms

import (
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

type TokenBucketLimiter struct {
	limit        int
	window       time.Duration
	store        storage.Storage
	prefix       string
	mu           sync.Mutex
	timePerToken int64
}

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
		timePerToken: tpt,
	}
}

func (t *TokenBucketLimiter) Allow(key string) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	storageKey := t.prefix + ":" + key
	now := time.Now().UnixNano()

	state, _ := t.store.HMGet(storageKey, "tokens", "last_refill")
	var tokens int64
	var lastRefill int64

	if state["last_refill"] == 0 {
		tokens = int64(t.limit)
		lastRefill = now
	} else {
		tokens = int64(state["tokens"])
		lastRefill = int64(state["last_refill"])

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

	allowed := tokens > 0
	if allowed {
		tokens--
	}

	fields := map[string]float64{
		"tokens":      float64(tokens),
		"last_refill": float64(lastRefill),
	}
	_ = t.store.HMSet(storageKey, fields, t.window)

	return allowed, nil
}
