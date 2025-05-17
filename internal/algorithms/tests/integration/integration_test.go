package integration

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/internal/algorithms/tests/common"
	"github.com/AliRizaAynaci/gorl/storage"
	redisstore "github.com/AliRizaAynaci/gorl/storage/redis"
)

// TestAlgorithmsWithRedis verifies that all strategies work correctly
func TestAlgorithmsWithRedis(t *testing.T) {
	backends := map[string]func() storage.Storage{
		"redis": func() storage.Storage {
			return redisstore.NewRedisStore("redis://localhost:6379/0")
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
				cfg := core.Config{
					Limit:  3,
					Window: 1 * time.Second,
				}
				limiter := s.constructor(cfg, store)
				common.CommonLimiterBehavior(t, limiter, "test-user", 3)
			})
		}
	}
}
