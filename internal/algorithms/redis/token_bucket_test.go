package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func redisURL() string {
	url := os.Getenv("GORL_REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/0"
	}
	return url
}

func TestTokenBucketLimiter_Allow(t *testing.T) {
	cfg := core.Config{
		Limit:    2,
		Window:   time.Second,
		RedisURL: redisURL(),
	}
	limiter := NewTokenBucketLimiter(cfg)
	key := "test-redis-tb"

	client := limiter.(*TokenBucketLimiter).client
	ctx := context.Background()
	client.Del(ctx, "gorl:tb:"+key)

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
}
