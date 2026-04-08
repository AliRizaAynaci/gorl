package core

import (
	"context"
	"fmt"
	"time"
)

// ResourcePolicy defines the rate-limit policy for a single resource.
type ResourcePolicy struct {
	Limit  int           // Maximum allowed requests/tokens per window
	Window time.Duration // Time window duration
}

// Validate checks the resource policy for common errors.
func (p ResourcePolicy) Validate() error {
	return validateLimitWindow(p.Limit, p.Window)
}

// ResourceConfig holds the configuration for creating a resource-scoped limiter.
type ResourceConfig struct {
	Strategy      StrategyType              // Rate limiting algorithm to use across all resources
	DefaultPolicy ResourcePolicy            // Fallback policy for resources not present in Resources
	Resources     map[string]ResourcePolicy // Per-resource policy overrides
	RedisURL      string                    // Redis connection string for distributed mode
	FailOpen      bool                      // If true, allow requests when backend is unavailable
	// Optional: metrics collector (nil -> NoopMetrics)
	Metrics MetricsCollector
}

// Validate checks the resource-scoped configuration for common errors.
func (c ResourceConfig) Validate() error {
	if err := c.DefaultPolicy.Validate(); err != nil {
		return fmt.Errorf("default policy: %w", err)
	}
	for resource, policy := range c.Resources {
		if resource == "" {
			return fmt.Errorf("%w: resource name must not be empty", ErrConfigInvalid)
		}
		if err := policy.Validate(); err != nil {
			return fmt.Errorf("resource %q: %w", resource, err)
		}
	}
	return nil
}

// ResourceLimiter defines the interface for resource-scoped rate limiting.
type ResourceLimiter interface {
	// AllowResource returns a Result indicating if the request is permitted for the given resource and key.
	AllowResource(ctx context.Context, resource, key string) (Result, error)
	// Close releases any resources held by the limiter.
	Close() error
}

func validateLimitWindow(limit int, window time.Duration) error {
	if limit <= 0 {
		return fmt.Errorf("%w: limit must be greater than 0", ErrConfigInvalid)
	}
	if window <= 0 {
		return fmt.Errorf("%w: window must be greater than 0", ErrConfigInvalid)
	}
	return nil
}
