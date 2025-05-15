package redis

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func BenchmarkTokenBucketLimiter_Allow(b *testing.B) {
	b.ReportAllocs()
	cfg := core.Config{
		Limit:    1000,
		Window:   time.Minute,
		RedisURL: redisURL(),
	}
	limiter := NewTokenBucketLimiter(cfg)
	key := "bench-redis-tb"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(key)
	}
}
