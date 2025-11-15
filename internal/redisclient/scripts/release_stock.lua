-- Release reserved stock (compensation/rollback)
-- KEYS[1] = inventory key
-- ARGV[1] = quantity to release

local reserved = tonumber(redis.call("HGET", KEYS[1], "reserved") or "0")
local qty = tonumber(ARGV[1])

-- Return stock to available pool
redis.call("HINCRBY", KEYS[1], "available", qty)
redis.call("HINCRBY", KEYS[1], "reserved", -qty)

return 1  -- success
