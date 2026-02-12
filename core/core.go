// Package core defines the core interfaces, types, and constants used by the rate limiting library.
package core

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Common error values for rate limiting failures.
var (
	// ErrBackendUnavailable indicates that the storage backend (e.g., Redis) is not reachable.
	ErrBackendUnavailable = errors.New("backend unavailable")
	// ErrConfigInvalid indicates that the provided configuration for the rate limiter is not valid.
	ErrConfigInvalid = errors.New("invalid configuration")
	// ErrUnknownStrategy indicates that the requested rate limiting strategy is not supported.
	ErrUnknownStrategy = errors.New("unknown rate limiting strategy")
)

// StrategyType represents the available rate limiting algorithms.
type StrategyType string

// KeyFuncType represents how the rate limiter generates a key per request.
type KeyFuncType string

const (
	// FixedWindow is the basic fixed window rate limiting algorithm.
	FixedWindow StrategyType = "fixed_window"
	// SlidingWindow is the sliding window algorithm.
	SlidingWindow StrategyType = "sliding_window"
	// TokenBucket is the token bucket algorithm, allowing bursts.
	TokenBucket StrategyType = "token_bucket"
	// LeakyBucket is the leaky bucket algorithm.
	LeakyBucket StrategyType = "leaky_bucket"

	// KeyByIP limits per remote IP address.
	KeyByIP KeyFuncType = "ip"
	// KeyByAPIKey limits per API key (from request header).
	KeyByAPIKey KeyFuncType = "api_key"
	// KeyByToken limits per bearer token.
	KeyByToken KeyFuncType = "token"
	// KeyByCustom allows a user-defined key function.
	KeyByCustom KeyFuncType = "custom"
)

// KeyExtractor defines the function signature for custom key extraction.
type KeyExtractor func(ctx interface{}) string

// Config holds the configuration for creating a rate limiter.
type Config struct {
	Strategy  StrategyType  // Rate limiting algorithm to use
	KeyBy     KeyFuncType   // Keying strategy for limiting
	Limit     int           // Maximum allowed requests/tokens per window
	Window    time.Duration // Time window duration
	RedisURL  string        // Redis connection string for distributed mode
	HeaderKey string        // Optional: header key for API keys/tokens
	FailOpen  bool          // If true, allow requests when backend is unavailable
	// CustomKeyExtractor is an optional function for custom rate limiting key generation.
	CustomKeyExtractor KeyExtractor
	// Optional: metrics collector (nil → NoopMetrics)
	Metrics MetricsCollector
}

// Validate checks the configuration for common errors.
func (c Config) Validate() error {
	if c.Limit <= 0 {
		return fmt.Errorf("%w: limit must be greater than 0", ErrConfigInvalid)
	}
	if c.Window <= 0 {
		return fmt.Errorf("%w: window must be greater than 0", ErrConfigInvalid)
	}
	return nil
}

// Result represents the outcome of a rate limiting check.
type Result struct {
	Allowed    bool          // True if the request is permitted
	Limit      int           // Total capacity configured
	Remaining  int           // Current remaining capacity
	Reset      time.Duration // Time until the next full reset or refill
	RetryAfter time.Duration // Time to wait before the next allowed request (if denied)
}

// Limiter defines the interface that all rate limiting strategies must implement.
type Limiter interface {
	// Allow returns a Result indicating if the request is permitted and metadata about the state.
	Allow(ctx context.Context, key string) (Result, error)
	// Close releases any resources held by the limiter.
	Close() error
}
