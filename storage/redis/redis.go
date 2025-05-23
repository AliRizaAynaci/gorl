// Package redis provides a Redis-backed storage implementation for the rate limiter.
package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/AliRizaAynaci/gorl/storage"
	goredis "github.com/redis/go-redis/v9"
)

// RedisStore implements the storage.Storage interface using a Redis backend.
// It provides atomic operations necessary for distributed rate limiting.
type RedisStore struct {
	client *goredis.Client
	ctx    context.Context
}

// NewRedisStore parses the URL and returns a Redis-backed minimal Storage.
func NewRedisStore(redisURL string) storage.Storage {
	opt, err := goredis.ParseURL(redisURL)
	if err != nil {
		panic(err)
	}
	client := goredis.NewClient(opt)
	return &RedisStore{
		client: client,
		ctx:    context.Background(),
	}
}

// Incr atomically increments the numeric value at key by 1.
// If the key is missing or expired, initializes it to 1 and sets TTL.
func (s *RedisStore) Incr(key string, ttl time.Duration) (float64, error) {
	val, err := s.client.Incr(s.ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// (Re)set TTL on every increment
	_, _ = s.client.Expire(s.ctx, key, ttl).Result()
	return float64(val), nil
}

// Get retrieves the numeric value at key, or 0 if not found/expired.
func (s *RedisStore) Get(key string) (float64, error) {
	str, err := s.client.Get(s.ctx, key).Result()
	if err == goredis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	f, _ := strconv.ParseFloat(str, 64)
	return f, nil
}

// Set stores the numeric value at key with the given TTL.
func (s *RedisStore) Set(key string, val float64, ttl time.Duration) error {
	return s.client.Set(s.ctx, key, val, ttl).Err()
}

func (s *RedisStore) Client() *goredis.Client {
	return s.client
}
