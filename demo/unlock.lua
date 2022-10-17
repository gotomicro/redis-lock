-- 两个动作：1. 检测是不是预期中的值（也就是，是不是你的锁）；
-- 2. 如果是，删除；如果不是，返回一个值
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    -- 返回 0 代表的是 key 不在，或者值不对
    return 0
end