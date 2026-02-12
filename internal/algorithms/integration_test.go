package algorithms_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/v2/storage"
	redisstore "github.com/AliRizaAynaci/gorl/v2/storage/redis"
)

// verifyLimits checks if the limiter allows exactly 'limit' requests and denies the next one.
func verifyLimits(t *testing.T, limiter core.Limiter, key string, limit int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < limit; i++ {
		res, err := limiter.Allow(ctx, key)
		if err != nil || !res.Allowed {
			t.Fatalf("expected allowed, got %v, err %v (req %d)", res.Allowed, err, i+1)
		}
	}
	res, err := limiter.Allow(ctx, key)
	if res.Allowed || err != nil {
		t.Fatalf("expected denied after limit, got %v, err %v", res.Allowed, err)
	}
}

// TestAlgorithmsWithRedis verifies that all strategies work correctly with Redis storage.
func TestAlgorithmsWithRedis(t *testing.T) {
	// Check if Redis is available either via Env or default localhost
	redisURL := "redis://127.0.0.1:6379/0"
	if url := os.Getenv("GORL_REDIS_URL"); url != "" {
		redisURL = url
	}

	// Try to connect to Redis, skip if unavailable
	_, err := redisstore.NewRedisStore(redisURL)
	if err != nil {
		t.Skipf("skipping redis integration tests: %v", err)
	}

	backends := map[string]func() storage.Storage{
		"redis": func() storage.Storage {
			store, err := redisstore.NewRedisStore(redisURL)
			if err != nil {
				t.Fatalf("failed to create redis store: %v", err)
			}
			return store
		},
		// future backends can be added here
	}

	strategies := []struct {
		name        string
		constructor func(core.Config, storage.Storage) core.Limiter
	}{
		{"FixedWindow", algorithms.NewFixedWindowLimiter},
		{"SlidingWindow", algorithms.NewSlidingWindowLimiter},
		{"TokenBucket", algorithms.NewTokenBucketLimiter},
		{"LeakyBucket", algorithms.NewLeakyBucketLimiter},
	}

	for beName, storeFn := range backends {
		for _, s := range strategies {
			t.Run(beName+"/"+s.name, func(t *testing.T) {
				store := storeFn()
				defer store.Close() // Ensure store is closed
				
				cfg := core.Config{
					Limit:   3,
					Window:  1 * time.Second,
					Metrics: &core.NoopMetrics{},
				}
				limiter := s.constructor(cfg, store)
				verifyLimits(t, limiter, "test-user-integ", 3)
			})
		}
	}
}
