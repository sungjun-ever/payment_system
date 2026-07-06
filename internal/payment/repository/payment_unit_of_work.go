package repository

import (
	"context"
	idempotencydomain "order_system/internal/idempotency/domain"
	idempotencyrepository "order_system/internal/idempotency/repository"
	orderdomain "order_system/internal/order/domain"
	orderrepository "order_system/internal/order/repository"
	paymentport "order_system/internal/payment"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type paymentStore struct {
	mysql               *gorm.DB
	rds                 *redis.Client
	paymentRepo         PaymentGormRepository
	attemptRepo         PaymentAttemptGormRepository
	orderRepo           orderrepository.OrderGormRepository
	idempotencyGormRepo idempotencyrepository.IdempotencyGormRepository
}

func NewPaymentStore(
	db *gorm.DB,
	rds *redis.Client,
	paymentRepo PaymentGormRepository,
	attemptRepo PaymentAttemptGormRepository,
	orderRepo orderrepository.OrderGormRepository,
	idempotencyRepo idempotencyrepository.IdempotencyGormRepository,
) paymentport.PaymentStore {
	return &paymentStore{
		mysql:               db,
		rds:                 rds,
		paymentRepo:         paymentRepo,
		attemptRepo:         attemptRepo,
		orderRepo:           orderRepo,
		idempotencyGormRepo: idempotencyRepo,
	}
}

func (p *paymentStore) ValidateIdempotency(
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

func (p *paymentStore) GetItemsByOrderID(ctx context.Context, orderID uint) ([]*orderdomain.OrderItem, error) {
	return (&orderrepository.OrderItemGormRepository{Mysql: p.mysql}).GetItemsByOrderID(ctx, orderID)
}

func (p *paymentStore) FindOrderForPayment(ctx context.Context, orderID uint) (*orderdomain.Order, error) {
	return p.orderRepo.Find(ctx, orderID)
}

func (p *paymentStore) UpdateSoldQuantity(ctx context.Context, productID uint, quantity int) error {
	return (&productrepository.InventoryGormRepository{Mysql: p.mysql}).UpdateSoldQuantity(ctx, productID, quantity)
}

func (p *paymentStore) CreateJob(ctx context.Context, fields productdomain.InventoryJobCreateContext) error {
	return (&productrepository.InventoryJobGormRepository{Mysql: p.mysql}).CreateJob(ctx, fields)
}

func (p *paymentStore) SetConfirmSaleDoneKey(ctx context.Context, orderID uint, productID uint) error {
	return (&productrepository.InventoryRedisRepository{Rds: p.rds}).SetConfirmSaleDoneKey(ctx, orderID, productID)
}

func (p *paymentStore) GetConfirmSaleDoneKey(ctx context.Context, orderID uint, productID uint) (string, error) {
	return (&productrepository.InventoryRedisRepository{Rds: p.rds}).GetConfirmSaleDoneKey(ctx, orderID, productID)
}

func (p *paymentStore) CreateInventoryMovement(ctx context.Context, entity *productdomain.InventoryMovement) error {
	return (&productrepository.InventoryMovementGormRepository{Mysql: p.mysql}).CreateInventoryMovement(ctx, entity)
}

func (p *paymentStore) Tx(ctx context.Context, txFn func(tx paymentport.PayTx) error) error {
	return p.mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return txFn(&paymentTx{
			paymentWriter:           &PaymentGormRepository{Mysql: tx},
			paymentReader:           &PaymentGormRepository{Mysql: tx},
			attemptWriter:           &PaymentAttemptGormRepository{Mysql: tx},
			attemptReader:           &PaymentAttemptGormRepository{Mysql: tx},
			orderWriter:             &orderrepository.OrderGormRepository{Mysql: tx},
			orderReader:             &orderrepository.OrderGormRepository{Mysql: tx},
			idempotencyWriter:       p.idempotencyGormRepo.WithTx(tx),
			idempotencyReader:       &idempotencyrepository.IdempotencyGormRepository{Mysql: tx},
			orderItemReader:         &orderrepository.OrderItemGormRepository{Mysql: tx},
			inventoryWriter:         &productrepository.InventoryGormRepository{Mysql: tx},
			inventoryJobWriter:      &productrepository.InventoryJobGormRepository{Mysql: tx},
			inventoryMovementWriter: &productrepository.InventoryMovementGormRepository{Mysql: tx},
		})
	})
}

type paymentTx struct {
	paymentWriter           paymentport.PaymentWriter
	paymentReader           paymentport.PaymentReader
	attemptWriter           paymentport.AttemptWriter
	attemptReader           paymentport.AttemptReader
	orderWriter             paymentport.OrderWrite
	orderReader             paymentport.OrderReader
	idempotencyWriter       paymentport.IdempotencyWrite
	idempotencyReader       paymentport.IdempotencyReader
	orderItemReader         paymentport.OrderItemReader
	inventoryWriter         paymentport.InventoryWriter
	inventoryJobWriter      paymentport.InventoryJobWriter
	inventoryMovementWriter paymentport.InventoryMovementWriter
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
func (tx *paymentTx) OrdersReader() paymentport.OrderReader         { return tx.orderReader }
func (tx *paymentTx) OrderItemsReader() paymentport.OrderItemReader { return tx.orderItemReader }
func (tx *paymentTx) InventoryWriter() paymentport.InventoryWriter  { return tx.inventoryWriter }
func (tx *paymentTx) InventoryJobWriter() paymentport.InventoryJobWriter {
	return tx.inventoryJobWriter
}
func (tx *paymentTx) InventoryMovementWriter() paymentport.InventoryMovementWriter {
	return tx.inventoryMovementWriter
}
