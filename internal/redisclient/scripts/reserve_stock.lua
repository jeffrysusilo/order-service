-- Reserve stock atomically
-- KEYS[1] = inventory key (e.g., "inventory:123")
-- ARGV[1] = quantity to reserve

local available = tonumber(redis.call("HGET", KEYS[1], "available") or "0")
local reserved = tonumber(redis.call("HGET", KEYS[1], "reserved") or "0")
local qty = tonumber(ARGV[1])

-- Check if enough stock available
if available >= qty then
    redis.call("HINCRBY", KEYS[1], "available", -qty)
    redis.call("HINCRBY", KEYS[1], "reserved", qty)
    return 1  -- success
end

return 0  -- insufficient stock
