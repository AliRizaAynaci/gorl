local limit = tonumber(ARGV[1])
local now_us = tonumber(ARGV[2])
local ttl_ms = tonumber(ARGV[3])
local time_per_token_us = tonumber(ARGV[4])

local tokens = tonumber(redis.call("GET", KEYS[1]) or "0")
local last_refill = tonumber(redis.call("GET", KEYS[2]) or "0")

if last_refill == 0 then
  tokens = limit
  last_refill = now_us
else
  local elapsed = now_us - last_refill
  local new_tokens = math.floor(elapsed / time_per_token_us)
  if new_tokens > 0 then
    tokens = tokens + new_tokens
    if tokens > limit then
      tokens = limit
    end
    last_refill = last_refill + (new_tokens * time_per_token_us)
  end
end

local allowed = 0
if tokens > 0 then
  allowed = 1
  tokens = tokens - 1
end

local elapsed_since_refill = now_us - last_refill
local next_token_delay = time_per_token_us - elapsed_since_refill
if next_token_delay < 0 then
  next_token_delay = 0
end

local missing_tokens = limit - tokens
local reset_us = 0
if missing_tokens > 0 then
  reset_us = (missing_tokens * time_per_token_us) - elapsed_since_refill
  if reset_us < 0 then
    reset_us = 0
  end
end

redis.call("SET", KEYS[1], tokens, "PX", ttl_ms)
redis.call("SET", KEYS[2], last_refill, "PX", ttl_ms)

local retry_after_us = 0
if allowed == 0 then
  retry_after_us = next_token_delay
end

return {allowed, tokens, reset_us, retry_after_us}
