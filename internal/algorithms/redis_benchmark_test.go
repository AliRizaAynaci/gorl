package algorithms_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/v2/storage"
	redisstore "github.com/AliRizaAynaci/gorl/v2/storage/redis"
)

const (
	redisBenchmarkKeyCount     = 1024
	redisBenchmarkRequestLimit = 1_000_000_000
)

var redisBenchmarkSequence atomic.Uint64

type redisBenchmarkLimiterFactory func(core.Config, storage.Storage) core.Limiter

type redisBenchmarkFailure struct {
	err    error
	denied bool
}

func newRedisBenchmarkStore(b *testing.B) *redisstore.RedisStore {
	b.Helper()

	url := "redis://127.0.0.1:6379/0"
	if configuredURL := os.Getenv("GORL_REDIS_URL"); configuredURL != "" {
		url = configuredURL
	}

	rawStore, err := redisstore.NewRedisStore(url)
	if err != nil {
		b.Fatalf("connect to Redis for benchmark: %v", err)
	}
	store, ok := rawStore.(*redisstore.RedisStore)
	if !ok {
		b.Fatalf("unexpected Redis benchmark store type %T", rawStore)
	}
	return store
}

func redisBenchmarkMarker(b *testing.B) string {
	b.Helper()

	name := strings.NewReplacer("/", "-", "=", "-").Replace(b.Name())
	return fmt.Sprintf("gorl-bench-%d-%d-%s", os.Getpid(), redisBenchmarkSequence.Add(1), name)
}

func redisBenchmarkKeys(marker string, multiKey bool) []string {
	if !multiKey {
		return []string{marker}
	}

	keys := make([]string, redisBenchmarkKeyCount)
	for i := range keys {
		keys[i] = fmt.Sprintf("%s-%d", marker, i)
	}
	return keys
}

func cleanupRedisBenchmarkKeys(b *testing.B, store *redisstore.RedisStore, marker string) {
	b.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cursor uint64
	for {
		keys, nextCursor, err := store.Client().Scan(ctx, cursor, "*"+marker+"*", 256).Result()
		if err != nil {
			b.Errorf("scan Redis benchmark keys for cleanup: %v", err)
			return
		}
		if len(keys) > 0 {
			if err := store.Client().Del(ctx, keys...).Err(); err != nil {
				b.Errorf("delete Redis benchmark keys: %v", err)
				return
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return
		}
	}
}

func newRedisBenchmarkLimiter(
	b *testing.B,
	factory redisBenchmarkLimiterFactory,
	limit int,
) (core.Limiter, []string, string) {
	b.Helper()

	store := newRedisBenchmarkStore(b)
	marker := redisBenchmarkMarker(b)
	b.Cleanup(func() {
		cleanupRedisBenchmarkKeys(b, store, marker)
		if err := store.Close(); err != nil {
			b.Errorf("close Redis benchmark store: %v", err)
		}
	})

	limiter := factory(core.Config{
		Limit:   limit,
		Window:  time.Hour,
		Metrics: &core.NoopMetrics{},
	}, store)
	return limiter, redisBenchmarkKeys(marker, false), marker
}

func warmRedisBenchmarkLimiter(b *testing.B, limiter core.Limiter, marker string) {
	b.Helper()

	result, err := limiter.Allow(context.Background(), marker+"-warmup")
	if err != nil {
		b.Fatalf("warm Redis benchmark limiter: %v", err)
	}
	if !result.Allowed {
		b.Fatal("warm Redis benchmark limiter: request was denied")
	}
}

func benchmarkRedisSequential(
	b *testing.B,
	factory redisBenchmarkLimiterFactory,
	multiKey bool,
) {
	b.Helper()
	b.ReportAllocs()

	limiter, keys, marker := newRedisBenchmarkLimiter(b, factory, redisBenchmarkRequestLimit)
	if multiKey {
		keys = redisBenchmarkKeys(marker, true)
	}
	warmRedisBenchmarkLimiter(b, limiter, marker)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := limiter.Allow(ctx, keys[i%len(keys)])
		if err != nil {
			b.Fatalf("Redis benchmark Allow: %v", err)
		}
		if !result.Allowed {
			b.Fatalf("Redis benchmark Allow denied iteration %d", i)
		}
	}
}

func benchmarkRedisDenied(b *testing.B, factory redisBenchmarkLimiterFactory) {
	b.Helper()
	b.ReportAllocs()

	limiter, keys, _ := newRedisBenchmarkLimiter(b, factory, 1)
	ctx := context.Background()
	result, err := limiter.Allow(ctx, keys[0])
	if err != nil {
		b.Fatalf("prime denied-path Redis benchmark: %v", err)
	}
	if !result.Allowed {
		b.Fatal("prime denied-path Redis benchmark: first request was denied")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = limiter.Allow(ctx, keys[0])
		if err != nil {
			b.Fatalf("denied-path Redis benchmark Allow: %v", err)
		}
		if result.Allowed {
			b.Fatalf("denied-path Redis benchmark allowed iteration %d", i)
		}
	}
}

func benchmarkRedisParallel(
	b *testing.B,
	factory redisBenchmarkLimiterFactory,
	multiKey bool,
) {
	b.Helper()
	b.ReportAllocs()

	limiter, keys, marker := newRedisBenchmarkLimiter(b, factory, redisBenchmarkRequestLimit)
	if multiKey {
		keys = redisBenchmarkKeys(marker, true)
	}
	warmRedisBenchmarkLimiter(b, limiter, marker)
	ctx := context.Background()
	var failure atomic.Pointer[redisBenchmarkFailure]
	var workerSequence atomic.Uint64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := int(workerSequence.Add(1) - 1)
		for pb.Next() {
			result, err := limiter.Allow(ctx, keys[i%len(keys)])
			if err != nil {
				failure.CompareAndSwap(nil, &redisBenchmarkFailure{err: err})
			} else if !result.Allowed {
				failure.CompareAndSwap(nil, &redisBenchmarkFailure{denied: true})
			}
			i++
		}
	})
	b.StopTimer()

	if got := failure.Load(); got != nil {
		if got.err != nil {
			b.Fatalf("parallel Redis benchmark Allow: %v", got.err)
		}
		if got.denied {
			b.Fatal("parallel Redis benchmark Allow unexpectedly denied a request")
		}
	}
}

func BenchmarkRedis_FixedWindow_SingleKey(b *testing.B) {
	benchmarkRedisSequential(b, algorithms.NewFixedWindowLimiter, false)
}

func BenchmarkRedis_FixedWindow_MultiKey(b *testing.B) {
	benchmarkRedisSequential(b, algorithms.NewFixedWindowLimiter, true)
}

func BenchmarkRedis_FixedWindow_DeniedSingleKey(b *testing.B) {
	benchmarkRedisDenied(b, algorithms.NewFixedWindowLimiter)
}

func BenchmarkRedis_FixedWindow_ParallelSingleKey(b *testing.B) {
	benchmarkRedisParallel(b, algorithms.NewFixedWindowLimiter, false)
}

func BenchmarkRedis_FixedWindow_ParallelMultiKey(b *testing.B) {
	benchmarkRedisParallel(b, algorithms.NewFixedWindowLimiter, true)
}

func BenchmarkRedis_SlidingWindow_SingleKey(b *testing.B) {
	benchmarkRedisSequential(b, algorithms.NewSlidingWindowLimiter, false)
}

func BenchmarkRedis_SlidingWindow_MultiKey(b *testing.B) {
	benchmarkRedisSequential(b, algorithms.NewSlidingWindowLimiter, true)
}

func BenchmarkRedis_SlidingWindow_DeniedSingleKey(b *testing.B) {
	benchmarkRedisDenied(b, algorithms.NewSlidingWindowLimiter)
}

func BenchmarkRedis_SlidingWindow_ParallelSingleKey(b *testing.B) {
	benchmarkRedisParallel(b, algorithms.NewSlidingWindowLimiter, false)
}

func BenchmarkRedis_SlidingWindow_ParallelMultiKey(b *testing.B) {
	benchmarkRedisParallel(b, algorithms.NewSlidingWindowLimiter, true)
}

func BenchmarkRedis_TokenBucket_SingleKey(b *testing.B) {
	benchmarkRedisSequential(b, algorithms.NewTokenBucketLimiter, false)
}

func BenchmarkRedis_TokenBucket_MultiKey(b *testing.B) {
	benchmarkRedisSequential(b, algorithms.NewTokenBucketLimiter, true)
}

func BenchmarkRedis_TokenBucket_DeniedSingleKey(b *testing.B) {
	benchmarkRedisDenied(b, algorithms.NewTokenBucketLimiter)
}

func BenchmarkRedis_TokenBucket_ParallelSingleKey(b *testing.B) {
	benchmarkRedisParallel(b, algorithms.NewTokenBucketLimiter, false)
}

func BenchmarkRedis_TokenBucket_ParallelMultiKey(b *testing.B) {
	benchmarkRedisParallel(b, algorithms.NewTokenBucketLimiter, true)
}

func BenchmarkRedis_LeakyBucket_SingleKey(b *testing.B) {
	benchmarkRedisSequential(b, algorithms.NewLeakyBucketLimiter, false)
}

func BenchmarkRedis_LeakyBucket_MultiKey(b *testing.B) {
	benchmarkRedisSequential(b, algorithms.NewLeakyBucketLimiter, true)
}

func BenchmarkRedis_LeakyBucket_DeniedSingleKey(b *testing.B) {
	benchmarkRedisDenied(b, algorithms.NewLeakyBucketLimiter)
}

func BenchmarkRedis_LeakyBucket_ParallelSingleKey(b *testing.B) {
	benchmarkRedisParallel(b, algorithms.NewLeakyBucketLimiter, false)
}

func BenchmarkRedis_LeakyBucket_ParallelMultiKey(b *testing.B) {
	benchmarkRedisParallel(b, algorithms.NewLeakyBucketLimiter, true)
}
