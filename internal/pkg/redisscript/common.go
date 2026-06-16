package redisscript

import "github.com/redis/go-redis/v9"

var DeleteLockScript = redis.NewScript(`
local key = KEYS[1]
local token = ARGV[1]

local result = redis.call("GET", key)

if result == nil then
	return 1
end

if result ~= token then
	return 0
end

redis.call("DEL", key)

return 1
`)
