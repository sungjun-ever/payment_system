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

var RestoreReservedQuantitiesScript = redis.NewScript(`
local n = #KEYS / 2
local ttl = tonumber(ARGV[n + 1])
local results = {}

for i = 1, n do
	local inventoryKey = KEYS[i]
	local doneKey = KEYS[n + i]
	local quantity = tonumber(ARGV[i])

	if quantity == nil or quantity <= 0 then
		table.insert(results, {-1,i}) -- 입력값 오류
	elseif redis.call("EXISTS", doneKey) == 1 then
		table.insert(results, {2, i}) -- 이미 예약 재고 복구 완료
	elseif redis.call("HGET", inventoryKey, "reserved_quantity") == nil then
		table.insert(results, {0, i}) -- 예약 재고 없음
	else
		redis.call("HINCRBY", inventoryKey, "reserved_quantity", -quantity)
		redis.call("SET", doneKey, "1", "EX", ttl)
		table.insert(results, {1, i})
	end
end

return results
`)

var RestoreReservedQuantityScript = redis.NewScript(`
local inventoryKey = KEYS[1]
local doneKey = KEYS[2]
local quantity = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])

if quantity == nil or quantity <= 0 then
	return -1
elseif redis.call("EXISTS", doneKey) == 1 then
	return 2
elseif redis.call("HGET", inventoryKey, "reserved_quantity") == nil then
	return 0
else
	redis.call("HINCRBY", inventoryKey, "reserved_quantity", -quantity)
	redis.call("SET", doneKey, "1", "EX", ttl)
	return 1
end

`)
