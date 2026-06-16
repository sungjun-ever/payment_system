package redisscript

import "github.com/redis/go-redis/v9"

// ValidateAndUpdateReservedQuantityScript 예약재고 검증 및 반영
var ValidateAndUpdateReservedQuantityScript = redis.NewScript(`
local count = #KEYS

-- 전체 수량 검증
for i = 1, count do
	local key = KEYS[i]
	local quantity = tonumber(ARGV[i])

	local totalQuantity = redis.call("HGET", key, "total_quantity")
	local reservedQuantity = redis.call("HGET", key, "reserved_quantity")
	local soldQuantity = redis.call("HGET", key, "sold_quantity")

	-- 재고 없음
	if totalQuantity == nil or reservedQuantity == nil or soldQuantity == nil then
		return {0, i}
	end

	-- 잘못된 수량
	if quantity == nil or quantity <= 0 then
		return {-1, i}
	end

	-- 재고 부족
	local availableQuantity = tonumber(totalQuantity) - tonumber(reservedQuantity) - tonumber(soldQuantity)
	if availableQuantity < quantity then
		return {-2, i}
	end
end

-- 예약 재고 증가
for i = 1, count do
	local key = KEYS[i]
	local quantity = tonumber(ARGV[i])
	redis.call("HINCRBY", key, "reserved_quantity", quantity)
end

return {1, 0}
`)

var UpdateReservedQuantitiesScript = redis.NewScript(`
local count = #KEYS
for i = 1, count do
	local key = KEYS[i]
	local quantity = tonumber(ARGV[i])
	local result = redis.call("HGET", key, "reserved_quantity")

	if result == nil then
		return {0, i}
	end

	redis.call("HINCRBY", key, "reserved_quantity", quantity)
end

return {1, -1}
`)
