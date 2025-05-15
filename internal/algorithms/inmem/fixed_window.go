package algorithms

import (
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

type fixedWindowBucket struct {
	count       int
	windowStart time.Time
	mu          sync.Mutex // Per-bucket locking
}

type fixedWindowLimiter struct {
	limit   int
	window  time.Duration
	buckets sync.Map // key: string, value: *fixedWindowBucket
}

func (f *fixedWindowLimiter) Allow(key string) (bool, error) {
	now := time.Now()

	var bkt *fixedWindowBucket
	val, ok := f.buckets.Load(key)
	if !ok {
		bkt = &fixedWindowBucket{
			count:       1,
			windowStart: now,
		}
		f.buckets.Store(key, bkt)
		return true, nil
	} else {
		bkt = val.(*fixedWindowBucket)
	}

	bkt.mu.Lock()
	defer bkt.mu.Unlock()

	if now.Sub(bkt.windowStart) < f.window {
		if bkt.count < f.limit {
			bkt.count++
			return true, nil
		}
		return false, nil
	}
	// Window expired, start new window
	bkt.count = 1
	bkt.windowStart = now
	return true, nil
}

func NewFixedWindowLimiter(cfg core.Config) core.Limiter {
	return &fixedWindowLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
	}
}
