package algorithms

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage/inmem"
)

// TestFixedWindow_Basic validates the basic rate limiting functionality of the Fixed Window algorithm.
// It ensures that requests within the limit are allowed and requests exceeding the limit are denied.
func TestFixedWindow_Basic(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 3, Window: 2 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		res, err := limiter.Allow(ctx, "user-1")
		if err != nil || !res.Allowed {
			t.Fatalf("req %d: expected allowed, got %v, err %v", i+1, res.Allowed, err)
		}
	}
	res, err := limiter.Allow(ctx, "user-1")
	if res.Allowed || err != nil {
		t.Fatalf("expected denied after limit, got %v, err %v", res.Allowed, err)
	}
}

// TestFixedWindow_DifferentKeys verifies that rate limits are applied independently for different keys.
func TestFixedWindow_DifferentKeys(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 1, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	r1, _ := limiter.Allow(ctx, "user-a")
	r2, _ := limiter.Allow(ctx, "user-b")
	r3, _ := limiter.Allow(ctx, "user-a")

	if !r1.Allowed || !r2.Allowed {
		t.Fatal("first request for each key should be allowed")
	}
	if r3.Allowed {
		t.Fatal("second request for user-a should be denied")
	}
}

// TestFixedWindow_WindowReset ensures that the rate limit counter resets after the window duration passes.
func TestFixedWindow_WindowReset(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 1, Window: 100 * time.Millisecond, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	limiter.Allow(ctx, "key")
	res, _ := limiter.Allow(ctx, "key")
	if res.Allowed {
		t.Fatal("should be denied within window")
	}

	time.Sleep(150 * time.Millisecond)
	res, _ = limiter.Allow(ctx, "key")
	if !res.Allowed {
		t.Fatal("should be allowed after window reset")
	}
}

// TestFixedWindow_FailOpen verifies that the limiter allows requests when the storage backend fails
// and FailOpen is set to true.
func TestFixedWindow_FailOpen(t *testing.T) {
	store := &failingStore{}
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestFixedWindow_FailClosed verifies that the limiter denies requests and returns an error
// when the storage backend fails and FailOpen is set to false.
func TestFixedWindow_FailClosed(t *testing.T) {
	store := &failingStore{}
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny with error, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestFixedWindow_Concurrency tests the limiter's behavior under concurrent load to ensure race conditions are handled.
func TestFixedWindow_Concurrency(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 10, Window: 2 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	var wg sync.WaitGroup
	var count int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if res, _ := limiter.Allow(ctx, "c"); res.Allowed {
				atomic.AddInt32(&count, 1)
			}
		}()
	}
	wg.Wait()
	if count != 10 {
		t.Errorf("expected exactly 10 allowed, got %d", count)
	}
}

// TestFixedWindow_Close checks if the Close method cleans up resources correctly.
func TestFixedWindow_Close(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	if err := limiter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// TestFixedWindow_MetricsRecording verifies that metrics are correctly recorded for allowed and denied requests.
func TestFixedWindow_MetricsRecording(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	m := &mockMetrics{}
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 1, Window: time.Second, Metrics: m,
	}, store)
	ctx := context.Background()

	limiter.Allow(ctx, "k")
	limiter.Allow(ctx, "k")

	if m.allows != 1 {
		t.Errorf("expected 1 allow metric, got %d", m.allows)
	}
	if m.denies != 1 {
		t.Errorf("expected 1 deny metric, got %d", m.denies)
	}
	if m.latencies != 2 {
		t.Errorf("expected 2 latency observations, got %d", m.latencies)
	}
}

func TestFixedWindow_ResultMetadata(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 1, Window: 200 * time.Millisecond, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "meta")
	if err != nil {
		t.Fatalf("unexpected error on allowed request: %v", err)
	}
	if !allowed.Allowed {
		t.Fatal("first request should be allowed")
	}
	if allowed.Remaining != 0 {
		t.Fatalf("expected remaining=0 after consuming capacity, got %d", allowed.Remaining)
	}
	if allowed.Reset <= 0 {
		t.Fatalf("expected positive reset, got %v", allowed.Reset)
	}

	denied, err := limiter.Allow(ctx, "meta")
	if err != nil {
		t.Fatalf("unexpected error on denied request: %v", err)
	}
	if denied.Allowed {
		t.Fatal("second request should be denied")
	}
	if denied.RetryAfter <= 0 {
		t.Fatalf("expected positive retry_after, got %v", denied.RetryAfter)
	}
}

// BenchmarkFixedWindow_SingleKey benchmarks the performance of the Fixed Window limiter with a single key.
func BenchmarkFixedWindow_SingleKey(b *testing.B) {
	b.ReportAllocs()
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 100000, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench")
	}
}

func BenchmarkFixedWindow_MultiKey(b *testing.B) {
	b.ReportAllocs()
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewFixedWindowLimiter(core.Config{
		Limit: 100000, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(ctx, key)
	}
}
