package redisscript

import "github.com/redis/go-redis/v9"

var RotateRefreshTokenScript = redis.NewScript(`
local refreshKey = KEYS[1]
local cookieToken = ARGV[1]
local newRefreshToken = ARGV[2]
local ttl = tonumber(ARGV[3])

local originRefresh = redis.call("GET", refreshKey)

if not originRefresh then
	return 0
end

if originRefresh ~= cookieToken then
	return -1
end

redis.call("SET", refreshKey, newRefreshToken, "PX", ttl)

return 1
`)

var DeleteTokenAndBlacklistScript = redis.NewScript(`
local refreshKey = KEYS[1]
local blacklistKey = KEYS[2]
local ttl = tonumber(ARGV[1])

redis.call("DEL", refreshKey)

if ttl > 0 then
	redis.call("SET", blacklistKey, "1", "PX", ttl)
end

return 1
`)
