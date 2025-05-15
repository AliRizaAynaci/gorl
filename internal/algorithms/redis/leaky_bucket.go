package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/redis/go-redis/v9"
)

type LeakyBucketLimiter struct {
	limit    int
	window   time.Duration
	client   *redis.Client
	prefix   string
	failOpen bool
}

var leakyBucketLua = `
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local limit = tonumber(ARGV[2])
	local window = tonumber(ARGV[3])

	local bucket = redis.call("HMGET", key, "water", "last_leak")
	local water = tonumber(bucket[1]) or 0
	local last_leak = tonumber(bucket[2]) or now

	local elapsed = now - last_leak
	local leak_rate = limit / window
	local leaked = elapsed * leak_rate
	water = math.max(0, water - leaked)
	last_leak = now

	if water < limit then
		redis.call("HMSET", key, "water", water + 1, "last_leak", now)
		redis.call("EXPIRE", key, window)
		return 1
	else
		redis.call("HMSET", key, "water", water, "last_leak", now)
		redis.call("EXPIRE", key, window)
		return 0
	end
`

func (r *LeakyBucketLimiter) Allow(key string) (bool, error) {
	ctx := context.Background()
	now := time.Now().Unix()
	redisKey := fmt.Sprintf("%s:%s", r.prefix, key)
	result, err := r.client.Eval(ctx, leakyBucketLua, []string{redisKey},
		now, r.limit, int(r.window.Seconds()),
	).Result()
	if err != nil {
		// If the backend (Redis) is unavailable, either allow or block the request
		// depending on the failOpen policy. Always return a custom error for observability.
		if r.failOpen {
			return true, core.ErrBackendUnavailable
		}
		return false, core.ErrBackendUnavailable
	}
	allowed, ok := result.(int64)
	return ok && allowed == 1, nil
}

func NewLeakyBucketLimiter(cfg core.Config) core.Limiter {
	opt, _ := redis.ParseURL(cfg.RedisURL)
	client := redis.NewClient(opt)
	return &LeakyBucketLimiter{
		limit:    cfg.Limit,
		window:   cfg.Window,
		client:   client,
		prefix:   "gorl:lb",
		failOpen: cfg.FailOpen, // Pass config failOpen
	}
}
