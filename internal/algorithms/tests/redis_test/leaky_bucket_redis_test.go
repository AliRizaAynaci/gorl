package redis_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/storage/redis"
)

func TestLeakyBucketLimiter_Basic(t *testing.T) {
	store := redis.NewRedisStore("redis://localhost:6379/0")

	cfg := core.Config{
		Limit:  3,
		Window: 2 * time.Second,
	}
	limiter := algorithms.NewLeakyBucketLimiter(cfg, store)
	CommonLimiterBehavior(t, limiter, "user-2", 3)
}

func BenchmarkLeakyBucketLimiter_SingleKey(b *testing.B) {
	b.ReportAllocs()

	store := redis.NewRedisStore("redis://localhost:6379/0")

	cfg := core.Config{
		Limit:  10000,
		Window: time.Second,
	}
	limiter := algorithms.NewLeakyBucketLimiter(cfg, store)
	key := "bench-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(key)
	}
}

func BenchmarkLeakyBucketLimiter_MultiKey(b *testing.B) {
	b.ReportAllocs()

	store := redis.NewRedisStore("redis://localhost:6379/0")

	cfg := core.Config{
		Limit:  10000,
		Window: time.Second,
	}
	limiter := algorithms.NewLeakyBucketLimiter(cfg, store)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(key)
	}
}

func TestLeakyBucketLimiter_Concurrency(t *testing.T) {
	store := redis.NewRedisStore("redis://localhost:6379/0")

	cfg := core.Config{
		Limit:  10,
		Window: 2 * time.Second,
	}
	limiter := algorithms.NewLeakyBucketLimiter(cfg, store)
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
	if allowedCount < 9 || allowedCount > 12 {
		t.Errorf("concurrency allowedCount = %d, expected ~10", allowedCount)
	}
}
