package algorithms

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func TestTokenBucketLimiter_Allow(t *testing.T) {
	cfg := core.Config{
		Limit:  3,
		Window: time.Second,
	}
	limiter := NewTokenBucketLimiter(cfg)

	key := "user-1"
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(key)
		if !allowed || err != nil {
			t.Fatalf("expected allowed on attempt %d", i+1)
		}
	}
	allowed, err := limiter.Allow(key)
	if err != nil {
		t.Fatalf("unexpected error on 4th request: %v", err)
	}
	if allowed {
		t.Fatal("expected denied on 4th request")
	}

}
