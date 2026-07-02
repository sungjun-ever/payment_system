package repository

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	idempotencyrepository "order_system/internal/idempotency/repository"
	orderdomain "order_system/internal/order/domain"
	orderrepository "order_system/internal/order/repository"
	paymentport "order_system/internal/payment"

	"gorm.io/gorm"
)

type paymentStore struct {
	mysql               *gorm.DB
	paymentRepo         PaymentGormRepository
	attemptRepo         PaymentAttemptGormRepository
	orderRepo           orderrepository.OrderGormRepository
	idempotencyGormRepo idempotencyrepository.IdempotencyGormRepository
}

func NewPaymentStore(
	db *gorm.DB,
	paymentRepo PaymentGormRepository,
	attemptRepo PaymentAttemptGormRepository,
	orderRepo orderrepository.OrderGormRepository,
	idempotencyRepo idempotencyrepository.IdempotencyGormRepository,
) paymentport.PaymentStore {
	return &paymentStore{
		mysql:               db,
		paymentRepo:         paymentRepo,
		attemptRepo:         attemptRepo,
		orderRepo:           orderRepo,
		idempotencyGormRepo: idempotencyRepo,
	}
}

func (p paymentStore) ValidateIdempotency(
	ctx context.Context,
	userID uint,
	idempotencyKey string,
	hashedRequestBody string,
) (*idempotencydomain.IdempotencyKey, error) {
	return p.idempotencyGormRepo.Validate(
		ctx,
		userID,
		idempotencydomain.ScopePayOrder,
		idempotencyKey,
		hashedRequestBody,
	)
}

func (p paymentStore) FindOrderForPayment(ctx context.Context, orderID uint) (*orderdomain.Order, error) {
	return p.orderRepo.Find(ctx, orderID)
}

func (p paymentStore) Tx(ctx context.Context, txFn func(tx paymentport.PayTx) error) error {
	return p.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&paymentTx{
			paymentWriter:     &PaymentGormRepository{Mysql: tx},
			paymentReader:     &PaymentGormRepository{Mysql: tx},
			attemptWriter:     &PaymentAttemptGormRepository{Mysql: tx},
			attemptReader:     &PaymentAttemptGormRepository{Mysql: tx},
			orderWriter:       &orderrepository.OrderGormRepository{Mysql: tx},
			orderReader:       &orderrepository.OrderGormRepository{Mysql: tx},
			idempotencyWriter: p.idempotencyGormRepo.WithTx(tx),
			idempotencyReader: &idempotencyrepository.IdempotencyGormRepository{Mysql: tx},
		})
	})
}

type paymentTx struct {
	paymentWriter     paymentport.PaymentWriter
	paymentReader     paymentport.PaymentReader
	attemptWriter     paymentport.AttemptWriter
	attemptReader     paymentport.AttemptReader
	orderWriter       paymentport.OrderWrite
	orderReader       paymentport.OrderReader
	idempotencyWriter paymentport.IdempotencyWrite
	idempotencyReader paymentport.IdempotencyReader
}

func (tx *paymentTx) PaymentsWriter() paymentport.PaymentWriter {
	return tx.paymentWriter
}
func (tx *paymentTx) PaymentsReader() paymentport.PaymentReader { return tx.paymentReader }
func (tx *paymentTx) AttemptsWriter() paymentport.AttemptWriter { return tx.attemptWriter }
func (tx *paymentTx) AttemptsReader() paymentport.AttemptReader { return tx.attemptReader }
func (tx *paymentTx) IdempotenciesWriter() paymentport.IdempotencyWrite {
	return tx.idempotencyWriter
}
func (tx *paymentTx) IdempotenciesReader() paymentport.IdempotencyReader {
	return tx.idempotencyReader
}
func (tx *paymentTx) OrdersWriter() paymentport.OrderWrite {
	return tx.orderWriter
}
func (tx *paymentTx) OrdersReader() paymentport.OrderReader { return tx.orderReader }
