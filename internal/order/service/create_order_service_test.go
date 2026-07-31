package service

import (
	"testing"

	idempotencydomain "order_system/internal/idempotency/domain"
	"order_system/internal/order/domain"
	productdomain "order_system/internal/product/domain"
)

func TestOrderService_CreateOrder(t *testing.T) {
	fixture := newOrderServiceFixture(t)
	fixture.productReader.products = map[uint]*productdomain.Product{
		1: {Name: "Test Product Name1", Price: 10000, Status: productdomain.StatusActive},
		2: {Name: "Test Product Name2", Price: 10000, Status: productdomain.StatusActive},
	}
	fixture.idempotencyReader.result = &idempotencydomain.IdempotencyKey{}

	dto := domain.CreateRequest{
		UserID:      1,
		OrderNo:     "ORD_123124124",
		TotalAmount: 20000,
		OrderedAt:   "2026-05-12 10:00:00",
		OrderedItems: []domain.OrderedItem{
			{ProductID: 1, ProductName: "Test Product Name1", UnitPrice: 10000, Quantity: 1, TotalPrice: 10000},
			{ProductID: 2, ProductName: "Test Product Name2", UnitPrice: 10000, Quantity: 1, TotalPrice: 10000},
		},
	}

	got, err := fixture.svc.CreateOrder(fixture.ctx, "idempotency_key", "request_hash", dto)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if got == nil {
		t.Fatal("CreateOrder() returned nil resource")
	}
	if got.ID != 1 || got.OrderNo != dto.OrderNo || got.Status != domain.StatusPending || got.TotalAmount != dto.TotalAmount {
		t.Errorf("CreateOrder() = %+v, want ID=1, OrderNo=%q, Status=%q, TotalAmount=%d", got, dto.OrderNo, domain.StatusPending, dto.TotalAmount)
	}
	if fixture.orderWriter.created == nil {
		t.Fatal("OrderWriter.Create() was not called")
	}
	if len(fixture.orderItemWriter.created) != len(dto.OrderedItems) {
		t.Errorf("OrderItemWriter.CreateRows() item count = %d, want %d", len(fixture.orderItemWriter.created), len(dto.OrderedItems))
	}
	if fixture.idempotencyWriter.updatedFields["status"] != idempotencydomain.StatusSucceeded {
		t.Errorf("IdempotencyWriter.Update() status = %v, want %v", fixture.idempotencyWriter.updatedFields["status"], idempotencydomain.StatusSucceeded)
	}
	if len(fixture.inventoryWriter.updated) != len(dto.OrderedItems) {
		t.Errorf("InventoryWriter.UpdateReservedQuantity() calls = %d, want %d", len(fixture.inventoryWriter.updated), len(dto.OrderedItems))
	}
	if fixture.idempotencyRedisLock.getCalls != 1 || fixture.idempotencyRedisLock.deleteCalls != 1 {
		t.Errorf("idempotency lock calls = get:%d delete:%d, want get:1 delete:1", fixture.idempotencyRedisLock.getCalls, fixture.idempotencyRedisLock.deleteCalls)
	}
}
