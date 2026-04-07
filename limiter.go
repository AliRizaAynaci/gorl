// Package gorl is the main package for the rate limiter library.
// It provides a simple entry point (New function) to create rate limiters
// with various algorithms and storage backends.
package gorl

import (
	"github.com/AliRizaAynaci/gorl/v2/core"
	"github.com/AliRizaAynaci/gorl/v2/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/v2/storage"
	"github.com/AliRizaAynaci/gorl/v2/storage/inmem"
	"github.com/AliRizaAynaci/gorl/v2/storage/redis"
)

var strategyRegistry = map[core.StrategyType]func(core.Config, storage.Storage) core.Limiter{
	core.FixedWindow:   algorithms.NewFixedWindowLimiter,
	core.TokenBucket:   algorithms.NewTokenBucketLimiter,
	core.SlidingWindow: algorithms.NewSlidingWindowLimiter,
	core.LeakyBucket:   algorithms.NewLeakyBucketLimiter,
}

// New creates a new rate limiter instance using the specified algorithm and storage backend.
// If cfg.RedisURL is provided, Redis is used as the storage backend. Otherwise, an in-memory backend is used.
// Supported strategies: FixedWindow, TokenBucket, SlidingWindow, LeakyBucket.
func New(cfg core.Config) (core.Limiter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.Metrics = normalizeMetrics(cfg.Metrics)

	store, err := newStore(cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	constructor, ok := strategyRegistry[cfg.Strategy]
	if !ok {
		return nil, core.ErrUnknownStrategy
	}
	return constructor(cfg, store), nil
}

// NewResourceLimiter creates a resource-scoped limiter that shares a single storage backend
// while allowing per-resource policies under the same strategy.
func NewResourceLimiter(cfg core.ResourceConfig) (core.ResourceLimiter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.Metrics = normalizeMetrics(cfg.Metrics)

	store, err := newStore(cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	constructor, ok := strategyRegistry[cfg.Strategy]
	if !ok {
		_ = store.Close()
		return nil, core.ErrUnknownStrategy
	}

	return newResourceRouter(cfg, store, constructor), nil
}

func normalizeMetrics(metrics core.MetricsCollector) core.MetricsCollector {
	if metrics == nil {
		return &core.NoopMetrics{}
	}
	return metrics
}

func newStore(redisURL string) (storage.Storage, error) {
	if redisURL != "" {
		return redis.NewRedisStore(redisURL)
	}
	return inmem.NewInMemoryStore(), nil
}
