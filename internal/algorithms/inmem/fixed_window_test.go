package algorithms

import (
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
)

func TestFixedWindowLimiter_Allow(t *testing.T) {
	cfg := core.Config{
		Limit:  2,
		Window: 100 * time.Millisecond,
	}
	limiter := NewFixedWindowLimiter(cfg)
	key := "fw-user"

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
		t.Fatal("expected denied on 3rd request")
	}
	// Yeni window'da tekrar izin verilmeli
	time.Sleep(120 * time.Millisecond)
	allowed, err = limiter.Allow(key)
	if !allowed || err != nil {
		t.Fatal("expected allowed after window reset")
	}
}
