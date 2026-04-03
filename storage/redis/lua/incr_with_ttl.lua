local ttl_ms = tonumber(ARGV[1])

local value = redis.call("INCR", KEYS[1])
redis.call("PEXPIRE", KEYS[1], ttl_ms)

return value
