package redis

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func BenchmarkSlidingWindowLimiter_Allow(b *testing.B) {
	b.ReportAllocs()
	cfg := core.Config{
		Limit:    1000,
		Window:   time.Minute,
		RedisURL: redisURL(),
	}
	limiter := NewSlidingWindowLimiter(cfg)
	key := "bench-redis-sw"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(key)
	}
}
