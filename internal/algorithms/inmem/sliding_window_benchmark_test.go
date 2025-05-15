package algorithms

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func BenchmarkSlidingWindowLimiter_Allow(b *testing.B) {
	b.ReportAllocs()
	cfg := core.Config{
		Limit:  1000,
		Window: time.Minute,
	}
	limiter := NewSlidingWindowLimiter(cfg)
	key := "bench-sw"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.Allow(key)
	}
}
