package algorithms

import (
	"sync"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func TestTokenBucketLimiter_Allow_Concurrency(t *testing.T) {
	cfg := core.Config{
		Limit:  100,
		Window: time.Minute,
	}
	limiter := NewTokenBucketLimiter(cfg)
	key := "concurrent-user"

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// 200 concurrent requests, limit 100
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := limiter.Allow(key)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if allowed {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successCount != 100 {
		t.Errorf("expected exactly 100 allowed, got %d", successCount)
	}
}
