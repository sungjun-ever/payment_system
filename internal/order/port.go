package order

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"
)

type OrderProductReader interface {
	Find(ctx context.Context, id uint) (*productdomain.Product, error)
}

type OrderIdempotencyReader interface {
	Validate(
		ctx context.Context,
		userID uint,
		scope idempotencydomain.Scope,
		idempotencyKey string,
		hashedRequestBody string,
	) (*idempotencydomain.IdempotencyKey, error)
}

type IdempotencyLock interface {
	GetLock(ctx context.Context, lockKey string, token string) error
	DeleteLock(ctx context.Context, lockKey string, token string) error
}

type InventoryReservation interface {
	ValidateAndUpdateReservedQuantity(ctx context.Context, keys []string, args ...interface{}) (uint, error)
	RestoreProductsReservedQuantityInRedis(
		ctx context.Context,
		orderNo string,
		orderID uint,
		items []productrepository.RestoreItem,
	) []productrepository.RestoreFailed
}
