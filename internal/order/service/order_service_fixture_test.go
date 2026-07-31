package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

type orderServiceFixture struct {
	ctx                  context.Context
	svc                  *OrderService
	productReader        *fakeProductReader
	idempotencyReader    *fakeIdempotencyReader
	idempotencyRedisLock *fakeIdempotencyRedisLock
	inventoryReservation *fakeInventoryReservation
	slackSender          *fakeSlackSender

	orderWriter        *fakeOrderWriter
	orderItemWriter    *fakeOrderItemWriter
	orderItemReader    *fakeOrderItemReader
	idempotencyWriter  *fakeIdempotencyWriter
	inventoryWriter    *fakeInventoryWriter
	inventoryJobWriter *fakeInventoryJobWriter
}

func newOrderServiceFixture(t *testing.T) *orderServiceFixture {
	t.Helper()

	orderWriter := &fakeOrderWriter{}
	orderItemWriter := &fakeOrderItemWriter{}
	orderItemReader := &fakeOrderItemReader{}
	idempotencyWriter := &fakeIdempotencyWriter{cancelOK: true}
	inventoryWriter := &fakeInventoryWriter{restored: make(map[uint]int)}
	inventoryJobWriter := &fakeInventoryJobWriter{}

	orderTx := &fakeOrderUnitOfWork{
		orderWriter:        orderWriter,
		orderReader:        &fakeOrderReader{},
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

	return &orderServiceFixture{
		ctx:                  context.Background(),
		svc:                  NewOrderService(slog.New(slog.NewTextHandler(io.Discard, nil)), orderTx, productReader, idempotencyReader, idempotencyRedisLock, inventoryReservation, slackSender),
		productReader:        productReader,
		idempotencyReader:    idempotencyReader,
		idempotencyRedisLock: idempotencyRedisLock,
		inventoryReservation: inventoryReservation,
		slackSender:          slackSender,
		orderWriter:          orderWriter,
		orderItemWriter:      orderItemWriter,
		orderItemReader:      orderItemReader,
		idempotencyWriter:    idempotencyWriter,
		inventoryWriter:      inventoryWriter,
		inventoryJobWriter:   inventoryJobWriter,
	}
}
