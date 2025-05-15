package algorithms

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func BenchmarkTokenBucketLimiter_Allow(b *testing.B) {
	b.ReportAllocs()
	cfg := core.Config{
		Limit:  1000,
		Window: time.Minute,
	}
	limiter := NewTokenBucketLimiter(cfg)
	key := "bench-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(key)
	}
}
