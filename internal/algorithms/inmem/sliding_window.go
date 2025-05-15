package algorithms

import (
	"sync"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

type slidingWindowBucket struct {
	timestamps []time.Time
	mu         sync.Mutex // Per-bucket locking
}

type slidingWindowLimiter struct {
	limit   int
	window  time.Duration
	buckets sync.Map // key: string, value: *slidingWindowBucket
}

func (s *slidingWindowLimiter) Allow(key string) (bool, error) {
	now := time.Now()

	var bkt *slidingWindowBucket
	val, ok := s.buckets.Load(key)
	if !ok {
		bkt = &slidingWindowBucket{timestamps: []time.Time{}}
		s.buckets.Store(key, bkt)
	} else {
		bkt = val.(*slidingWindowBucket)
	}

	bkt.mu.Lock()
	defer bkt.mu.Unlock()

	// Remove expired timestamps
	cutoff := now.Add(-s.window)
	validTimestamps := bkt.timestamps[:0]
	for _, ts := range bkt.timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	bkt.timestamps = validTimestamps

	if len(bkt.timestamps) < s.limit {
		bkt.timestamps = append(bkt.timestamps, now)
		return true, nil
	}
	return false, nil
}

func NewSlidingWindowLimiter(cfg core.Config) core.Limiter {
	return &slidingWindowLimiter{
		limit:  cfg.Limit,
		window: cfg.Window,
	}
}
