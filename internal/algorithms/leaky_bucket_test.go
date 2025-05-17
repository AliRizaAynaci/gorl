package algorithms

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/storage/inmem"
)

func TestLeakyBucketLimiter_Basic(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit:  3,
		Window: 2 * time.Second,
	}, store)
	key := "test-leaky"
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(key)
		if !allowed || err != nil {
			t.Fatalf("should allow (i=%d) got allowed=%v err=%v", i, allowed, err)
		}
	}
	allowed, err := limiter.Allow(key)
	if allowed || err != nil {
		t.Fatalf("should deny after limit, got allowed=%v err=%v", allowed, err)
	}
}

func BenchmarkLeakyBucketLimiter_SingleKey(b *testing.B) {
	b.ReportAllocs()

	store := inmem.NewInMemoryStore()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit:  10000,
		Window: time.Second,
	}, store)
	key := "bench-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(key)
	}
}

func BenchmarkLeakyBucketLimiter_MultiKey(b *testing.B) {
	b.ReportAllocs()

	store := inmem.NewInMemoryStore()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit:  10000,
		Window: time.Second,
	}, store)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(key)
	}
}

func TestLeakyBucketLimiter_Concurrency(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit:  10,
		Window: 2 * time.Second,
	}, store)
	key := "user-concurrent-lb"

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
	maxAllowed := 10
	tolerance := 3
	if int(allowedCount) < maxAllowed || int(allowedCount) > maxAllowed+tolerance {
		t.Errorf("concurrency error: allowedCount = %d, expected 10", allowedCount)
	}
}
