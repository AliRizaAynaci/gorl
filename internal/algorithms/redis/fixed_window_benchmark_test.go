package redis

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func BenchmarkFixedWindowLimiter_Allow(b *testing.B) {
	b.ReportAllocs()
	cfg := core.Config{
		Limit:    1000,
		Window:   time.Minute,
		RedisURL: redisURL(),
	}
	limiter := NewFixedWindowLimiter(cfg)
	key := "bench-redis-fw"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(key)
	}
}
