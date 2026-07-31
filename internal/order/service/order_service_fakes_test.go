package service

import (
	"context"

	idempotencydomain "order_system/internal/idempotency/domain"
	"order_system/internal/notification"
	orderport "order_system/internal/order"
	"order_system/internal/order/domain"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"
)

type fakeOrderUnitOfWork struct {
	orderWriter        orderport.OrderWriter
	orderReader        orderport.OrderReader
	orderItemWriter    orderport.OrderItemWriter
	orderItemReader    orderport.OrderItemReader
	idempotencyWriter  orderport.IdempotencyWriter
	inventoryWriter    orderport.InventoryWriter
	inventoryJobWriter orderport.InventoryJobWriter
}

func (f *fakeOrderUnitOfWork) Tx(ctx context.Context, fn func(orderport.OrderTx) error) error {
	return fn(fakeOrderTx{
		f.orderWriter,
		f.orderReader,
		f.orderItemWriter,
		f.orderItemReader,
		f.idempotencyWriter,
		f.inventoryWriter,
		f.inventoryJobWriter},
	)
}

type fakeOrderTx struct {
	orderWriter        orderport.OrderWriter
	orderReader        orderport.OrderReader
	orderItemWriter    orderport.OrderItemWriter
	orderItemReader    orderport.OrderItemReader
	idempotencyWriter  orderport.IdempotencyWriter
	inventoryWriter    orderport.InventoryWriter
	inventoryJobWriter orderport.InventoryJobWriter
}

func (f fakeOrderTx) OrderWriter() orderport.OrderWriter               { return f.orderWriter }
func (f fakeOrderTx) OrderReader() orderport.OrderReader               { return f.orderReader }
func (f fakeOrderTx) OrderItemWriter() orderport.OrderItemWriter       { return f.orderItemWriter }
func (f fakeOrderTx) OrderItemReader() orderport.OrderItemReader       { return f.orderItemReader }
func (f fakeOrderTx) IdempotencyWriter() orderport.IdempotencyWriter   { return f.idempotencyWriter }
func (f fakeOrderTx) InventoryWriter() orderport.InventoryWriter       { return f.inventoryWriter }
func (f fakeOrderTx) InventoryJobWriter() orderport.InventoryJobWriter { return f.inventoryJobWriter }

type fakeOrderWriter struct {
	created                      *domain.Order
	cancelledID, cancelledUserID uint
	cancelledOrderNo             string
}

func (f *fakeOrderWriter) Create(ctx context.Context, order *domain.Order) error {
	order.ID = 1
	order.Status = domain.StatusPending
	f.created = order
	return nil
}
func (f *fakeOrderWriter) CancelIfPendingByOrderID(ctx context.Context, orderID uint) (bool, error) {
	return true, nil
}
func (f *fakeOrderWriter) CancelIfPendingByOrderAndUserID(ctx context.Context, id uint, no string, userID uint) (bool, error) {
	f.cancelledID, f.cancelledOrderNo, f.cancelledUserID = id, no, userID
	return true, nil
}

type fakeOrderReader struct{}

func (f *fakeOrderReader) Find(ctx context.Context, id uint) (*domain.Order, error) { return nil, nil }

type fakeOrderItemWriter struct{ created []domain.OrderItem }

func (f *fakeOrderItemWriter) CreateRows(ctx context.Context, items []domain.OrderItem) error {
	f.created = items
	return nil
}

type fakeOrderItemReader struct{ items []*domain.OrderItem }

func (f *fakeOrderItemReader) GetItemsByOrderID(ctx context.Context, orderID uint) ([]*domain.OrderItem, error) {
	return f.items, nil
}

type fakeIdempotencyWriter struct {
	updatedFields, cancelFields map[string]interface{}
	cancelOK                    bool
}

func (f *fakeIdempotencyWriter) Update(ctx context.Context, userID uint, key string, scope idempotencydomain.Scope, fields map[string]interface{}) error {
	f.updatedFields = fields
	return nil
}
func (f *fakeIdempotencyWriter) CancelIfProcessing(ctx context.Context, orderID, userID uint, key string, scope idempotencydomain.Scope, fields map[string]interface{}) (bool, error) {
	f.cancelFields = fields
	return f.cancelOK, nil
}

type fakeInventoryWriter struct {
	updated  []uint
	restored map[uint]int
}

func (f *fakeInventoryWriter) RestoreReservedQuantity(
	ctx context.Context,
	productID uint,
	fields map[string]interface{},
) error {
	f.restored[productID] += fields["reserved_quantity"].(int)
	return nil
}
func (f *fakeInventoryWriter) UpdateReservedQuantity(
	ctx context.Context,
	productID uint,
	fields map[string]interface{},
) error {
	f.updated = append(f.updated, productID)
	return nil
}

type fakeInventoryJobWriter struct{}

func (f *fakeInventoryJobWriter) CreateJob(ctx context.Context, fields productdomain.InventoryJobCreateContext) error {
	return nil
}

type fakeProductReader struct {
	products map[uint]*productdomain.Product
}

func (f *fakeProductReader) Find(ctx context.Context, id uint) (*productdomain.Product, error) {
	return f.products[id], nil
}

type fakeIdempotencyReader struct {
	result *idempotencydomain.IdempotencyKey
}

func (f *fakeIdempotencyReader) Validate(
	ctx context.Context,
	userID uint,
	scope idempotencydomain.Scope,
	key, hash string,
) (*idempotencydomain.IdempotencyKey, error) {
	return f.result, nil
}

type fakeIdempotencyRedisLock struct{ getCalls, deleteCalls int }

func (f *fakeIdempotencyRedisLock) GetLock(ctx context.Context, key, token string) error {
	f.getCalls++
	return nil
}
func (f *fakeIdempotencyRedisLock) DeleteLock(ctx context.Context, key, token string) error {
	f.deleteCalls++
	return nil
}

type fakeInventoryReservation struct {
	restoreOrderID uint
	restoreItems   []productrepository.RestoreItem
}

func (f *fakeInventoryReservation) ValidateAndUpdateReservedQuantity(
	ctx context.Context,
	keys []string,
	args ...interface{},
) (uint, error) {
	return 0, nil
}
func (f *fakeInventoryReservation) RestoreProductsReservedQuantityInRedis(
	ctx context.Context,
	orderNo string,
	orderID uint,
	items []productrepository.RestoreItem,
) []productrepository.RestoreFailed {
	f.restoreOrderID, f.restoreItems = orderID, items
	return nil
}

type fakeSlackSender struct{}

func (f *fakeSlackSender) Send(ctx context.Context, msg notification.Message) error { return nil }
