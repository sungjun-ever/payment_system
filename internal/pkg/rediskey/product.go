package rediskey

import "strconv"

const (
	productKeyPrefix = "products:"
)

func ProductInventoryKey(productId uint) string {
	return productKeyPrefix + strconv.Itoa(int(productId)) + "inventory"
}

func InventoryLockKey(productId uint) string {
	return ProductInventoryKey(productId) + ":lock"
}
