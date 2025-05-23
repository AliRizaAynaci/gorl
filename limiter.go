// Package gorl is the main package for the rate limiter library.
// It provides a simple entry point (New function) to create rate limiters
// with various algorithms and storage backends.
package gorl

import (
	"github.com/AliRizaAynaci/gorl/core"
	"github.com/AliRizaAynaci/gorl/internal/algorithms"
	"github.com/AliRizaAynaci/gorl/storage"
	"github.com/AliRizaAynaci/gorl/storage/inmem"
	"github.com/AliRizaAynaci/gorl/storage/redis"
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
	var store storage.Storage

	if cfg.RedisURL != "" {
		store = redis.NewRedisStore(cfg.RedisURL)
	} else {
		store = inmem.NewInMemoryStore()
	}

	constructor, ok := strategyRegistry[cfg.Strategy]
	if !ok {
		return nil, core.ErrUnknownStrategy
	}
	return constructor(cfg, store), nil
}
