package algorithms_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/v2/storage"
	redisstore "github.com/AliRizaAynaci/gorl/v2/storage/redis"
)

func redisURLForTests() string {
	if url := os.Getenv("GORL_REDIS_URL"); url != "" {
		return url
	}
	return "redis://127.0.0.1:6379/0"
}

func newRedisStoreForTest(t *testing.T) storage.Storage {
	t.Helper()

	store, err := redisstore.NewRedisStore(redisURLForTests())
	if err != nil {
		t.Skipf("skipping redis integration tests: %v", err)
	}
	return store
}

func TestRedisAtomicAlgorithms_MultiInstanceBurst(t *testing.T) {
	strategies := []struct {
		name        string
		constructor func(core.Config, storage.Storage) core.Limiter
	}{
		{"FixedWindow", algorithms.NewFixedWindowLimiter},
		{"SlidingWindow", algorithms.NewSlidingWindowLimiter},
		{"TokenBucket", algorithms.NewTokenBucketLimiter},
		{"LeakyBucket", algorithms.NewLeakyBucketLimiter},
	}

	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			storeA := newRedisStoreForTest(t)
			limiterA := strategy.constructor(core.Config{
				Limit:   20,
				Window:  time.Minute,
				Metrics: &core.NoopMetrics{},
			}, storeA)
			defer limiterA.Close()

			storeB := newRedisStoreForTest(t)
			limiterB := strategy.constructor(core.Config{
				Limit:   20,
				Window:  time.Minute,
				Metrics: &core.NoopMetrics{},
			}, storeB)
			defer limiterB.Close()

			key := fmt.Sprintf("%s-burst-%d", strategy.name, time.Now().UnixNano())
			var allowed int32
			errCh := make(chan error, 200)

			var wg sync.WaitGroup
			for i := 0; i < 200; i++ {
				wg.Add(1)
				limiter := limiterA
				if i%2 == 1 {
					limiter = limiterB
				}

				go func(l core.Limiter) {
					defer wg.Done()
					res, err := l.Allow(context.Background(), key)
					if err != nil {
						errCh <- err
						return
					}
					if res.Allowed {
						atomic.AddInt32(&allowed, 1)
					}
				}(limiter)
			}

			wg.Wait()
			close(errCh)

			for err := range errCh {
				if err != nil {
					t.Fatalf("unexpected redis burst error: %v", err)
				}
			}

			if got := atomic.LoadInt32(&allowed); got != 20 {
				t.Fatalf("expected exactly 20 allowed requests, got %d", got)
			}
		})
	}
}

func TestRedisAtomicAlgorithms_ResultMetadata(t *testing.T) {
	strategies := []struct {
		name        string
		constructor func(core.Config, storage.Storage) core.Limiter
	}{
		{"FixedWindow", algorithms.NewFixedWindowLimiter},
		{"SlidingWindow", algorithms.NewSlidingWindowLimiter},
		{"TokenBucket", algorithms.NewTokenBucketLimiter},
		{"LeakyBucket", algorithms.NewLeakyBucketLimiter},
	}

	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			store := newRedisStoreForTest(t)
			limiter := strategy.constructor(core.Config{
				Limit:   2,
				Window:  time.Minute,
				Metrics: &core.NoopMetrics{},
			}, store)
			defer limiter.Close()

			key := fmt.Sprintf("%s-meta-%d", strategy.name, time.Now().UnixNano())

			first, err := limiter.Allow(context.Background(), key)
			if err != nil {
				t.Fatalf("unexpected error on first request: %v", err)
			}
			if !first.Allowed {
				t.Fatal("first request should be allowed")
			}
			if first.Reset <= 0 {
				t.Fatalf("expected positive reset, got %v", first.Reset)
			}

			second, err := limiter.Allow(context.Background(), key)
			if err != nil {
				t.Fatalf("unexpected error on second request: %v", err)
			}
			if !second.Allowed {
				t.Fatal("second request should be allowed")
			}
			if second.Remaining != 0 {
				t.Fatalf("expected remaining=0 after second request, got %d", second.Remaining)
			}

			denied, err := limiter.Allow(context.Background(), key)
			if err != nil {
				t.Fatalf("unexpected error on denied request: %v", err)
			}
			if denied.Allowed {
				t.Fatal("third request should be denied")
			}
			if denied.Remaining != 0 {
				t.Fatalf("expected remaining=0 on denied request, got %d", denied.Remaining)
			}
			if denied.Reset <= 0 {
				t.Fatalf("expected positive reset on denied request, got %v", denied.Reset)
			}
			if denied.RetryAfter <= 0 {
				t.Fatalf("expected positive retry_after on denied request, got %v", denied.RetryAfter)
			}
		})
	}
}
