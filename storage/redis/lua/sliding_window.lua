local limit = tonumber(ARGV[1])
local now_us = tonumber(ARGV[2])
local window_us = tonumber(ARGV[3])
local ttl_ms = tonumber(ARGV[4])

local window_start = tonumber(redis.call("GET", KEYS[1]) or "0")
local curr = tonumber(redis.call("GET", KEYS[2]) or "0")
local prev = tonumber(redis.call("GET", KEYS[3]) or "0")

if window_start == 0 then
  window_start = now_us
  curr = 0
  prev = 0
else
  local elapsed = now_us - window_start
  if elapsed >= window_us then
    local intervals = math.floor(elapsed / window_us)
    if intervals == 1 then
      prev = curr
    else
      prev = 0
    end
    curr = 0
    window_start = window_start + (intervals * window_us)
  end
end

local since = now_us - window_start
if since < 0 then
  since = 0
end

local ratio = since / window_us
local sliding = (prev * (1 - ratio)) + curr

local allowed = 0
local sliding_after = sliding
if sliding < limit then
  allowed = 1
  curr = curr + 1
  sliding_after = sliding_after + 1
end

redis.call("SET", KEYS[1], window_start, "PX", ttl_ms)
redis.call("SET", KEYS[2], curr, "PX", ttl_ms)
redis.call("SET", KEYS[3], prev, "PX", ttl_ms)

local remaining = math.floor(limit - sliding_after)
if remaining < 0 then
  remaining = 0
end

local window_until_boundary = window_us - since
if window_until_boundary < 0 then
  window_until_boundary = 0
end

local reset_us = 0
if curr > 0 then
  reset_us = (2 * window_us) - since
  if reset_us < 0 then
    reset_us = 0
  end
elseif prev > 0 then
  reset_us = window_until_boundary
end

local retry_after_us = 0
if allowed == 0 then
  if curr >= limit then
    retry_after_us = window_until_boundary
  elseif prev > 0 then
    local required_ratio = 1 - ((limit - curr) / prev)
    local delay_ratio = required_ratio - ratio
    if delay_ratio < 0 then
      delay_ratio = 0
    end
    retry_after_us = math.ceil(delay_ratio * window_us)
    if retry_after_us <= 0 then
      retry_after_us = 1
    end
    if retry_after_us > window_until_boundary then
      retry_after_us = window_until_boundary
    end
  else
    retry_after_us = window_until_boundary
  end
end

return {allowed, remaining, math.floor(reset_us), math.floor(retry_after_us)}
