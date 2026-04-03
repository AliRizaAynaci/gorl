// Package redis provides a Redis-backed storage implementation for the rate limiter.
package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/AliRizaAynaci/gorl/v2/storage"
	goredis "github.com/redis/go-redis/v9"
)

// RedisStore implements the storage.Storage interface using a Redis backend.
// It also exposes Lua-scripted helpers for atomic multi-key state transitions.
type RedisStore struct {
	client *goredis.Client
}

// NewRedisStore parses the URL and returns a Redis-backed Storage.
// Returns an error if the URL is invalid or if the connection fails.
func NewRedisStore(redisURL string) (storage.Storage, error) {
	opt, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}
	client := goredis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisStore{
		client: client,
	}, nil
}

// Incr atomically increments the numeric value at key by 1.
// If the key is missing or expired, initializes it to 1 and sets TTL.
func (s *RedisStore) Incr(ctx context.Context, key string, ttl time.Duration) (float64, error) {
	raw, err := s.runScript(ctx, scriptIncrWithTTL, []string{key}, ttlMilliseconds(ttl))
	if err != nil {
		return 0, err
	}

	val, err := asInt64(raw)
	if err != nil {
		return 0, fmt.Errorf("failed to parse increment result: %w", err)
	}
	return float64(val), nil
}

// Get retrieves the numeric value at key, or 0 if not found/expired.
func (s *RedisStore) Get(ctx context.Context, key string) (float64, error) {
	str, err := s.client.Get(ctx, key).Result()
	if err == goredis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse stored value %q: %w", str, err)
	}
	return f, nil
}

// Set stores the numeric value at key with the given TTL.
func (s *RedisStore) Set(ctx context.Context, key string, val float64, ttl time.Duration) error {
	return s.client.Set(ctx, key, val, ttl).Err()
}

// Close closes the underlying Redis client connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Client returns the underlying go-redis client for advanced usage.
func (s *RedisStore) Client() *goredis.Client {
	return s.client
}
