// Package core defines the core interfaces, types, and constants used by the rate limiting library.
package core

import (
	"errors"
	"time"
)

// Common error values for rate limiting failures.
var (
	// ErrBackendUnavailable indicates that the storage backend (e.g., Redis) is not reachable.
	ErrBackendUnavailable = errors.New("backend unavailable") // e.g., Redis is down
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
// It receives a context object (for example, *fiber.Ctx, *gin.Context, etc.) and returns the rate limit key as string.
type KeyExtractor func(ctx interface{}) string

// Config holds the configuration for creating a rate limiter.
type Config struct {
	Strategy  StrategyType  // Rate limiting algorithm to use
	KeyBy     KeyFuncType   // Keying strategy for limiting
	Limit     int           // Maximum allowed requests/tokens per window
	Window    time.Duration // Time window duration
	RedisURL  string        // Redis connection string for distributed mode
	HeaderKey string        // Optional: header key for API keys/tokens
	FailOpen  bool          // If true, allow requests when backend is unavailable (fail-open); if false, block (fail-close)
	// CustomKeyExtractor is an optional function for custom rate limiting key generation.
	// Used only when KeyBy == KeyByCustom.
	CustomKeyExtractor KeyExtractor
	// Optional: metrics collector (nil → NoopMetrics)
	Metrics MetricsCollector
}

// Limiter defines the interface that all rate limiting strategies must implement.
type Limiter interface {
	// Allow returns true if the request with given key is allowed, and an error if there was an internal error.
	Allow(key string) (bool, error)
}
