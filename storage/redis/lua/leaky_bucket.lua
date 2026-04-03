local limit = tonumber(ARGV[1])
local now_us = tonumber(ARGV[2])
local window_us = tonumber(ARGV[3])
local ttl_ms = tonumber(ARGV[4])

local us_per_token = math.floor(window_us / limit)
if us_per_token < 1 then
  us_per_token = 1
end

local water = tonumber(redis.call("GET", KEYS[1]) or "0")
local last_leak = tonumber(redis.call("GET", KEYS[2]) or "0")

if last_leak == 0 then
  water = 0
  last_leak = now_us
else
  local elapsed = now_us - last_leak
  local leaked = math.floor(elapsed / us_per_token)
  if leaked > 0 then
    water = water - leaked
    if water < 0 then
      water = 0
    end
    last_leak = last_leak + (leaked * us_per_token)
  end
end

local allowed = 0
if water < limit then
  allowed = 1
  water = water + 1
end

local elapsed_since_leak = now_us - last_leak
local next_leak = us_per_token - elapsed_since_leak
if next_leak < 0 then
  next_leak = 0
end

local reset_us = 0
if water > 0 then
  reset_us = (water * us_per_token) - elapsed_since_leak
  if reset_us < 0 then
    reset_us = 0
  end
end

local remaining = limit - water
if remaining < 0 then
  remaining = 0
end

redis.call("SET", KEYS[1], water, "PX", ttl_ms)
redis.call("SET", KEYS[2], last_leak, "PX", ttl_ms)

local retry_after_us = 0
if allowed == 0 then
  retry_after_us = next_leak
end

return {allowed, remaining, reset_us, retry_after_us}
