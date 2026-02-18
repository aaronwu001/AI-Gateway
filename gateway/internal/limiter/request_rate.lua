-- scripts/request_rate.lua

local tokens_key = KEYS[1]
local timestamp_key = KEYS[2]

local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

-- 1. 計算 TTL (過期時間)
local ttl = 60 -- 預設給個 60 秒
if rate > 0 then
    local fill_time = capacity / rate
    ttl = math.floor(fill_time * 2)
end
if ttl < 1 then ttl = 1 end -- 至少活 1 秒

-- 2. 讀取舊狀態
local last_tokens = tonumber(redis.call("get", tokens_key))
if last_tokens == nil then
    last_tokens = capacity
end

local last_refilled = tonumber(redis.call("get", timestamp_key))
if last_refilled == nil then
    last_refilled = 0
end

-- 3. 計算補充 (Lazy Refill)
local filled_tokens = last_tokens
if rate > 0 then -- ✨ FIX: 只有當速率 > 0 才計算補水
    local delta = math.max(0, now - last_refilled)
    filled_tokens = math.min(capacity, last_tokens + (delta * rate))
end

-- 4. 判斷與扣款
local allowed = filled_tokens >= requested
local new_tokens = filled_tokens
if allowed then
    new_tokens = filled_tokens - requested
end

-- 5. 寫入 Redis
redis.call("setex", tokens_key, ttl, new_tokens)
redis.call("setex", timestamp_key, ttl, now)

if allowed then
    return 1
else
    return 0
end