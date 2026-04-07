package gorl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
)

func TestNewResourceLimiter_AllStrategies(t *testing.T) {
	strategies := []core.StrategyType{
		core.FixedWindow,
		core.SlidingWindow,
		core.TokenBucket,
		core.LeakyBucket,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			limiter, err := NewResourceLimiter(core.ResourceConfig{
				Strategy:      strategy,
				DefaultPolicy: core.ResourcePolicy{Limit: 2, Window: time.Second},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer limiter.Close()

			res, err := limiter.AllowResource(context.Background(), "login", "user-123")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !res.Allowed {
				t.Fatal("expected request to be allowed")
			}
		})
	}
}

func TestNewResourceLimiter_UsesDefaultPolicyForUnknownResource(t *testing.T) {
	limiter, err := NewResourceLimiter(core.ResourceConfig{
		Strategy:      core.FixedWindow,
		DefaultPolicy: core.ResourcePolicy{Limit: 1, Window: time.Minute},
		Resources: map[string]core.ResourcePolicy{
			"search": {Limit: 2, Window: time.Minute},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()

	firstUnknown, err := limiter.AllowResource(ctx, "unknown", "user-123")
	if err != nil {
		t.Fatalf("unexpected error for unknown resource: %v", err)
	}
	if !firstUnknown.Allowed {
		t.Fatal("expected first unknown resource request to be allowed")
	}

	secondUnknown, err := limiter.AllowResource(ctx, "unknown", "user-123")
	if err != nil {
		t.Fatalf("unexpected error for unknown resource: %v", err)
	}
	if secondUnknown.Allowed {
		t.Fatal("expected second unknown resource request to be denied by default policy")
	}

	for i := 0; i < 2; i++ {
		res, err := limiter.AllowResource(ctx, "search", "user-123")
		if err != nil {
			t.Fatalf("unexpected error for named resource: %v", err)
		}
		if !res.Allowed {
			t.Fatalf("expected named resource request %d to be allowed", i+1)
		}
	}

	thirdSearch, err := limiter.AllowResource(ctx, "search", "user-123")
	if err != nil {
		t.Fatalf("unexpected error for named resource: %v", err)
	}
	if thirdSearch.Allowed {
		t.Fatal("expected third named resource request to be denied")
	}
}

func TestNewResourceLimiter_InvalidDefaultPolicy(t *testing.T) {
	_, err := NewResourceLimiter(core.ResourceConfig{
		Strategy:      core.FixedWindow,
		DefaultPolicy: core.ResourcePolicy{Limit: 0, Window: time.Second},
	})
	if !errors.Is(err, core.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}

func TestNewResourceLimiter_InvalidNamedPolicy(t *testing.T) {
	_, err := NewResourceLimiter(core.ResourceConfig{
		Strategy:      core.FixedWindow,
		DefaultPolicy: core.ResourcePolicy{Limit: 1, Window: time.Second},
		Resources: map[string]core.ResourcePolicy{
			"login": {Limit: 0, Window: time.Second},
		},
	})
	if !errors.Is(err, core.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}

func TestNewResourceLimiter_UnknownStrategy(t *testing.T) {
	_, err := NewResourceLimiter(core.ResourceConfig{
		Strategy:      "unknown",
		DefaultPolicy: core.ResourcePolicy{Limit: 1, Window: time.Second},
	})
	if !errors.Is(err, core.ErrUnknownStrategy) {
		t.Fatalf("expected ErrUnknownStrategy, got %v", err)
	}
}
