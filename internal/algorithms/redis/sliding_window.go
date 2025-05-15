package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/redis/go-redis/v9"
)

type SlidingWindowLimiter struct {
	limit    int
	window   time.Duration
	client   *redis.Client
	prefix   string
	failOpen bool // fail-open policy flag
}

// Allow returns whether the request is allowed and an error if backend is unavailable.
func (r *SlidingWindowLimiter) Allow(key string) (bool, error) {
	ctx := context.Background()
	now := time.Now().UnixNano()
	redisKey := fmt.Sprintf("%s:%s", r.prefix, key)
	windowStart := time.Now().Add(-r.window).UnixNano()

	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart))
	pipe.ZAdd(ctx, redisKey, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, redisKey, r.window)
	res := pipe.ZCard(ctx, redisKey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		// If backend is unavailable, follow fail-open or fail-close policy and return a custom error.
		if r.failOpen {
			return true, core.ErrBackendUnavailable
		}
		return false, core.ErrBackendUnavailable
	}
	count, _ := res.Result()
	return count <= int64(r.limit), nil
}

// Pass the failOpen config option from core.Config
func NewSlidingWindowLimiter(cfg core.Config) core.Limiter {
	opt, _ := redis.ParseURL(cfg.RedisURL)
	client := redis.NewClient(opt)
	return &SlidingWindowLimiter{
		limit:    cfg.Limit,
		window:   cfg.Window,
		client:   client,
		prefix:   "gorl:sw",
		failOpen: cfg.FailOpen,
	}
}
