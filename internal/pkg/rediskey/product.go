package rediskey

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	InventoryLockTTL = 10 * time.Second
	productKeyPrefix = "products:"
)

func ProductInventoryKey(productId uint) string {
	return productKeyPrefix + strconv.Itoa(int(productId)) + "inventory"
}

func InventoryLockKey(productId uint) string {
	return "lock:" + ProductInventoryKey(productId)
}

func InventoryLockToken() string {
	newUUID, _ := uuid.NewV7()
	return "inventory:" + newUUID.String()
}
