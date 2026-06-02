package redisscript

import "github.com/redis/go-redis/v9"

var DeleteTokenScript = redis.NewScript(`
-- refresh key, blacklist key
local refreshKey = KEYS[1]
local blacklistKey = KEYS[2]
local ttl = ARGV[1]

redis.call("DEL", refreshKey)

if ttl > 0 then
	redis.call("SET", blacklistKey, "1", "PX", ttl)
end

return 1
`)
