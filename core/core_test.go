package core

import (
	"errors"
	"testing"
	"time"
)

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := Config{
		Strategy: FixedWindow,
		Limit:    10,
		Window:   time.Second,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestConfig_Validate_InvalidLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
	}{
		{"zero", 0},
		{"negative", -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Limit: tt.limit, Window: time.Second}
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected error for invalid limit")
			}
			if !errors.Is(err, ErrConfigInvalid) {
				t.Fatalf("expected ErrConfigInvalid, got %v", err)
			}
		})
	}
}

func TestConfig_Validate_InvalidWindow(t *testing.T) {
	tests := []struct {
		name   string
		window time.Duration
	}{
		{"zero", 0},
		{"negative", -time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Limit: 10, Window: tt.window}
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected error for invalid window")
			}
			if !errors.Is(err, ErrConfigInvalid) {
				t.Fatalf("expected ErrConfigInvalid, got %v", err)
			}
		})
	}
}

func TestNoopMetrics(t *testing.T) {
	m := &NoopMetrics{}
	// Should not panic
	m.IncAllow()
	m.IncDeny()
	m.ObserveLatency(time.Millisecond)
}

func TestStrategyTypeConstants(t *testing.T) {
	if FixedWindow != "fixed_window" {
		t.Errorf("expected fixed_window, got %s", FixedWindow)
	}
	if SlidingWindow != "sliding_window" {
		t.Errorf("expected sliding_window, got %s", SlidingWindow)
	}
	if TokenBucket != "token_bucket" {
		t.Errorf("expected token_bucket, got %s", TokenBucket)
	}
	if LeakyBucket != "leaky_bucket" {
		t.Errorf("expected leaky_bucket, got %s", LeakyBucket)
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrBackendUnavailable == nil {
		t.Error("ErrBackendUnavailable should not be nil")
	}
	if ErrConfigInvalid == nil {
		t.Error("ErrConfigInvalid should not be nil")
	}
	if ErrUnknownStrategy == nil {
		t.Error("ErrUnknownStrategy should not be nil")
	}
}
