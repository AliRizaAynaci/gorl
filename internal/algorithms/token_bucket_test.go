package algorithms

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage/inmem"
)

// TestTokenBucket_Basic validates elementary correct behavior of the Token Bucket algorithm.
// It ensures requests within the limit consume tokens and are allowed.
func TestTokenBucket_Basic(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewTokenBucketLimiter(core.Config{
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

// TestTokenBucket_Refill verifies that tokens are refilled over time according to the rate.
// It exhausts tokens, waits for partial refill, and checks allowance.
func TestTokenBucket_Refill(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	// Window (2s) >> sleep (200ms) so keys don't expire.
	// limit=10, tpt=200ms → after 200ms sleep, newTokens = 200ms/200ms = 1.
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 10, Window: 2 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	// Exhaust all 10 tokens
	for i := 0; i < 10; i++ {
		limiter.Allow(ctx, "k")
	}
	res, _ := limiter.Allow(ctx, "k")
	if res.Allowed {
		t.Fatal("should be denied after exhausting tokens")
	}

	// Wait enough for at least 1 token to be refilled (tpt = 200ms)
	time.Sleep(250 * time.Millisecond)
	res, _ = limiter.Allow(ctx, "k")
	if !res.Allowed {
		t.Fatal("should be allowed after refill")
	}
}

// TestTokenBucket_SmallTimePerToken ensures the algorithm handles cases where timePerToken is extremely small (high throughput).
func TestTokenBucket_SmallTimePerToken(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	// Very large limit with very small window → tpt could be 0, clamped to 1
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 1000000, Window: time.Nanosecond, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed {
		t.Fatal("first request should be allowed")
	}
}

// TestTokenBucket_DifferentKeys verifies that token buckets for different keys are independent.
func TestTokenBucket_DifferentKeys(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewTokenBucketLimiter(core.Config{
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

// TestTokenBucket_FailOpen ensures requests pass when storage fails if FailOpen is true.
func TestTokenBucket_FailOpen(t *testing.T) {
	store := &failingStore{}
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestTokenBucket_FailClosed ensures requests are denied when storage fails if FailOpen is false.
func TestTokenBucket_FailClosed(t *testing.T) {
	store := &failingStore{}
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny with error, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestTokenBucket_SetError_FailOpen tests behavior when setting the token count fails, with FailOpen enabled.
func TestTokenBucket_SetError_FailOpen(t *testing.T) {
	store := newFailOnSetStore()
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow despite Set error, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestTokenBucket_SetError_FailClosed tests behavior when setting the token count fails, with FailOpen disabled.
func TestTokenBucket_SetError_FailClosed(t *testing.T) {
	store := newFailOnSetStore()
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny on Set error, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestTokenBucket_SecondSetError_FailOpen tests fail-open logic when the second Set operation (refillKey) fails.
func TestTokenBucket_SecondSetError_FailOpen(t *testing.T) {
	// First Set (tokensKey) succeeds, second Set (refillKey) fails
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 1}
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: true,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if !res.Allowed || err != nil {
		t.Fatalf("fail-open should allow, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestTokenBucket_SecondSetError_FailClosed tests fail-closed logic when the second Set operation (refillKey) fails.
func TestTokenBucket_SecondSetError_FailClosed(t *testing.T) {
	store := &setFailAfterNStore{data: make(map[string]float64), failAfter: 1}
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("fail-closed should deny, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestTokenBucket_GetRefillError verifies error handling when fetching the refill timestamp fails.
func TestTokenBucket_GetRefillError(t *testing.T) {
	store := &getFailAfterNStore{data: make(map[string]float64), failAfter: 1}
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{}, FailOpen: false,
	}, store)
	ctx := context.Background()

	res, err := limiter.Allow(ctx, "key")
	if res.Allowed || err == nil {
		t.Fatalf("expected Get error, got allowed=%v err=%v", res.Allowed, err)
	}
}

// TestTokenBucket_TokensCappedToLimit ensures that refilled tokens never exceed the configured limit,
// even if a long time has passed since the last request.
func TestTokenBucket_TokensCappedToLimit(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	// Using a configuration where heavy refilling occurs.
	// limit=3, window=3s -> fill rate = 1 token/s.
	limiter2 := NewTokenBucketLimiter(core.Config{
		Limit: 3, Window: 3 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	// Init: tokens=3, use 1, stored tokens=2
	limiter2.Allow(ctx, "cap")

	// Sleep 2s: rate=1/s, so newTokens=2. tokens=2+2=4>3 → cap to 3.
	time.Sleep(2 * time.Second)

	// Should get exactly 3 tokens
	for i := 0; i < 3; i++ {
		res, _ := limiter2.Allow(ctx, "cap")
		if !res.Allowed {
			t.Fatalf("req %d: should be allowed after capped refill", i+1)
		}
	}
	res, _ := limiter2.Allow(ctx, "cap")
	if res.Allowed {
		t.Fatal("should be denied after consuming all capped tokens")
	}
}

// TestTokenBucket_NoRefillSmallElapsed verify that no tokens are added if the elapsed time is less than `timePerToken`.
func TestTokenBucket_NoRefillSmallElapsed(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 2, Window: 5 * time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()

	// Use a token, then immediately try again (no refill should occur)
	res1, _ := limiter.Allow(ctx, "k")
	res2, _ := limiter.Allow(ctx, "k")
	res3, _ := limiter.Allow(ctx, "k")

	if !res1.Allowed || !res2.Allowed {
		t.Fatal("first two should be allowed")
	}
	if res3.Allowed {
		t.Fatal("third should be denied")
	}
}

func TestTokenBucket_ResultMetadata(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewTokenBucketLimiter(core.Config{
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
	if denied.Reset < denied.RetryAfter {
		t.Fatalf("expected reset to be at least retry_after, got reset=%v retry_after=%v", denied.Reset, denied.RetryAfter)
	}
}

func TestTokenBucket_Close(t *testing.T) {
	store := inmem.NewInMemoryStore()
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 5, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	if err := limiter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestTokenBucket_MetricsRecording(t *testing.T) {
	store := inmem.NewInMemoryStore()
	defer store.Close()
	m := &mockMetrics{}
	limiter := NewTokenBucketLimiter(core.Config{
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

func BenchmarkTokenBucket_SingleKey(b *testing.B) {
	b.ReportAllocs()
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 100000, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "bench")
	}
}

func BenchmarkTokenBucket_MultiKey(b *testing.B) {
	b.ReportAllocs()
	store := inmem.NewInMemoryStore()
	defer store.Close()
	limiter := NewTokenBucketLimiter(core.Config{
		Limit: 100000, Window: time.Second, Metrics: &core.NoopMetrics{},
	}, store)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user-%d", i%1000)
		limiter.Allow(ctx, key)
	}
}
