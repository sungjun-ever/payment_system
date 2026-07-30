package service

import (
	"context"
	"io"
	"log/slog"
	idempotencydomain "order_system/internal/idempotency/domain"
	"order_system/internal/notification"
	orderport "order_system/internal/order"
	"order_system/internal/order/domain"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"
	"testing"
	"time"
)

func TestOrderService_CreateOrder(t *testing.T) {
	ctx := context.Background()

	orderWriter := &fakeOrderWriter{}
	orderReader := &fakeOrderReader{}
	orderItemWriter := &fakeOrderItemWriter{}
	orderItemReader := &fakeOrderItemReader{}
	idempotencyWriter := &fakeIdempotencyWriter{}
	inventoryWriter := &fakeInventoryWriter{}
	inventoryJobWriter := &fakeInventoryJobWriter{}

	orderTx := &fakeOrderUnitOfWork{
		orderWriter:        orderWriter,
		orderReader:        orderReader,
		orderItemWriter:    orderItemWriter,
		orderItemReader:    orderItemReader,
		idempotencyWriter:  idempotencyWriter,
		inventoryWriter:    inventoryWriter,
		inventoryJobWriter: inventoryJobWriter,
	}

	productReader := &fakeProductReader{}
	idempotencyReader := &fakeIdempotencyReader{}
	idempotencyRedisLock := &fakeIdempotencyRedisLock{}
	inventoryReservation := &fakeInventoryReservation{}
	slackSender := &fakeSlackSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := NewOrderService(
		logger,
		orderTx,
		productReader,
		idempotencyReader,
		idempotencyRedisLock,
		inventoryReservation,
		slackSender,
	)

	idempotencyKey := "idempotency_key"
	requestHash := "request_hash"
	dto := domain.CreateRequest{
		UserID:      1,
		OrderNo:     "ORD_123124124",
		TotalAmount: 20000,
		OrderedAt:   time.Now().String(),
		OrderedItems: []domain.OrderedItem{
			{
				ProductID:   1,
				ProductName: "Test Product Name1",
				UnitPrice:   10000,
				Quantity:    1,
				TotalPrice:  10000,
			},
			{
				ProductID:   2,
				ProductName: "Test Product Name2",
				UnitPrice:   10000,
				Quantity:    1,
				TotalPrice:  10000,
			},
		},
	}

	got, err := svc.CreateOrder(ctx, idempotencyKey, requestHash, dto)

	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	if got.OrderNo != dto.OrderNo {
		t.Errorf("CreateOrder() OrderNo = %v, want %v", got.OrderNo, dto.OrderNo)
	}

	if got.Status != domain.StatusPending {
		t.Errorf("CreateOrder() Status = %v, want %v", got.Status, domain.StatusPending)
	}

	if got.TotalAmount != dto.TotalAmount {
		t.Errorf("CreateOrder() TotalAmount = %v, want %v", got.TotalAmount, dto.TotalAmount)
	}
	
}

type fakeOrderUnitOfWork struct {
	orderWriter        orderport.OrderWriter
	orderReader        orderport.OrderReader
	orderItemWriter    orderport.OrderItemWriter
	orderItemReader    orderport.OrderItemReader
	idempotencyWriter  orderport.IdempotencyWriter
	inventoryWriter    orderport.InventoryWriter
	inventoryJobWriter orderport.InventoryJobWriter
}

func (f fakeOrderUnitOfWork) Tx(ctx context.Context, txFn func(tx orderport.OrderTx) error) error {
	//TODO implement me
	panic("implement me")
}

type fakeOrderWriter struct {
}

func (f fakeOrderWriter) Create(ctx context.Context, order *domain.Order) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeOrderWriter) CancelIfPendingByOrderID(ctx context.Context, orderID uint) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (f fakeOrderWriter) CancelIfPendingByOrderAndUserID(ctx context.Context, id uint, orderNo string, userID uint) (bool, error) {
	//TODO implement me
	panic("implement me")
}

type fakeOrderReader struct {
}

func (f fakeOrderReader) Find(ctx context.Context, id uint) (*domain.Order, error) {
	//TODO implement me
	panic("implement me")
}

type fakeOrderItemWriter struct {
}

func (f fakeOrderItemWriter) CreateRows(ctx context.Context, orderItems []domain.OrderItem) error {

	//TODO implement me
	panic("implement me")
}

type fakeOrderItemReader struct {
}

func (f fakeOrderItemReader) GetItemsByOrderID(ctx context.Context, orderID uint) ([]*domain.OrderItem, error) {
	//TODO implement me
	panic("implement me")
}

type fakeIdempotencyWriter struct {
}

func (f fakeIdempotencyWriter) Update(ctx context.Context, userID uint, key string, scope idempotencydomain.Scope, fields map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeIdempotencyWriter) CancelIfProcessing(ctx context.Context, orderID uint, userID uint, idempotencyKey string, scope idempotencydomain.Scope, fields map[string]interface{}) (bool, error) {
	//TODO implement me
	panic("implement me")
}

type fakeInventoryWriter struct {
}

func (f fakeInventoryWriter) RestoreReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeInventoryWriter) UpdateReservedQuantity(ctx context.Context, productID uint, fields map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

type fakeInventoryJobWriter struct {
}

func (f fakeInventoryJobWriter) CreateJob(ctx context.Context, fields productdomain.InventoryJobCreateContext) error {
	//TODO implement me
	panic("implement me")
}

type fakeProductReader struct {
}

func (f fakeProductReader) Find(ctx context.Context, id uint) (*productdomain.Product, error) {
	//TODO implement me
	panic("implement me")
}

type fakeIdempotencyReader struct {
}

func (f fakeIdempotencyReader) Validate(ctx context.Context, userID uint, scope idempotencydomain.Scope, idempotencyKey string, hashedRequestBody string) (*idempotencydomain.IdempotencyKey, error) {
	//TODO implement me
	panic("implement me")
}

type fakeIdempotencyRedisLock struct {
}

func (f fakeIdempotencyRedisLock) GetLock(ctx context.Context, lockKey string, token string) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeIdempotencyRedisLock) DeleteLock(ctx context.Context, lockKey string, token string) error {
	//TODO implement me
	panic("implement me")
}

type fakeInventoryReservation struct {
}

func (f fakeInventoryReservation) ValidateAndUpdateReservedQuantity(ctx context.Context, keys []string, args ...interface{}) (uint, error) {
	//TODO implement me
	panic("implement me")
}

func (f fakeInventoryReservation) RestoreProductsReservedQuantityInRedis(ctx context.Context, orderNo string, orderID uint, items []productrepository.RestoreItem) []productrepository.RestoreFailed {
	//TODO implement me
	panic("implement me")
}

type fakeSlackSender struct {
}

func (f fakeSlackSender) Send(ctx context.Context, msg notification.Message) error {
	//TODO implement me
	panic("implement me")
}
