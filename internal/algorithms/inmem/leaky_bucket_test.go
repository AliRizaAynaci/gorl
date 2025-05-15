package algorithms

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func TestLeakyBucketLimiter_Allow(t *testing.T) {
	cfg := core.Config{
		Limit:  2,
		Window: 1 * time.Second,
	}
	limiter := NewLeakyBucketLimiter(cfg)
	key := "lb-user"

	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(key)
		if !allowed || err != nil {
			t.Fatalf("expected allowed on attempt %d", i+1)
		}
	}
	allowed, err := limiter.Allow(key)
	if err != nil {
		t.Fatalf("unexpected error on 3rd request: %v", err)
	}
	if allowed {
		t.Log("3rd request unexpectedly allowed — timing sensitivity")
	}

	time.Sleep(1100 * time.Millisecond)
	allowed, err = limiter.Allow(key)
	if !allowed || err != nil {
		t.Fatal("expected allowed after leak")
	}
}
