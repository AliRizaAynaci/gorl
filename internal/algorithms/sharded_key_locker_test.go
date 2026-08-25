package algorithms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/storage"
	"github.com/AliRizaAynaci/gorl/v2/storage/inmem"
)

type statefulLimiterFactory func(core.Config, storage.Storage) core.Limiter

var statefulLimiterCases = []struct {
	name    string
	factory statefulLimiterFactory
}{
	{name: "sliding_window", factory: NewSlidingWindowLimiter},
	{name: "token_bucket", factory: NewTokenBucketLimiter},
	{name: "leaky_bucket", factory: NewLeakyBucketLimiter},
}

type limiterCallResult struct {
	result core.Result
	err    error
}

func TestGenericStatefulLimiters_ConcurrentSameKeyHonorsLimit(t *testing.T) {
	const (
		limit      = 50
		goroutines = 500
	)

	for _, testCase := range statefulLimiterCases {
		t.Run(testCase.name, func(t *testing.T) {
			limiter := testCase.factory(core.Config{
				Limit: limit, Window: time.Hour, Metrics: &core.NoopMetrics{},
			}, inmem.NewInMemoryStore())
			t.Cleanup(func() {
				if err := limiter.Close(); err != nil {
					t.Errorf("close limiter: %v", err)
				}
			})

			start := make(chan struct{})
			results := make(chan limiterCallResult, goroutines)
			for i := 0; i < goroutines; i++ {
				go func() {
					<-start
					result, err := limiter.Allow(context.Background(), "shared-key")
					results <- limiterCallResult{result: result, err: err}
				}()
			}
			close(start)

			allowed := 0
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			for i := 0; i < goroutines; i++ {
				select {
				case call := <-results:
					if call.err != nil {
						t.Fatalf("Allow returned an error: %v", call.err)
					}
					if call.result.Allowed {
						allowed++
					}
				case <-timer.C:
					t.Fatalf("timed out after receiving %d/%d results", i, goroutines)
				}
			}

			if allowed != limit {
				t.Fatalf("allowed %d concurrent requests, want exactly %d", allowed, limit)
			}
		})
	}
}

type blockingGetStore struct {
	storage.Storage
	blockedKey string
	entered    chan struct{}
	release    chan struct{}
	enterOnce  sync.Once
}

func (s *blockingGetStore) Get(ctx context.Context, key string) (float64, error) {
	if strings.Contains(key, "{"+s.blockedKey+"}") {
		s.enterOnce.Do(func() { close(s.entered) })
		select {
		case <-s.release:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return s.Storage.Get(ctx, key)
}

func TestGenericStatefulLimiters_DifferentShardsProgressIndependently(t *testing.T) {
	for _, testCase := range statefulLimiterCases {
		t.Run(testCase.name, func(t *testing.T) {
			const blockedKey = "blocked-key"
			store := &blockingGetStore{
				Storage:    inmem.NewInMemoryStore(),
				blockedKey: blockedKey,
				entered:    make(chan struct{}),
				release:    make(chan struct{}),
			}
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(store.release) }) }

			limiter := testCase.factory(core.Config{
				Limit: 10, Window: time.Hour, Metrics: &core.NoopMetrics{},
			}, store)
			t.Cleanup(func() {
				release()
				if err := limiter.Close(); err != nil {
					t.Errorf("close limiter: %v", err)
				}
			})

			freeKey := keyOnDifferentShard(t, limiter, blockedKey)
			blockedDone := callLimiterAsync(limiter, blockedKey)
			select {
			case <-store.entered:
			case <-time.After(time.Second):
				t.Fatal("blocked key did not enter storage")
			}

			select {
			case call := <-callLimiterAsync(limiter, freeKey):
				if call.err != nil {
					t.Fatalf("different-key Allow returned an error: %v", call.err)
				}
				if !call.result.Allowed {
					t.Fatal("different-key Allow was unexpectedly denied")
				}
			case <-time.After(time.Second):
				t.Fatal("different-shard key was blocked by an unrelated request")
			}

			release()
			select {
			case call := <-blockedDone:
				if call.err != nil {
					t.Fatalf("blocked-key Allow returned an error after release: %v", call.err)
				}
			case <-time.After(time.Second):
				t.Fatal("blocked-key Allow did not finish after release")
			}
		})
	}
}

type failFirstGetStore struct {
	storage.Storage
	failed atomic.Bool
}

// blockingScriptStore stalls the Lua path for one key so a test can observe
// whether an unrelated request is held up behind it.
type blockingScriptStore struct {
	storage.Storage
	blockedKey string
	entered    chan struct{}
	release    chan struct{}
	enterOnce  sync.Once
}

func (s *blockingScriptStore) EvalScript(
	ctx context.Context,
	_ string,
	keys []string,
	_ ...int64,
) ([]int64, error) {
	if len(keys) > 0 && strings.Contains(keys[0], "{"+s.blockedKey+"}") {
		s.enterOnce.Do(func() { close(s.entered) })
		select {
		case <-s.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return []int64{1, 9, 1_000, 0}, nil
}

// TestRedisScriptLimiters_DoNotSerializeOnKeyLocks asserts the behavior that
// matters -- a Redis-script limiter never takes a generic key lock -- rather
// than asserting how that is implemented. Both keys are deliberately mapped to
// the SAME lock shard: if the Redis path ever started acquiring the shard
// mutex, the second call would block behind the first and this test would fail.
func TestRedisScriptLimiters_DoNotSerializeOnKeyLocks(t *testing.T) {
	for _, testCase := range statefulLimiterCases {
		t.Run(testCase.name, func(t *testing.T) {
			const blockedKey = "blocked-key"
			store := &blockingScriptStore{
				Storage:    inmem.NewInMemoryStore(),
				blockedKey: blockedKey,
				entered:    make(chan struct{}),
				release:    make(chan struct{}),
			}
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(store.release) }) }

			limiter := testCase.factory(core.Config{
				Limit: 10, Window: time.Hour, Metrics: &core.NoopMetrics{},
			}, store)
			t.Cleanup(func() {
				release()
				if err := limiter.Close(); err != nil {
					t.Errorf("close limiter: %v", err)
				}
			})

			sameShardKey := keyOnSameShard(t, limiter, blockedKey)
			blockedDone := callLimiterAsync(limiter, blockedKey)
			select {
			case <-store.entered:
			case <-time.After(time.Second):
				t.Fatal("blocked key did not enter the Redis script path")
			}

			select {
			case call := <-callLimiterAsync(limiter, sameShardKey):
				if call.err != nil {
					t.Fatalf("same-shard Allow returned an error: %v", call.err)
				}
				if !call.result.Allowed || call.result.Remaining != 9 {
					t.Fatalf("unexpected Redis-script result: %+v", call.result)
				}
			case <-time.After(time.Second):
				t.Fatal("Redis-script limiter serialized two keys on a shared lock shard")
			}

			release()
			select {
			case call := <-blockedDone:
				if call.err != nil {
					t.Fatalf("blocked-key Allow returned an error after release: %v", call.err)
				}
			case <-time.After(time.Second):
				t.Fatal("blocked-key Allow did not finish after release")
			}
		})
	}
}

func (s *failFirstGetStore) Get(ctx context.Context, key string) (float64, error) {
	if s.failed.CompareAndSwap(false, true) {
		return 0, errors.New("injected Get failure")
	}
	return s.Storage.Get(ctx, key)
}

func TestGenericStatefulLimiters_ErrorReleasesKeyLock(t *testing.T) {
	for _, testCase := range statefulLimiterCases {
		t.Run(testCase.name, func(t *testing.T) {
			limiter := testCase.factory(core.Config{
				Limit: 10, Window: time.Hour, Metrics: &core.NoopMetrics{},
			}, &failFirstGetStore{Storage: inmem.NewInMemoryStore()})
			t.Cleanup(func() {
				if err := limiter.Close(); err != nil {
					t.Errorf("close limiter: %v", err)
				}
			})

			if _, err := limiter.Allow(context.Background(), "retry-key"); err == nil {
				t.Fatal("first Allow succeeded, want injected storage error")
			}

			select {
			case call := <-callLimiterAsync(limiter, "retry-key"):
				if call.err != nil {
					t.Fatalf("second Allow returned an error: %v", call.err)
				}
				if !call.result.Allowed {
					t.Fatal("second Allow was unexpectedly denied")
				}
			case <-time.After(time.Second):
				t.Fatal("second Allow deadlocked after the first call returned an error")
			}
		})
	}
}

func callLimiterAsync(limiter core.Limiter, key string) <-chan limiterCallResult {
	result := make(chan limiterCallResult, 1)
	go func() {
		callResult, err := limiter.Allow(context.Background(), key)
		result <- limiterCallResult{result: callResult, err: err}
	}()
	return result
}

func keyOnDifferentShard(t *testing.T, limiter core.Limiter, blockedKey string) string {
	t.Helper()

	blockedLock := limiterKeyLocker(t, limiter).mutexFor(blockedKey)
	for i := 0; i < keyLockShardCount*2; i++ {
		candidate := fmt.Sprintf("free-key-%d", i)
		if limiterKeyLocker(t, limiter).mutexFor(candidate) != blockedLock {
			return candidate
		}
	}
	t.Fatal("could not find a key on a different lock shard")
	return ""
}

func keyOnSameShard(t *testing.T, limiter core.Limiter, blockedKey string) string {
	t.Helper()

	locker := limiterKeyLocker(t, limiter)
	blockedLock := locker.mutexFor(blockedKey)
	for i := 0; i < keyLockShardCount*keyLockShardCount; i++ {
		candidate := fmt.Sprintf("same-shard-key-%d", i)
		if candidate != blockedKey && locker.mutexFor(candidate) == blockedLock {
			return candidate
		}
	}
	t.Fatal("could not find a key on the same lock shard")
	return ""
}

func limiterKeyLocker(t *testing.T, limiter core.Limiter) *shardedKeyLocker {
	t.Helper()

	switch typed := limiter.(type) {
	case *SlidingWindowLimiter:
		return typed.locks
	case *TokenBucketLimiter:
		return typed.locks
	case *LeakyBucketLimiter:
		return typed.locks
	default:
		t.Fatalf("unsupported limiter type %T", limiter)
		return nil
	}
}
