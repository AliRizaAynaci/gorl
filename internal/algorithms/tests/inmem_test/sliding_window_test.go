package inmem_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/storage/inmem"
)

func TestSlidingWindowLimiter_Basic(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := algorithms.NewSlidingWindowLimiter(core.Config{
		Limit:  3,
		Window: 2 * time.Second,
	}, store)
	CommonLimiterBehavior(t, limiter, "user-1", 3)
}

func BenchmarkSlidingWindowLimiter_SingleKey(b *testing.B) {
	b.ReportAllocs()

	store := inmem.NewInMemoryStore()
	limiter := algorithms.NewSlidingWindowLimiter(core.Config{
		Limit:  10000,
		Window: time.Second,
	}, store)
	key := "bench-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(key)
	}
}

func BenchmarkSlidingWindowLimiter_MultiKey(b *testing.B) {
	b.ReportAllocs()

	store := inmem.NewInMemoryStore()
	limiter := algorithms.NewSlidingWindowLimiter(core.Config{
		Limit:  10000,
		Window: time.Second,
	}, store)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(key)
	}
}

func TestSlidingWindowLimiter_Concurrency(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := algorithms.NewSlidingWindowLimiter(core.Config{
		Limit:  10,
		Window: 2 * time.Second,
	}, store)
	key := "user-concurrent"

	var wg sync.WaitGroup
	var allowedCount int32
	workerCount := 100

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := limiter.Allow(key)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if allowed {
				atomic.AddInt32(&allowedCount, 1)
			}
		}()
	}
	wg.Wait()
	if allowedCount != 10 {
		t.Errorf("concurrency error: allowedCount = %d, expected 10", allowedCount)
	}
}
