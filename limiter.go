package gorl

import (
	"github.com/AliRizaAynaci/gorl/core"
	inmem "github.com/AliRizaAynaci/gorl/internal/algorithms/inmem"
	redislimiter "github.com/AliRizaAynaci/gorl/internal/algorithms/redis"
)

// In-memory strategy registry
var inMemRegistry = map[core.StrategyType]func(core.Config) core.Limiter{
	core.TokenBucket:   inmem.NewTokenBucketLimiter,
	core.FixedWindow:   inmem.NewFixedWindowLimiter,
	core.SlidingWindow: inmem.NewSlidingWindowLimiter,
	core.LeakyBucket:   inmem.NewLeakyBucketLimiter,
}

// Redis strategy registry
var redisRegistry = map[core.StrategyType]func(core.Config) core.Limiter{
	core.TokenBucket:   redislimiter.NewTokenBucketLimiter,
	core.FixedWindow:   redislimiter.NewFixedWindowLimiter,
	core.SlidingWindow: redislimiter.NewSlidingWindowLimiter,
	core.LeakyBucket:   redislimiter.NewLeakyBucketLimiter,
}

func New(cfg core.Config) (core.Limiter, error) {
	var registry map[core.StrategyType]func(core.Config) core.Limiter
	if cfg.RedisURL != "" {
		registry = redisRegistry
	} else {
		registry = inMemRegistry
	}

	if constructor, ok := registry[cfg.Strategy]; ok {
		return constructor(cfg), nil
	}
	return nil, core.ErrUnknownStrategy
}
