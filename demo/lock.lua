if redis.call("get", KEYS[1]) == ARGV[1]
then
    -- 刷新过期时间
    return redis.call("pexpire", KEYS[1], ARGV[2])
else
    -- 设置 key value 和过期时间
    return redis.call("set", KEYS[1], ARGV[1], "NX", "PX", ARGV[2])
end