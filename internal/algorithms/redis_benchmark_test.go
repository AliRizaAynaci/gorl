package algorithms_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/v2/storage"
	redisstore "github.com/AliRizaAynaci/gorl/v2/storage/redis"
)

func newRedisStore(b *testing.B) storage.Storage {
	url := "redis://127.0.0.1:6379/0"
	if u := os.Getenv("GORL_REDIS_URL"); u != "" {
		url = u
	}
	store, err := redisstore.NewRedisStore(url)
	if err != nil {
		b.Skipf("skipping redis benchmark: %v", err)
	}
	return store
}

// Fixed Window
func BenchmarkRedis_FixedWindow_SingleKey(b *testing.B) {
	store := newRedisStore(b)
	// Do not defer store.Close() in benchmarks to simulate long running app? 
	// Or close at end? Redis connection pool should be kept open.
	// b.StopTimer() ... Close ...
	// Usually just letting it be is fine for benchmark unless connection limit hit.
	// But sharing same store instance is better.
	defer store.Close()

	limiter := algorithms.NewFixedWindowLimiter(core.Config{
		Limit: 1000000, Window: time.Hour, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench-redis-fx")
	}
}

func BenchmarkRedis_FixedWindow_MultiKey(b *testing.B) {
	b.ReportAllocs()
	store := newRedisStore(b)
	defer store.Close()
	limiter := algorithms.NewFixedWindowLimiter(core.Config{
		Limit: 1000000, Window: time.Hour, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-redis-fx-%d", i%1000)
		limiter.Allow(ctx, key)
	}
}

// Sliding Window
func BenchmarkRedis_SlidingWindow_SingleKey(b *testing.B) {
	store := newRedisStore(b)
	defer store.Close()
	limiter := algorithms.NewSlidingWindowLimiter(core.Config{
		Limit: 1000000, Window: time.Hour, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench-redis-sw")
	}
}

func BenchmarkRedis_SlidingWindow_MultiKey(b *testing.B) {
	b.ReportAllocs()
	store := newRedisStore(b)
	defer store.Close()
	limiter := algorithms.NewSlidingWindowLimiter(core.Config{
		Limit: 1000000, Window: time.Hour, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-redis-sw-%d", i%1000)
		limiter.Allow(ctx, key)
	}
}

// Token Bucket
func BenchmarkRedis_TokenBucket_SingleKey(b *testing.B) {
	store := newRedisStore(b)
	defer store.Close()
	limiter := algorithms.NewTokenBucketLimiter(core.Config{
		Limit: 1000000, Window: time.Hour, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench-redis-tb")
	}
}

func BenchmarkRedis_TokenBucket_MultiKey(b *testing.B) {
	b.ReportAllocs()
	store := newRedisStore(b)
	defer store.Close()
	limiter := algorithms.NewTokenBucketLimiter(core.Config{
		Limit: 1000000, Window: time.Hour, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-redis-tb-%d", i%1000)
		limiter.Allow(ctx, key)
	}
}

// Leaky Bucket
func BenchmarkRedis_LeakyBucket_SingleKey(b *testing.B) {
	store := newRedisStore(b)
	defer store.Close()
	limiter := algorithms.NewLeakyBucketLimiter(core.Config{
		Limit: 1000000, Window: time.Hour, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench-redis-lb")
	}
}

func BenchmarkRedis_LeakyBucket_MultiKey(b *testing.B) {
	b.ReportAllocs()
	store := newRedisStore(b)
	defer store.Close()
	limiter := algorithms.NewLeakyBucketLimiter(core.Config{
		Limit: 1000000, Window: time.Hour, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-redis-lb-%d", i%1000)
		limiter.Allow(ctx, key)
	}
}
