package service

import (
	"testing"

	idempotencydomain "order_system/internal/idempotency/domain"
	"order_system/internal/order/domain"
)

func TestOrderService_CancelOrder(t *testing.T) {
	fixture := newOrderServiceFixture(t)
	fixture.idempotencyReader.result = &idempotencydomain.IdempotencyKey{}
	fixture.orderItemReader.items = []*domain.OrderItem{
		{OrderID: 1, ProductID: 1, Quantity: 2},
		{OrderID: 1, ProductID: 2, Quantity: 1},
	}

	got, err := fixture.svc.CancelOrder(fixture.ctx, 1, "ORD-1", 2, "cancel-key")
	if err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}

	if got == nil {
		t.Fatal("CancelOrder() returned nil resource")
	}

	if got.Message != "success" {
		t.Errorf("CancelOrder() message = %q, want %q", got.Message, "success")
	}

	if fixture.orderWriter.cancelledID != 1 || fixture.orderWriter.cancelledOrderNo != "ORD-1" || fixture.orderWriter.cancelledUserID != 2 {
		t.Errorf("CancelIfPendingByOrderAndUserID() arguments = (%d, %q, %d), want (1, %q, 2)",
			fixture.orderWriter.cancelledID, fixture.orderWriter.cancelledOrderNo, fixture.orderWriter.cancelledUserID, "ORD-1")
	}

	if fixture.inventoryWriter.restored[1] != 2 || fixture.inventoryWriter.restored[2] != 1 {
		t.Errorf("RestoreReservedQuantity() = %#v, want product 1:2 and product 2:1",
			fixture.inventoryWriter.restored)
	}

	if fixture.idempotencyWriter.cancelFields["status"] != idempotencydomain.StatusSucceeded ||
		fixture.idempotencyWriter.cancelFields["response_code"] != 200 {
		t.Errorf("CancelIfProcessing() fields = %#v, want succeeded response with code 200",
			fixture.idempotencyWriter.cancelFields)
	}

	if fixture.inventoryReservation.restoreOrderID != 1 || len(fixture.inventoryReservation.restoreItems) != 2 {
		t.Errorf("Redis inventory restore = orderID:%d items:%#v, want orderID:1 with 2 items",
			fixture.inventoryReservation.restoreOrderID, fixture.inventoryReservation.restoreItems)
	}

	if fixture.idempotencyRedisLock.getCalls != 1 || fixture.idempotencyRedisLock.deleteCalls != 1 {
		t.Errorf("idempotency lock calls = get:%d delete:%d, want get:1 delete:1",
			fixture.idempotencyRedisLock.getCalls, fixture.idempotencyRedisLock.deleteCalls)
	}
}
