package algorithms

import (
	"math"
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
	now := time.Now().UnixNano()

	state, _ := l.store.HMGet(storageKey, "water_level", "last_leak")

	var waterLevel int
	var lastLeak int64
	if state["last_leak"] == 0 {
		waterLevel = 0
		lastLeak = now
	} else {
		waterLevel = int(state["water_level"])
		lastLeak = int64(state["last_leak"])

		elapsed := now - lastLeak
		tokensPerNano := float64(l.limit) / float64(l.window.Nanoseconds())
		leakedTokens := int64(math.Floor(float64(elapsed) * tokensPerNano))

		if leakedTokens > 0 {
			waterLevel -= int(leakedTokens)
			if waterLevel < 0 {
				waterLevel = 0
			}
			lastLeak += int64(math.Floor(float64(leakedTokens) / tokensPerNano))
		}
	}

	allowed := waterLevel < l.limit
	if allowed {
		waterLevel++
	}

	fields := map[string]float64{
		"water_level": float64(waterLevel),
		"last_leak":   float64(lastLeak),
	}
	_ = l.store.HMSet(storageKey, fields, l.window)

	return allowed, nil
}
