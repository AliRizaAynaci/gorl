package algorithms

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage"
	"github.com/AliRizaAynaci/gorl/v2/storage/inmem"
)

const (
	benchmarkKeyCount     = 1024
	benchmarkRequestLimit = 1_000_000_000
)

type benchmarkLimiterFactory func(core.Config, storage.Storage) core.Limiter

type benchmarkFailure struct {
	err    error
	denied bool
}

func benchmarkKeys(prefix string, multiKey bool) []string {
	if !multiKey {
		return []string{prefix}
	}

	keys := make([]string, benchmarkKeyCount)
	for i := range keys {
		keys[i] = fmt.Sprintf("%s-%d", prefix, i)
	}
	return keys
}

func newBenchmarkLimiter(b *testing.B, factory benchmarkLimiterFactory, limit int) core.Limiter {
	b.Helper()

	store := inmem.NewInMemoryStore()
	b.Cleanup(func() {
		if err := store.Close(); err != nil {
			b.Errorf("close in-memory benchmark store: %v", err)
		}
	})

	return factory(core.Config{
		Limit:   limit,
		Window:  time.Hour,
		Metrics: &core.NoopMetrics{},
	}, store)
}

func warmBenchmarkLimiter(b *testing.B, limiter core.Limiter, key string) {
	b.Helper()

	result, err := limiter.Allow(context.Background(), key)
	if err != nil {
		b.Fatalf("warm benchmark limiter: %v", err)
	}
	if !result.Allowed {
		b.Fatal("warm benchmark limiter: request was denied")
	}
}

func benchmarkLimiterSequential(b *testing.B, factory benchmarkLimiterFactory, multiKey bool) {
	b.Helper()
	b.ReportAllocs()

	limiter := newBenchmarkLimiter(b, factory, benchmarkRequestLimit)
	keys := benchmarkKeys("bench", multiKey)
	warmBenchmarkLimiter(b, limiter, "bench-warmup")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := limiter.Allow(ctx, keys[i%len(keys)])
		if err != nil {
			b.Fatalf("benchmark Allow: %v", err)
		}
		if !result.Allowed {
			b.Fatalf("benchmark Allow denied iteration %d", i)
		}
	}
}

func benchmarkLimiterDenied(b *testing.B, factory benchmarkLimiterFactory) {
	b.Helper()
	b.ReportAllocs()

	limiter := newBenchmarkLimiter(b, factory, 1)
	ctx := context.Background()
	result, err := limiter.Allow(ctx, "bench-denied")
	if err != nil {
		b.Fatalf("prime denied-path benchmark: %v", err)
	}
	if !result.Allowed {
		b.Fatal("prime denied-path benchmark: first request was denied")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = limiter.Allow(ctx, "bench-denied")
		if err != nil {
			b.Fatalf("denied-path benchmark Allow: %v", err)
		}
		if result.Allowed {
			b.Fatalf("denied-path benchmark allowed iteration %d", i)
		}
	}
}

func benchmarkLimiterParallel(b *testing.B, factory benchmarkLimiterFactory, multiKey bool) {
	b.Helper()
	b.ReportAllocs()

	limiter := newBenchmarkLimiter(b, factory, benchmarkRequestLimit)
	keys := benchmarkKeys("bench", multiKey)
	warmBenchmarkLimiter(b, limiter, "bench-warmup")
	ctx := context.Background()
	var failure atomic.Pointer[benchmarkFailure]
	var workerSequence atomic.Uint64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := int(workerSequence.Add(1) - 1)
		for pb.Next() {
			result, err := limiter.Allow(ctx, keys[i%len(keys)])
			if err != nil {
				failure.CompareAndSwap(nil, &benchmarkFailure{err: err})
			} else if !result.Allowed {
				failure.CompareAndSwap(nil, &benchmarkFailure{denied: true})
			}
			i++
		}
	})
	b.StopTimer()

	if got := failure.Load(); got != nil {
		if got.err != nil {
			b.Fatalf("parallel benchmark Allow: %v", got.err)
		}
		if got.denied {
			b.Fatal("parallel benchmark Allow unexpectedly denied a request")
		}
	}
}
