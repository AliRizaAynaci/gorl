package algorithms

import (
	"sync"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func TestLeakyBucketLimiter_Allow_Concurrency(t *testing.T) {
	cfg := core.Config{
		Limit:  80,
		Window: time.Minute,
	}
	limiter := NewLeakyBucketLimiter(cfg)
	key := "concurrent-lb-user"

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 160; i++ {
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

	maxAllowed := 80
	tolerance := 2

	if successCount < maxAllowed || successCount > maxAllowed+tolerance {
		t.Errorf("expected allowed between %d and %d, got %d", maxAllowed, maxAllowed+tolerance, successCount)
	}
}
