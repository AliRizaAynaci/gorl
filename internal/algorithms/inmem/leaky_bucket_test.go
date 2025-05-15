package algorithms

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func TestLeakyBucketLimiter_Allow(t *testing.T) {
	cfg := core.Config{
		Limit:  2,
		Window: 200 * time.Millisecond,
	}
	limiter := NewLeakyBucketLimiter(cfg)
	key := "lb-user"

	// İlk iki istek geçmeli
	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(key)
		if !allowed || err != nil {
			t.Fatalf("expected allowed on attempt %d", i+1)
		}
	}
	// 3. istek hemen gelirse blocklanmalı
	allowed, err := limiter.Allow(key)
	if err != nil {
		t.Fatalf("unexpected error on 3rd request: %v", err)
	}
	if allowed {
		t.Fatal("expected denied on 3rd request")
	}
	// Window geçince tekrar izin verilmeli (leak sonrası)
	time.Sleep(210 * time.Millisecond)
	allowed, err = limiter.Allow(key)
	if !allowed || err != nil {
		t.Fatal("expected allowed after leak")
	}
}
