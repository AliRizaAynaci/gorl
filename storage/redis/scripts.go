package redis

import (
	"context"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	scriptIncrWithTTL   = "incr_with_ttl"
	scriptSlidingWindow = "sliding_window"
	scriptTokenBucket   = "token_bucket"
	scriptLeakyBucket   = "leaky_bucket"
)

// Embed the whole script directory so editor/go list glob resolution does not
// become a separate failure mode.
//go:embed lua
var luaScriptsFS embed.FS

var scriptRegistry = map[string]*goredis.Script{
	scriptIncrWithTTL:   goredis.NewScript(mustReadLuaScript("lua/incr_with_ttl.lua")),
	scriptSlidingWindow: goredis.NewScript(mustReadLuaScript("lua/sliding_window.lua")),
	scriptTokenBucket:   goredis.NewScript(mustReadLuaScript("lua/token_bucket.lua")),
	scriptLeakyBucket:   goredis.NewScript(mustReadLuaScript("lua/leaky_bucket.lua")),
}

func mustReadLuaScript(path string) string {
	body, err := luaScriptsFS.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to load redis lua script %q: %v", path, err))
	}
	return string(body)
}

func ttlMilliseconds(ttl time.Duration) int64 {
	ms := ttl.Milliseconds()
	if ms <= 0 {
		return 1
	}
	return ms
}

func (s *RedisStore) runScript(ctx context.Context, name string, keys []string, args ...int64) (interface{}, error) {
	script, ok := scriptRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown redis script %q", name)
	}

	argv := make([]interface{}, len(args))
	for i, arg := range args {
		argv[i] = arg
	}

	res, err := script.Run(ctx, s.client, keys, argv...).Result()
	if err != nil {
		return nil, err
	}
	return res, nil
}

// EvalScript runs a named Lua script and converts its result array into int64 values.
func (s *RedisStore) EvalScript(ctx context.Context, name string, keys []string, args ...int64) ([]int64, error) {
	raw, err := s.runScript(ctx, name, keys, args...)
	if err != nil {
		return nil, err
	}

	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected redis script result type %T", raw)
	}

	out := make([]int64, len(items))
	for i, item := range items {
		val, err := asInt64(item)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis script result at index %d: %w", i, err)
		}
		out[i] = val
	}
	return out, nil
}

func asInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case nil:
		return 0, nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case string:
		return parseInt64String(v)
	case []byte:
		return parseInt64String(string(v))
	default:
		return 0, fmt.Errorf("unsupported redis value type %T", value)
	}
}

func parseInt64String(s string) (int64, error) {
	if strings.ContainsRune(s, '.') {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		return int64(f), nil
	}
	return strconv.ParseInt(s, 10, 64)
}
