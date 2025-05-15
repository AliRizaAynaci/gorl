package algorithms

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func BenchmarkLeakyBucketLimiter_Allow(b *testing.B) {
	b.ReportAllocs()
	cfg := core.Config{
		Limit:  1000,
		Window: time.Minute,
	}
	limiter := NewLeakyBucketLimiter(cfg)
	key := "bench-lb"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(key)
	}
}
