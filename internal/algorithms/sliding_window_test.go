package algorithms

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage/inmem"
)

// TestSlidingWindow_Basic validates the fundamental behavior of the Sliding Window algorithm.
// It checks that requests are allowed up to the limit and denied afterwards within the window.
func TestSlidingWindow_Basic(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewSlidingWindowLimiter(core.Config{
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

// TestSlidingWindow_WindowSlide verifies that the window slides correctly,
// allowing expired requests to fall off and new requests to be accepted.
func TestSlidingWindow_WindowSlide(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 2, Window: 100 * time.Millisecond, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	limiter.Allow(ctx, "k")
	limiter.Allow(ctx, "k")

	// Should be denied
	res, _ := limiter.Allow(ctx, "k")
	if res.Allowed {
		t.Fatal("should be denied")
	}

	// Wait for window to pass
	time.Sleep(150 * time.Millisecond)
	res, _ = limiter.Allow(ctx, "k")
	if !res.Allowed {
		t.Fatal("should be allowed after window slide")
	}
}

// TestSlidingWindow_DifferentKeys ensures isolation between different keys.
func TestSlidingWindow_DifferentKeys(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewSlidingWindowLimiter(core.Config{
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

func TestSlidingWindow_FailOpen(t *testing.T) {
	store := &failingStore{}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_FailClosed(t *testing.T) {
	store := &failingStore{}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny with error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_MetricsRecording(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	m := &mockMetrics{}
	limiter := NewSlidingWindowLimiter(core.Config{
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
}

func TestSlidingWindow_ResultMetadata(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 2, Window: 200 * time.Millisecond, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	first, err := limiter.Allow(ctx, "meta")
	if err != nil {
		t.Fatalf("unexpected error on first request: %v", err)
	}
	if !first.Allowed {
		t.Fatal("first request should be allowed")
	}
	if first.Remaining != 1 {
		t.Fatalf("expected remaining=1, got %d", first.Remaining)
	}
	if first.Reset <= 0 {
		t.Fatalf("expected positive reset, got %v", first.Reset)
	}

	second, _ := limiter.Allow(ctx, "meta")
	if second.Remaining != 0 {
		t.Fatalf("expected remaining=0 after second request, got %d", second.Remaining)
	}

	denied, err := limiter.Allow(ctx, "meta")
	if err != nil {
		t.Fatalf("unexpected error on denied request: %v", err)
	}
	if denied.Allowed {
		t.Fatal("third request should be denied")
	}
	if denied.RetryAfter <= 0 {
		t.Fatalf("expected positive retry_after, got %v", denied.RetryAfter)
	}
}

// --- Set Error Handling ---

func TestSlidingWindow_SetError_FailOpen(t *testing.T) {
	store := newFailOnSetStore()
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	// First call hits initialization Set errors in fail-open mode
	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow despite Set error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_SetError_FailClosed(t *testing.T) {
	store := newFailOnSetStore()
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny on Set error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_WindowAdvance_SetError_FailOpen(t *testing.T) {
	// Use a store that tracks state but fails on Set during window advancement
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 3}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 10, Window: 50 * time.Millisecond, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	// First request succeeds (Sets < failAfter)
	limiter.Allow(ctx, "key")

	// Wait for window to advance
	time.Sleep(100 * time.Millisecond)

	// Second request triggers window advancement which leads to Set errors
	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_WindowAdvance_SetError_FailClosed(t *testing.T) {
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 3}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 10, Window: 50 * time.Millisecond, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	limiter.Allow(ctx, "key")
	time.Sleep(100 * time.Millisecond)

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny on Set error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_SecondInitSetError_FailOpen(t *testing.T) {
	// Second and third init Sets fail
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 1}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_ThirdInitSetError_FailOpen(t *testing.T) {
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 2}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_IncrError_FailOpen(t *testing.T) {
	// Store that fails on Incr but succeeds on Get/Set
	store := &incrFailStore{data: make(map[string]float64)}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	// First call initializes, second triggers Incr error
	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow despite Incr error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_IncrError_FailClosed(t *testing.T) {
	store := &incrFailStore{data: make(map[string]float64)}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny, got allowed=%v err=%v", res.Allowed, err)
	}
}

// --- Window advance Set error coverage ---

func TestSlidingWindow_AdvancePrevKeySetError(t *testing.T) {
	// Init uses 3 Sets. Window advance Get(currKey) succeeds, then Set(prevKey) fails at call 4
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 3}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 10, Window: 50 * time.Millisecond, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()
	limiter.Allow(ctx, "k")
	time.Sleep(100 * time.Millisecond)
	res, err := limiter.Allow(ctx, "k")
	if res.Allowed || err == nil {
		t.Fatalf("expected error from prevKey Set, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_AdvanceCurrKeySetError(t *testing.T) {
	// Set(prevKey) succeeds at call 4, Set(currKey) fails at call 5
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 4}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 10, Window: 50 * time.Millisecond, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()
	limiter.Allow(ctx, "k")
	time.Sleep(100 * time.Millisecond)
	res, err := limiter.Allow(ctx, "k")
	if res.Allowed || err == nil {
		t.Fatalf("expected error from currKey Set, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_AdvanceTsKeySetError(t *testing.T) {
	// Set(currKey) succeeds at call 5, Set(tsKey) fails at call 6
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 5}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 10, Window: 50 * time.Millisecond, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()
	limiter.Allow(ctx, "k")
	time.Sleep(100 * time.Millisecond)
	res, err := limiter.Allow(ctx, "k")
	if res.Allowed || err == nil {
		t.Fatalf("expected error from tsKey Set, got allowed=%v err=%v", res.Allowed, err)
	}
}

// --- Get error paths ---

func TestSlidingWindow_GetPrevCountError(t *testing.T) {
	// Use a store that fails Get after a certain number of calls
	store := &getFailAfterNStore{data: make(map[string]float64), failAfter: 1}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	// First Get(tsKey) succeeds, second Get(prevKey) fails
	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("expected Get error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_GetCurrCountError(t *testing.T) {
	store := &getFailAfterNStore{data: make(map[string]float64), failAfter: 2}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("expected Get error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_AdvanceGetCurrKeyError(t *testing.T) {
	// Init: 1 Get(tsKey). Window advance: 1 Get(tsKey) + 1 Get(currKey).
	// So total 3 Gets: failAfter:2 causes Get(currKey) in advance to fail.
	store := &getFailAfterNStore{data: make(map[string]float64), failAfter: 2}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 10, Window: 50 * time.Millisecond, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	limiter.Allow(ctx, "k") // uses 1 Get
	time.Sleep(100 * time.Millisecond)
	// Second request: Get(tsKey) at call #2 succeeds, Get(currKey) at call #3 fails
	res, err := limiter.Allow(ctx, "k")
	if res.Allowed || err == nil {
		t.Fatalf("expected Get(currKey) error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_MultipleRequestsSameWindow(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: 5 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	// First request initializes the window
	res1, _ := limiter.Allow(ctx, "k")
	if !res1.Allowed {
		t.Fatal("first request should be allowed")
	}

	// Second request within same window (enters else branch, elapsed < window)
	res2, _ := limiter.Allow(ctx, "k")
	if !res2.Allowed {
		t.Fatal("second request in same window should be allowed")
	}

	// Third request
	res3, _ := limiter.Allow(ctx, "k")
	if !res3.Allowed {
		t.Fatal("third request in same window should be allowed")
	}
}

func TestSlidingWindow_GetPrevCountErrorInSameWindow(t *testing.T) {
	// Init: 1 Get(tsKey). Same window: 1 Get(tsKey) + 1 Get(prevKey).
	// failAfter:2 → 3rd Get (prevKey) fails
	store := &getFailAfterNStore{data: make(map[string]float64), failAfter: 2}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: 5 * time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	limiter.Allow(ctx, "k")                 // 1 Get(tsKey) + init
	res, err := limiter.Allow(ctx, "k") 	// 1 Get(tsKey) succeeds, 1 Get(prevKey) fails
	if res.Allowed || err == nil {
		t.Fatalf("expected Get(prevKey) error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_GetCurrCountErrorInSameWindow(t *testing.T) {
	// failAfter:3 → 4th Get (currKey in count load phase) fails
	store := &getFailAfterNStore{data: make(map[string]float64), failAfter: 3}
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: 5 * time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	limiter.Allow(ctx, "k")                 // 1 Get(tsKey), init path
	res, err := limiter.Allow(ctx, "k") 	// 1 Get(tsKey), 1 Get(prevKey) ok, 1 Get(currKey) fails
	if res.Allowed || err == nil {
		t.Fatalf("expected Get(currKey) error, got allowed=%v err=%v", res.Allowed, err)
	}
}

func TestSlidingWindow_Close(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	if err := limiter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func BenchmarkSlidingWindow_SingleKey(b *testing.B) {
	b.ReportAllocs()
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 100000, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench")
	}
}

func BenchmarkSlidingWindow_MultiKey(b *testing.B) {
	b.ReportAllocs()
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewSlidingWindowLimiter(core.Config{
		Limit: 100000, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(ctx, key)
	}
}
