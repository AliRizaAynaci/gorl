package redis

import (
	"context"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func TestFixedWindowLimiter_Allow(t *testing.T) {
	cfg := core.Config{
		Limit:    2,
		Window:   1 * time.Second,
		RedisURL: redisURL(),
	}
	limiter := NewFixedWindowLimiter(cfg)
	key := "test-redis-fw"

	client := limiter.(*FixedWindowLimiter).client
	ctx := context.Background()
	client.Del(ctx, "gorl:fw:"+key)

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

	// Yeni window'da izin test
	time.Sleep(1100 * time.Millisecond)
	allowed, err = limiter.Allow(key)
	if !allowed || err != nil {
		t.Fatal("expected allowed after window reset")
	}
}
