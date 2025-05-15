package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/redis/go-redis/v9"
)

type FixedWindowLimiter struct {
	limit    int
	window   time.Duration
	client   *redis.Client
	prefix   string
	failOpen bool // fail-open policy flag
}

// Allow returns whether the request is allowed and an error if backend is unavailable.
func (r *FixedWindowLimiter) Allow(key string) (bool, error) {
	ctx := context.Background()
	redisKey := fmt.Sprintf("%s:%s", r.prefix, key)
	count, err := r.client.Incr(ctx, redisKey).Result()
	if err != nil {
		// If backend is unavailable, follow fail-open or fail-close policy and return a custom error.
		if r.failOpen {
			return true, core.ErrBackendUnavailable
		}
		return false, core.ErrBackendUnavailable
	}
	if count == 1 {
		r.client.Expire(ctx, redisKey, r.window)
	}
	return count <= int64(r.limit), nil
}

func NewFixedWindowLimiter(cfg core.Config) core.Limiter {
	opt, _ := redis.ParseURL(cfg.RedisURL)
	client := redis.NewClient(opt)
	return &FixedWindowLimiter{
		limit:    cfg.Limit,
		window:   cfg.Window,
		client:   client,
		prefix:   "gorl:fw",
		failOpen: cfg.FailOpen,
	}
}
