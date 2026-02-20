-- scripts/request_rate.lua

local tokens_key = KEYS[1]
local timestamp_key = KEYS[2]

local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

-- 1. Calculate TTL (expiration time)
local ttl = 60 -- Default to 60 seconds
if rate > 0 then
    local fill_time = capacity / rate
    ttl = math.floor(fill_time * 2)
end
if ttl < 1 then ttl = 1 end -- Keep alive for at least 1 second

-- 2. Read previous state
local last_tokens = tonumber(redis.call("get", tokens_key))
if last_tokens == nil then
    last_tokens = capacity
end

local last_refilled = tonumber(redis.call("get", timestamp_key))
if last_refilled == nil then
    last_refilled = 0
end

-- 3. Compute refill (lazy refill)
local filled_tokens = last_tokens
if rate > 0 then
    local delta = math.max(0, now - last_refilled)
    filled_tokens = math.min(capacity, last_tokens + (delta * rate))
end

-- 4. Check allowance and deduct tokens
local allowed = filled_tokens >= requested
local new_tokens = filled_tokens
if allowed then
    new_tokens = filled_tokens - requested
end

-- 5. Write back to Redis
redis.call("setex", tokens_key, ttl, new_tokens)
redis.call("setex", timestamp_key, ttl, now)

if allowed then
    return 1
else
    return 0
end