// Package storage provides a generic key-value storage interface
// specifically designed for rate limiting algorithms.
package storage

import (
	"context"
	"time"
)

// Storage defines a minimal key-value interface for rate limiting.
// Implementations need only support Get, Set, Incr with TTL, and Close.
// All complex algorithm logic lives in the algorithms package.
type Storage interface {
	// Incr atomically increments the numeric value at key by 1.
	// If the key is missing or expired, it initializes it to 1 and applies TTL.
	Incr(ctx context.Context, key string, ttl time.Duration) (float64, error)

	// Get retrieves the numeric value stored at key.
	// Returns 0 if the key does not exist or has expired.
	Get(ctx context.Context, key string) (float64, error)

	// Set stores the numeric value at key with the given TTL.
	Set(ctx context.Context, key string, val float64, ttl time.Duration) error

	// Close releases any resources held by the storage backend.
	Close() error
}
