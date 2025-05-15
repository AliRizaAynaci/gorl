package algorithms

import (
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

type leakyBucket struct {
	water        float64
	lastLeakTime time.Time
	mu           sync.Mutex // Per-bucket locking
}

type leakyBucketLimiter struct {
	limit   int
	window  time.Duration
	buckets sync.Map // key: string, value: *leakyBucket
}

func (l *leakyBucketLimiter) Allow(key string) (bool, error) {
	now := time.Now()

	var bkt *leakyBucket
	val, ok := l.buckets.Load(key)
	if !ok {
		bkt = &leakyBucket{
			water:        0,
			lastLeakTime: now,
		}
		l.buckets.Store(key, bkt)
	} else {
		bkt = val.(*leakyBucket)
	}

	bkt.mu.Lock()
	defer bkt.mu.Unlock()

	leakRate := float64(l.limit) / l.window.Seconds() // water per second
	elapsed := now.Sub(bkt.lastLeakTime).Seconds()
	leaked := elapsed * leakRate
	bkt.water = maxFloat(0, bkt.water-leaked)
	bkt.lastLeakTime = now

	if bkt.water < float64(l.limit) {
		bkt.water++
		return true, nil
	}
	return false, nil
}

func NewLeakyBucketLimiter(cfg core.Config) core.Limiter {
	return &leakyBucketLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
