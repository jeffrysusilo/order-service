-- Commit reserved stock (final deduction)
-- KEYS[1] = inventory key
-- ARGV[1] = quantity to commit

local reserved = tonumber(redis.call("HGET", KEYS[1], "reserved") or "0")
local qty = tonumber(ARGV[1])

-- Just reduce reserved count (already deducted from available)
if reserved >= qty then
    redis.call("HINCRBY", KEYS[1], "reserved", -qty)
    return 1  -- success
end

return 0  -- not enough reserved
