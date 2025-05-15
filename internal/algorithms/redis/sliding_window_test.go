package redis

import (
	"context"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	cfg := core.Config{
		Limit:    2,
		Window:   120 * time.Millisecond,
		RedisURL: redisURL(),
	}
	limiter := NewSlidingWindowLimiter(cfg)
	key := "test-redis-sw"

	client := limiter.(*SlidingWindowLimiter).client
	ctx := context.Background()
	client.Del(ctx, "gorl:sw:"+key)

	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(key)
		if !allowed || err != nil {
			t.Fatalf("expected allowed on attempt %d: err=%v", i+1, err)
		}
	}
	allowed, err := limiter.Allow(key)
	if err != nil {
		t.Fatalf("unexpected error on 3rd request: %v", err)
	}
	if allowed {
		t.Fatal("expected denied on 3rd request")
	}
	time.Sleep(130 * time.Millisecond)
	allowed, err = limiter.Allow(key)
	if !allowed || err != nil {
		t.Fatal("expected allowed after sliding window expires")
	}
}
