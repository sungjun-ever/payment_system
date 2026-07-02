package rediskey

import (
	"strconv"
	"time"
)

const (
	InventoryLockTTL = 10 * time.Second
	productKeyPrefix = "products:"
)

// ParseProductID 레디스키에서 id 파싱
func ParseProductID(key string) []int {
	ids := make([]int, 0, 10)

	start := 0
	for end := 0; end < len(key); end++ {
		if key[end] == '{' {
			start = end + 1
		} else if key[end] == '}' {
			id, _ := strconv.Atoi(key[start:end])
			ids = append(ids, id)
		}
	}

	return ids
}

func ProductKey(productId uint) string {
	return productKeyPrefix + "{" + strconv.Itoa(int(productId)) + "}"
}

func ProductInventoryKey(productId uint) string {
	return ProductKey(productId) + ":inventory"
}

func InventoryRestoreDoneKey(orderNo string, productID uint, operation string) string {
	return "inventory-restore-done:" + orderNo + ":" + strconv.Itoa(int(productID)) + ":" + operation
}
