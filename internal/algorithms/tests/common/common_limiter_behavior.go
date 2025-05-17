package common

import (
	"testing"

	"github.com/AliRizaAynaci/gorl/core"
)

func CommonLimiterBehavior(t *testing.T, limiter core.Limiter, key string, limit int) {
	t.Helper()
	for i := 0; i < limit; i++ {
		allowed, err := limiter.Allow(key)
		if err != nil || !allowed {
			t.Fatalf("expected allowed, got %v, err %v (req %d)", allowed, err, i+1)
		}
	}
	allowed, err := limiter.Allow(key)
	if allowed || err != nil {
		t.Fatalf("expected denied after limit, got %v, err %v", allowed, err)
	}
}
