package algorithms

import (
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage"
)

type LeakyBucketLimiter struct {
	limit  int
	window time.Duration
	store  storage.Storage
	prefix string
	mu     sync.Mutex
}

func NewLeakyBucketLimiter(cfg core.Config, store storage.Storage) core.Limiter {
	return &LeakyBucketLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  store,
		prefix: "gorl:lb",
	}
}

func (l *LeakyBucketLimiter) Allow(key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	storageKey := l.prefix + ":" + key
	now := float64(time.Now().UnixNano())

	state, err := l.store.HMGet(storageKey, "water_level", "last_leak")

	var currentWaterLevel, currentLastLeak float64
	if err != nil || state["last_leak"] == 0 {
		currentWaterLevel = 0
		currentLastLeak = now
	} else {
		currentWaterLevel = state["water_level"]
		currentLastLeak = state["last_leak"]

		elapsed := now - currentLastLeak
		leakRate := float64(l.limit) / float64(l.window.Nanoseconds())
		leakedAmount := elapsed * leakRate

		if leakedAmount > 0 {
			currentWaterLevel = maxFloat(0, currentWaterLevel-leakedAmount)
			currentLastLeak = currentLastLeak + leakedAmount/leakRate
		}
	}

	allowed := currentWaterLevel < float64(l.limit)

	if allowed {
		currentWaterLevel++
	}

	fields := map[string]float64{
		"water_level": currentWaterLevel,
		"last_leak":   currentLastLeak,
	}
	_ = l.store.HMSet(storageKey, fields, l.window)

	return allowed, nil
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
