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

func CommonLimiterBehavior(t *testing.T, limiter core.Limiter, key string, limit int) {
	t.Helper()
	for i := 0; i < limit; i++ {
		allowed, err := limiter.Allow(key)
		if err != nil || !allowed {
			t.Fatalf("expected allowed, got %v, err %v (req %d)", allowed, err, i+1)
		}
	}
	allowed, err := limiter.Allow(key)
	if allowed || err != nil {
		t.Fatalf("expected denied after limit, got %v, err %v", allowed, err)
	}
}

func TestFixedWindowLimiter_Basic(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := algorithms.NewFixedWindowLimiter(core.Config{
		Limit:  3,
		Window: 2 * time.Second,
	}, store)
	CommonLimiterBehavior(t, limiter, "user-1", 3)
}

func BenchmarkFixedWindowLimiter_SingleKey(b *testing.B) {
	b.ReportAllocs()

	store := inmem.NewInMemoryStore()
	limiter := algorithms.NewFixedWindowLimiter(core.Config{
		Limit:  10000,
		Window: time.Second,
	}, store)
	key := "bench-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(key)
	}
}

func BenchmarkFixedWindowLimiter_MultiKey(b *testing.B) {
	b.ReportAllocs()

	store := inmem.NewInMemoryStore()
	limiter := algorithms.NewFixedWindowLimiter(core.Config{
		Limit:  10000,
		Window: time.Second,
	}, store)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(key)
	}
}

func TestFixedWindowLimiter_Concurrency(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := algorithms.NewFixedWindowLimiter(core.Config{
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
