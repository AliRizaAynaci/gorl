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

// TestLeakyBucket_Basic verifies fundamental Allow/Deny behavior based on bucket capacity.
func TestLeakyBucket_Basic(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewLeakyBucketLimiter(core.Config{
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
		t.Fatalf("expected denied, got %v, err %v", res.Allowed, err)
	}
}

// TestLeakyBucket_Leak verifies that the bucket leaks (frees up capacity) over time.
func TestLeakyBucket_Leak(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	// Window (2s) >> sleep (200ms) so keys don't expire.
	// limit=5 → leakRate = 5/2s = 2.5/s, leakInterval = 400ms.
	// After 200ms, leaked = 200ms / 400ms = 0.5 -> 0.
	// Use limit=10, window=2s: leakInterval = 200ms. After 250ms: leaked = 1.
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 10, Window: 2 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	// Fill the bucket
	for i := 0; i < 10; i++ {
		limiter.Allow(ctx, "k")
	}
	res, _ := limiter.Allow(ctx, "k")
	if res.Allowed {
		t.Fatal("should be denied after filling bucket")
	}

	// Wait for some water to leak
	time.Sleep(250 * time.Millisecond)
	res, _ = limiter.Allow(ctx, "k")
	if !res.Allowed {
		t.Fatal("should be allowed after leak")
	}
}

// TestLeakyBucket_DifferentKeys ensures isolation between rate limits of different keys.
func TestLeakyBucket_DifferentKeys(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewLeakyBucketLimiter(core.Config{
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

func TestLeakyBucket_FailOpen(t *testing.T) {
	store := &failingStore{}
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestLeakyBucket_FailClosed(t *testing.T) {
	store := &failingStore{}
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny with error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestLeakyBucket_SetError_FailOpen(t *testing.T) {
	store := newFailOnSetStore()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow despite Set error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestLeakyBucket_SetError_FailClosed(t *testing.T) {
	store := newFailOnSetStore()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny on Set error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestLeakyBucket_SecondSetError_FailOpen(t *testing.T) {
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 1}
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestLeakyBucket_SecondSetError_FailClosed(t *testing.T) {
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 1}
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestLeakyBucket_GetLeakError verifies performance when retrieving the last leak time fails.
func TestLeakyBucket_GetLeakError(t *testing.T) {
	store := &getFailAfterNStore{data: make(map[string]float64), failAfter: 1}
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("expected Get error, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestLeakyBucket_WaterLevelFloor ensures that the water level is correctly reset to 0
// if the calculated leak amount exceeds the current level.
func TestLeakyBucket_WaterLevelFloor(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	// Window (5s) >> sleep so keys don't expire.
	// limit=3, leakInterval = 5s/3 ≈ 1.67s
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 3, Window: 5 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	// Use 1 token
	limiter.Allow(ctx, "k")

	// Wait for water to leak to 0.
	time.Sleep(2 * time.Second)

	// Should be able to use all 3 tokens
	for i := 0; i < 3; i++ {
		res, _ := limiter.Allow(ctx, "k")
		if !res.Allowed {
			t.Fatalf("req %d: should be allowed after full leak", i+1)
		}
	}
}

func TestLeakyBucket_Concurrency(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 10, Window: 2 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	var wg sync.WaitGroup
	var count int32
	for i := 0; i < 50; i++ {
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

func TestLeakyBucket_Close(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	if err := limiter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestLeakyBucket_MetricsRecording(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	m := &mockMetrics{}
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 1, Window: time.Second, Metrics: m,
	}, store)
	ctx := context.Background()

	limiter.Allow(ctx, "k")
	limiter.Allow(ctx, "k")

	if m.allows != 1 {
		t.Errorf("expected 1 allow, got %d", m.allows)
	}
	if m.denies != 1 {
		t.Errorf("expected 1 deny, got %d", m.denies)
	}
}

func BenchmarkLeakyBucket_SingleKey(b *testing.B) {
	b.ReportAllocs()
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 100000, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench")
	}
}

func BenchmarkLeakyBucket_MultiKey(b *testing.B) {
	b.ReportAllocs()
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewLeakyBucketLimiter(core.Config{
		Limit: 100000, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(ctx, key)
	}
}
