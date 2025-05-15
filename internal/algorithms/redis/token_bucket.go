package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/AliRizaAynaci/gorl/core"
	"github.com/redis/go-redis/v9"
)

type TokenBucketLimiter struct {
	limit    int
	window   time.Duration
	client   *redis.Client
	prefix   string
	failOpen bool // fail-open policy flag
}

var tokenBucketLua = `
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local refill = tonumber(ARGV[2])
	local limit = tonumber(ARGV[3])
	local window = tonumber(ARGV[4])

	local bucket = redis.call("HMGET", key, "tokens", "last_refill")
	local tokens = tonumber(bucket[1]) or limit
	local last_refill = tonumber(bucket[2]) or now

	local elapsed = now - last_refill
	local rate = refill / window
	local new_tokens = math.min(limit, tokens + elapsed * rate)
	if new_tokens < 1 then
		redis.call("HMSET", key, "tokens", new_tokens, "last_refill", now)
		redis.call("EXPIRE", key, window)
		return 0
	else
		redis.call("HMSET", key, "tokens", new_tokens - 1, "last_refill", now)
		redis.call("EXPIRE", key, window)
		return 1
	end
`

func (r *TokenBucketLimiter) Allow(key string) (bool, error) {
	ctx := context.Background()
	now := time.Now().Unix()
	redisKey := fmt.Sprintf("%s:%s", r.prefix, key)
	result, err := r.client.Eval(ctx, tokenBucketLua, []string{redisKey},
		now, r.limit, r.limit, int(r.window.Seconds()),
	).Result()
	if err != nil {
		// If backend is unavailable, follow fail-open or fail-close policy and return a custom error.
		if r.failOpen {
			return true, core.ErrBackendUnavailable
		}
		return false, core.ErrBackendUnavailable
	}
	allowed, ok := result.(int64)
	return ok && allowed == 1, nil
}

func NewTokenBucketLimiter(cfg core.Config) core.Limiter {
	opt, _ := redis.ParseURL(cfg.RedisURL)
	client := redis.NewClient(opt)
	return &TokenBucketLimiter{
		limit:    cfg.Limit,
		window:   cfg.Window,
		client:   client,
		prefix:   "gorl:tb",
		failOpen: cfg.FailOpen,
	}
}
