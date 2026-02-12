package gorl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/core"
)

func TestNew_FixedWindow(t *testing.T) {
	limiter, err := New(core.Config{
		Strategy: core.FixedWindow,
		Limit:    5,
		Window:   time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	res, err := limiter.Allow(ctx, "test")
	if err != nil || !res.Allowed {
		t.Fatalf("expected allowed, got %v, err %v", res.Allowed, err)
	}
}

func TestNew_SlidingWindow(t *testing.T) {
	limiter, err := New(core.Config{
		Strategy: core.SlidingWindow,
		Limit:    5,
		Window:   time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	res, err := limiter.Allow(ctx, "test")
	if err != nil || !res.Allowed {
		t.Fatalf("expected allowed, got %v, err %v", res.Allowed, err)
	}
}

func TestNew_TokenBucket(t *testing.T) {
	limiter, err := New(core.Config{
		Strategy: core.TokenBucket,
		Limit:    5,
		Window:   time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	res, err := limiter.Allow(ctx, "test")
	if err != nil || !res.Allowed {
		t.Fatalf("expected allowed, got %v, err %v", res.Allowed, err)
	}
}

func TestNew_LeakyBucket(t *testing.T) {
	limiter, err := New(core.Config{
		Strategy: core.LeakyBucket,
		Limit:    5,
		Window:   time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	res, err := limiter.Allow(ctx, "test")
	if err != nil || !res.Allowed {
		t.Fatalf("expected allowed, got %v, err %v", res.Allowed, err)
	}
}

func TestNew_UnknownStrategy(t *testing.T) {
	_, err := New(core.Config{
		Strategy: "unknown",
		Limit:    5,
		Window:   time.Second,
	})
	if !errors.Is(err, core.ErrUnknownStrategy) {
		t.Fatalf("expected ErrUnknownStrategy, got %v", err)
	}
}

func TestNew_InvalidLimit(t *testing.T) {
	_, err := New(core.Config{
		Strategy: core.FixedWindow,
		Limit:    0,
		Window:   time.Second,
	})
	if !errors.Is(err, core.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}

func TestNew_InvalidWindow(t *testing.T) {
	_, err := New(core.Config{
		Strategy: core.FixedWindow,
		Limit:    5,
		Window:   0,
	})
	if !errors.Is(err, core.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}

func TestNew_InvalidRedisURL(t *testing.T) {
	_, err := New(core.Config{
		Strategy: core.FixedWindow,
		Limit:    5,
		Window:   time.Second,
		RedisURL: "not-a-valid-url",
	})
	if err == nil {
		t.Fatal("expected error for invalid redis URL")
	}
}

func TestNew_NilMetricsDefaultsToNoop(t *testing.T) {
	limiter, err := New(core.Config{
		Strategy: core.FixedWindow,
		Limit:    5,
		Window:   time.Second,
		Metrics:  nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer limiter.Close()

	// Should work without panic (NoopMetrics used)
	ctx := context.Background()
	limiter.Allow(ctx, "test")
}

func TestNew_AllStrategiesRespectLimit(t *testing.T) {
	strategies := []core.StrategyType{
		core.FixedWindow,
		core.SlidingWindow,
		core.TokenBucket,
		core.LeakyBucket,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			limiter, err := New(core.Config{
				Strategy: strategy,
				Limit:    2,
				Window:   5 * time.Second,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer limiter.Close()

			ctx := context.Background()
			for i := 0; i < 2; i++ {
				res, err := limiter.Allow(ctx, "key")
				if err != nil || !res.Allowed {
					t.Fatalf("req %d: expected allowed, got %v, err %v", i+1, res.Allowed, err)
				}
			}

			res, err := limiter.Allow(ctx, "key")
			if res.Allowed || err != nil {
				t.Fatalf("expected denied after limit, got %v, err %v", res.Allowed, err)
			}
		})
	}
}
