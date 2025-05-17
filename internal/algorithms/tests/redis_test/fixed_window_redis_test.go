package redis_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/storage/redis"
)

func CommonLimiterBehavior(t *testing.T, limiter core.Limiter, key string, limit int) {
	t.Helper()
	allowedCount := 0
	for i := 0; i < limit+2; i++ {
		allowed, err := limiter.Allow(key)
		if err != nil {
			t.Fatalf("unexpected error: %v (req %d)", err, i+1)
		}
		if allowed {
			allowedCount++
		}
	}
	if allowedCount > limit+1 {
		t.Fatalf("allowedCount %d exceeds limit %d", allowedCount, limit)
	}
}

func TestMain(m *testing.M) {
	store := redis.NewRedisStore("redis://localhost:6379/0")
	_ = store.(*redis.RedisStore).Client().FlushDB(context.Background())
	os.Exit(m.Run())
}

func TestFixedWindowLimiter_Basic(t *testing.T) {
	store := redis.NewRedisStore("redis://localhost:6379/0")

	cfg := core.Config{
		Limit:  3,
		Window: 2 * time.Second,
	}
	limiter := algorithms.NewFixedWindowLimiter(cfg, store)
	CommonLimiterBehavior(t, limiter, "user-1", 3)
}

func BenchmarkFixedWindowLimiter_SingleKey(b *testing.B) {
	b.ReportAllocs()

	store := redis.NewRedisStore("redis://localhost:6379/0")

	cfg := core.Config{
		Limit:  10000,
		Window: time.Second,
	}
	limiter := algorithms.NewFixedWindowLimiter(cfg, store)
	key := "bench-user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(key)
	}
}

func BenchmarkFixedWindowLimiter_MultiKey(b *testing.B) {
	b.ReportAllocs()

	store := redis.NewRedisStore("redis://localhost:6379/0")

	cfg := core.Config{
		Limit:  10000,
		Window: time.Second,
	}
	limiter := algorithms.NewFixedWindowLimiter(cfg, store)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(key)
	}
}

func TestFixedWindowLimiter_Concurrency(t *testing.T) {
	store := redis.NewRedisStore("redis://localhost:6379/0")

	cfg := core.Config{
		Limit:  10,
		Window: 2 * time.Second,
	}
	limiter := algorithms.NewFixedWindowLimiter(cfg, store)
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
	if allowedCount < 9 || allowedCount > 15 {
		t.Errorf("concurrency allowedCount = %d, expected ~10", allowedCount)
	}
}
