package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"order_system/internal/notification"
	"order_system/internal/order"
	"time"

	idempotencydomain "order_system/internal/idempotency/domain"
	idempotencyrepository "order_system/internal/idempotency/repository"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"

	"order_system/internal/order/domain"
	"order_system/internal/order/repository"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/pkg/apperr/rediserr"
	"order_system/internal/pkg/apperr/serviceerr"
	"order_system/internal/pkg/rediskey"
	"order_system/internal/pkg/token"
)

var (
	ErrIdempotencyKeyNotFound         = errors.New("idempotency key not found")
	ErrRequestHashMismatch            = errors.New("request hash mismatch")
	ErrOrderAlreadyProcessed          = errors.New("order already processed")
	ErrInsufficientProductQuantity    = errors.New("insufficient product quantity")
	ErrDeleteOrderLockTimeout         = errors.New("delete order lock timeout")
	ErrRestoreReservedQuantityTimeout = errors.New("restore reserved quantity timeout")
)

type OrderService struct {
	logger               *slog.Logger
	orderStore           order.OrderStore
	idempotencyRedisLock order.IdempotencyLock
	inventoryReservation order.InventoryReservation
	slackSender          notification.Sender
}

func NewOrderService(
	logger *slog.Logger,
	orderStore order.OrderStore,
	idempotencyRedisLock order.IdempotencyLock,
	inventoryReservation order.InventoryReservation,
	slackSender notification.Sender,
) OrderService {
	return OrderService{
		logger:               logger,
		orderStore:           orderStore,
		idempotencyRedisLock: idempotencyRedisLock,
		inventoryReservation: inventoryReservation,
		slackSender:          slackSender,
	}
}

type requestedProduct struct {
	name      string
	unitPrice uint64
	quantity  int
}

type retryRestoreContext struct {
	OrderNo     string
	OrderID     uint
	ProductID   uint
	Quantity    int
	NextRetryAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type restorePayload struct {
	OrderNo   string
	OrderID   uint
	ProductID uint
	Quantity  int
}

// CreateOrder 주문 생성
func (os *OrderService) CreateOrder(
	parentCtx context.Context,
	claims *token.AccessClaims,
	idempotencyKey string,
	requestHash string,
	dto domain.CreateRequest,
) (*domain.Resource, error) {
	// 전체 로직 타임아웃 10초 지정
	ctx, cancel := context.WithTimeoutCause(parentCtx, 10*time.Second, serviceerr.ErrTimeout)
	defer cancel()

	// 요청 중복 방지를 위한 락 생성
	unlock, err := os.acquireOrderLock(ctx, idempotencyKey)
	if err != nil {
		return nil, err
	}
	defer unlock()

	// 멱등성 검사, 재고 반영 및 기존 응답 반환
	var response *domain.Resource
	response, err = os.validateIdempotencyAndReturnResponse(
		ctx,
		claims.UserID,
		idempotencydomain.ScopeOrderCreated,
		idempotencyKey,
		requestHash,
	)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response, nil
	}

	// 저장된 응답 본문이 없다면 주문 생성 시작
	// 요청 금액과 실제 상품 정보 유효성 검사
	err = os.validateOrderRequest(ctx, dto)
	if err != nil {
		return nil, err
	}

	// 먼저 재고 유효성 검사 및 예약 재고 반영을 한다.
	err = os.reserveProductsQuantity(ctx, dto.OrderedItems)
	if err != nil {
		return nil, err
	}

	// 재고 복구가 필요하다면 재고 복구 로직
	needRestoreInventory := false
	var orderID uint
	defer func() {
		// 재고 복구가 필요한 경우 복구 진행
		if needRestoreInventory {
			// 본 ctx와 같이 쓰면 제대로 동작하지 않기 때문에, ctx 분리
			cleanUpCtx, cleanUpCancel := context.WithTimeoutCause(
				context.WithoutCancel(parentCtx),
				2*time.Second,
				ErrRestoreReservedQuantityTimeout,
			)
			defer cleanUpCancel()

			restoreItems := make(map[uint]int)
			for _, item := range dto.OrderedItems {
				restoreItems[item.ProductID] += item.Quantity
			}

			os.restoreReservedQuantityInRedis(cleanUpCtx, dto.OrderNo, 0, restoreItems)
		}
	}()

	// 주문 생성, 주문 품목 생성, 멱등성 수정 트랜잭션을 시작한다
	resource, transactionErr := os.createOrderTransaction(ctx, dto, claims.UserID, idempotencyKey, requestHash)

	if transactionErr != nil {
		needRestoreInventory = true

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("create order timeout: %w", context.Cause(ctx))
		}

		return nil, fmt.Errorf("create order transaction failed: %w", transactionErr)
	}
	orderID = resource.ID
	// 10분 이내로 결제를 진행하지 않는 경우, 재고 원상 복구
	go os.cancelOrderIfNotPaidAfter(
		context.WithoutCancel(parentCtx),
		claims.UserID,
		orderID,
		dto.OrderedItems,
		10*time.Minute,
	)

	return resource, nil
}

// CancelOrder 주문 취소
func (os *OrderService) CancelOrder(
	parentCtx context.Context,
	orderID uint,
	orderNo string,
	userID uint,
) (*domain.CancelResource, error) {
	ctx, cancel := context.WithTimeoutCause(parentCtx, 5*time.Second, serviceerr.ErrTimeout)
	defer cancel()

	if err := os.cancelledOrderAndRestoreReservedQuantity(ctx, orderID, orderNo, userID); err != nil {
		return &domain.CancelResource{Message: "failed"}, err
	}

	return &domain.CancelResource{Message: "success"}, nil
}

func (os *OrderService) acquireOrderLock(parentCtx context.Context, idempotencyKey string) (func(), error) {
	lockKey := rediskey.IdempotencyLockKey(idempotencyKey)
	lockToken := rediskey.IdempotencyLockToken()

	if err := os.getLock(parentCtx, lockKey, lockToken); err != nil {
		return nil, err
	}

	return func() {
		os.deleteLock(parentCtx, lockKey, lockToken)
	}, nil
}

func (os *OrderService) getLock(ctx context.Context, lockKey, lockToken string) error {
	err := os.idempotencyRedisLock.GetLock(ctx, lockKey, lockToken)

	if err != nil {
		// 이미 락이 있는 경우 "처리 중" 반환
		if errors.Is(err, rediserr.ErrLockExists) {
			return fmt.Errorf("create order: %w: %w", err, ErrOrderAlreadyProcessed)
		}

		return fmt.Errorf("create order: %w", err)
	}

	return nil
}

func (os *OrderService) deleteLock(ctx context.Context, lockKey string, lockToken string) {
	defer func() {
		// 본 ctx와 같이 쓰면 제대로 동작하지 않기 때문에, ctx 분리
		cleanUpCtx, cleanUpCancel := context.WithTimeoutCause(
			context.WithoutCancel(ctx),
			2*time.Second,
			ErrDeleteOrderLockTimeout,
		)
		defer cleanUpCancel()

		deleteLockErr := os.idempotencyRedisLock.DeleteLock(cleanUpCtx, lockKey, lockToken)

		if deleteLockErr != nil {
			switch {
			case errors.Is(deleteLockErr, rediserr.ErrLockNotOwned):
				_ = os.slackSender.Send(cleanUpCtx, notification.Message{
					Channel: notification.ChannelSlack,
					To:      "slack bot",
					Title:   "",
					Body:    fmt.Sprintf("idempotency lock not owned, lockKey:%s, err: %s", lockKey, deleteLockErr.Error()),
				})
				os.logger.ErrorContext(cleanUpCtx, "idempotency lock not owned", "lockKey", lockKey, "err", deleteLockErr)
			case errors.Is(cleanUpCtx.Err(), context.DeadlineExceeded):
				_ = os.slackSender.Send(cleanUpCtx, notification.Message{
					Channel: notification.ChannelSlack,
					To:      "slack bot",
					Title:   "",
					Body:    fmt.Sprintf("delete idempotency lock timeout, lockKey:%s, err: %s", lockKey, context.Cause(cleanUpCtx).Error()),
				})
				os.logger.ErrorContext(cleanUpCtx, "delete idempotency lock timeout", "lockKey", lockKey, "err", context.Cause(cleanUpCtx))
			default:
				_ = os.slackSender.Send(cleanUpCtx, notification.Message{
					Channel: notification.ChannelSlack,
					To:      "slack bot",
					Title:   "",
					Body:    fmt.Sprintf("delete idempotency lock failed, lockKey:%s, err: %s", lockKey, deleteLockErr.Error()),
				})
				os.logger.ErrorContext(cleanUpCtx, "delete idempotency lock failed", "lockKey", lockKey, "err", deleteLockErr)
			}
		}
	}()
}

// validateOrderRequest 요청 금액과 실제 상품 정보 유효성 검사
func (os *OrderService) validateOrderRequest(ctx context.Context, dto domain.CreateRequest) error {
	products, err := os.validateOrderAmount(dto)

	if err != nil {
		return err
	}

	return os.validateProducts(ctx, products)
}

// validateOrderAmount 요청 상품별 금액과 주문 총액 유효성 검사
func (os *OrderService) validateOrderAmount(dto domain.CreateRequest) (map[uint]requestedProduct, error) {
	if len(dto.OrderedItems) == 0 {
		return nil, fmt.Errorf("ordered items is empty: %w", serviceerr.ErrInvalidArgument)
	}

	products := make(map[uint]requestedProduct)
	var totalAmount uint64

	for _, item := range dto.OrderedItems {
		if item.ProductID == 0 || item.ProductName == "" || item.UnitPrice == 0 || item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid order item: %w", serviceerr.ErrInvalidArgument)
		}

		itemTotalPrice, err := multiplyOrderAmount(item.UnitPrice, item.Quantity)

		if err != nil {
			return nil, err
		}

		if itemTotalPrice != item.TotalPrice {
			return nil, fmt.Errorf(
				"product id: %d, total price mismatch: %w",
				item.ProductID,
				serviceerr.ErrInvalidArgument,
			)
		}

		totalAmount, err = addOrderAmount(totalAmount, item.TotalPrice)

		if err != nil {
			return nil, err
		}

		product, exists := products[item.ProductID]

		if exists {
			if product.name != item.ProductName || product.unitPrice != item.UnitPrice {
				return nil, fmt.Errorf(
					"product id: %d, duplicated product info mismatch: %w",
					item.ProductID,
					serviceerr.ErrInvalidArgument,
				)
			}

			maxInt := int(^uint(0) >> 1)

			if product.quantity > maxInt-item.Quantity {
				return nil, fmt.Errorf(
					"product id: %d, quantity overflow: %w",
					item.ProductID,
					serviceerr.ErrInvalidArgument,
				)
			}

			product.quantity += item.Quantity
			products[item.ProductID] = product
			continue
		}

		products[item.ProductID] = requestedProduct{
			name:      item.ProductName,
			unitPrice: item.UnitPrice,
			quantity:  item.Quantity,
		}
	}

	if totalAmount != dto.TotalAmount {
		return nil, fmt.Errorf("order total amount mismatch: %w", serviceerr.ErrInvalidArgument)
	}

	return products, nil
}

// validateProducts 실제 상품 상태, 이름, 가격 유효성 검사
func (os *OrderService) validateProducts(ctx context.Context, products map[uint]requestedProduct) error {
	for productID, requested := range products {
		product, err := os.orderStore.FindProduct(ctx, productID)

		if err != nil {
			if errors.Is(err, dberr.ErrNotFound) {
				return fmt.Errorf(
					"product id: %d, not found: %w: %w",
					productID,
					err,
					serviceerr.ErrResourceNotFound,
				)
			}

			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf(
					"product id: %d, find product timeout: %w: %w",
					productID,
					context.Cause(ctx),
					serviceerr.ErrTimeout,
				)
			}

			return fmt.Errorf("product id: %d, find product failed: %w", productID, err)
		}

		if product.Status != productdomain.StatusActive {
			return fmt.Errorf(
				"product id: %d, inactive product: %w",
				productID,
				serviceerr.ErrInvalidArgument,
			)
		}

		if product.Name != requested.name {
			return fmt.Errorf(
				"product id: %d, product name mismatch: %w",
				productID,
				serviceerr.ErrInvalidArgument,
			)
		}

		if product.Price < 0 || uint64(product.Price) != requested.unitPrice {
			return fmt.Errorf(
				"product id: %d, product price mismatch: %w",
				productID,
				serviceerr.ErrInvalidArgument,
			)
		}
	}

	return nil
}

func multiplyOrderAmount(unitPrice uint64, quantity int) (uint64, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("invalid quantity: %w", serviceerr.ErrInvalidArgument)
	}

	quantityUint := uint64(quantity)

	if unitPrice > ^uint64(0)/quantityUint {
		return 0, fmt.Errorf("order item total price overflow: %w", serviceerr.ErrInvalidArgument)
	}

	return unitPrice * quantityUint, nil
}

func addOrderAmount(totalAmount uint64, itemTotalPrice uint64) (uint64, error) {
	if totalAmount > ^uint64(0)-itemTotalPrice {
		return 0, fmt.Errorf("order total amount overflow: %w", serviceerr.ErrInvalidArgument)
	}

	return totalAmount + itemTotalPrice, nil
}

// validateIdempotencyAndReturnResponse 멱등성을 확인 한 뒤, 기존 응답이 있다면 리턴
func (os *OrderService) validateIdempotencyAndReturnResponse(
	ctx context.Context,
	userID uint,
	scope idempotencydomain.Scope,
	idempotencyKey string,
	requestHash string,
) (*domain.Resource, error) {
	savedIdempotency, err := os.orderStore.ValidateIdempotency(
		ctx,
		userID,
		scope,
		idempotencyKey,
		requestHash,
	)

	if err != nil {
		// 저장된 멱등키가 없다면 오류
		if errors.Is(err, dberr.ErrNotFound) {
			return nil, fmt.Errorf("create order: %w: %w", err, ErrIdempotencyKeyNotFound)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("create order timeout: %w", context.Cause(ctx))
		}

		// 저장된 멱등키가 있고 저장된 요청 본문 해시와 현재 요청의 해시가 다르면 오류
		if errors.Is(err, idempotencyrepository.ErrIdempotencyHashMismatch) {
			return nil, fmt.Errorf("create order: %w: %w", err, ErrRequestHashMismatch)
		}

		return nil, fmt.Errorf("create order: %w", err)
	}

	if savedIdempotency.ResponseBody != nil {
		var response domain.Resource
		err = json.Unmarshal([]byte(*savedIdempotency.ResponseBody), &response)

		if err != nil {
			return nil, fmt.Errorf("marshal idempotency response failed: %w", err)
		}

		return &response, nil
	}

	return nil, nil
}

// reserveProductsQuantity 레디스에 있는 상품 재고 검증 및 예약 재고 반영
func (os *OrderService) reserveProductsQuantity(ctx context.Context, orderItems []domain.OrderedItem) error {
	var keys []string
	var args []interface{}

	maps := make(map[string]int)
	// 맵에 담아 중복 상품 재고 합산
	for _, o := range orderItems {
		key := rediskey.ProductInventoryKey(o.ProductID)
		maps[key] += o.Quantity
	}

	for k, v := range maps {
		keys = append(keys, k)
		args = append(args, v)
	}

	productID, err := os.inventoryReservation.ValidateAndUpdateReservedQuantity(ctx, keys, args)

	if err != nil {
		if errors.Is(err, rediserr.ErrNotFound) {
			return fmt.Errorf(
				"product id: %d, not found in redis: %w : %w",
				productID,
				err,
				ErrInsufficientProductQuantity,
			)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf(
				"product id: %d, redis timeout: %w : %w",
				productID,
				err,
				serviceerr.ErrTimeout,
			)
		}

		if errors.Is(err, productrepository.ErrRedisInvalidQuantity) {
			return fmt.Errorf(
				"product id: %d, invalid input quantity: %w : %w",
				productID,
				err,
				serviceerr.ErrInvalidArgument,
			)
		}

		if errors.Is(err, productrepository.ErrRedisInsufficientQuantity) {
			return fmt.Errorf(
				"product id: %d, insufficient quantity: %w : %w",
				productID,
				err,
				ErrInsufficientProductQuantity,
			)
		}
	}
	return nil
}

// restoreReservedQuantityInRedis 예약 재고 복구
func (os *OrderService) restoreReservedQuantityInRedis(
	ctx context.Context,
	orderNo string,
	orderID uint,
	productQuantities map[uint]int,
) {
	var items []productrepository.RestoreItem
	for product, quantity := range productQuantities {
		items = append(items, productrepository.RestoreItem{
			ProductID: product,
			Quantity:  quantity,
		})
	}

	// key 생성을 레포지토리에서 진행한다
	fails := os.inventoryReservation.RestoreProductsReservedQuantityInRedis(ctx, orderNo, orderID, items)

	// 실패한 재고 복구를 재시도에 등록한다.
	if len(fails) > 0 {
		var retryCtx []retryRestoreContext
		for _, fail := range fails {
			switch {
			// 예약 재고가 확인되지 않는 경우 로그 및 알림 처리
			// TODO 레디스에 재고 등록 및 재시도 등록
			case errors.Is(fail.Err, productrepository.ErrRedisNoneReservedQuantity):
				os.logger.ErrorContext(
					ctx,
					"restore reserved quantity failed",
					"fail info", fail,
				)
				_ = os.slackSender.Send(ctx, notification.Message{
					Channel: notification.ChannelSlack,
					To:      "slack bot",
					Title:   "",
					Body:    fmt.Sprintf("restore reserved quantity failed - fail:%v", fail),
				})
			case errors.Is(fail.Err, productrepository.ErrRedisInvalidQuantity):
				os.logger.ErrorContext(
					ctx,
					"restore reserved quantity failed",
					"fail info", fail,
				)
				_ = os.slackSender.Send(ctx, notification.Message{
					Channel: notification.ChannelSlack,
					To:      "slack bot",
					Title:   "",
					Body:    fmt.Sprintf("restore reserved quantity failed - fail:%v", fail),
				})
			default:
				os.logger.WarnContext(
					ctx,
					"restore reserved quantity failed",
					"fail info", fail,
				)
				retryCtx = append(retryCtx, retryRestoreContext{
					OrderNo:     fail.OrderNo,
					OrderID:     fail.OrderID,
					ProductID:   fail.ProductID,
					Quantity:    fail.Quantity,
					NextRetryAt: time.Now(),
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				})
			}
		}

		if len(retryCtx) > 0 {
			os.registerRedisRestoreInventoryRetry(ctx, retryCtx)
		}
	}
}

// createOrderTransaction 주문 생성 트랜잭션 시작
func (os *OrderService) createOrderTransaction(
	ctx context.Context,
	dto domain.CreateRequest,
	userID uint,
	idempotencyKey string,
	requestHash string,
) (*domain.Resource, error) {
	var returnOrder *domain.Order
	var resource *domain.Resource

	err := os.orderStore.Tx(ctx, func(tx order.OrderTx) error {
		orderEntity, toOrderEntityErr := dto.ToCreateOrderEntity(userID)

		if toOrderEntityErr != nil {
			return toOrderEntityErr
		}
		returnOrder = orderEntity

		// 주문 저장
		createOrderErr := tx.OrderWriters().Create(ctx, orderEntity)

		if createOrderErr != nil {
			if errors.Is(createOrderErr, repository.ErrDuplicateOrderNo) {
				return fmt.Errorf("create order: %w: %w", createOrderErr, serviceerr.ErrConflict)
			}

			return fmt.Errorf("create order: %w", createOrderErr)
		}

		// 주문 품목 저장
		orderItemsEntity := dto.ToCreateOrderItemsEntity(orderEntity.ID)

		createOrderItemsEntity := tx.OrderItemWriters().CreateRows(ctx, orderItemsEntity)

		if createOrderItemsEntity != nil {
			return fmt.Errorf("create order items: %w", createOrderItemsEntity)
		}

		// 반환 응답
		resource = &domain.Resource{
			ID:           returnOrder.ID,
			OrderNo:      returnOrder.OrderNo,
			Status:       returnOrder.Status,
			TotalAmount:  returnOrder.TotalAmount,
			OrderedAt:    returnOrder.OrderedAt,
			OrderedItems: dto.OrderedItems,
		}

		marshal, errMarshal := json.Marshal(resource)

		if errMarshal != nil {
			return fmt.Errorf("marshal create order response body failed: %w", errMarshal)
		}

		// 멱등성 정보 수정
		updateIdempotencyErr := tx.IdempotencyWriters().Update(
			ctx,
			userID,
			idempotencyKey,
			idempotencydomain.ScopeOrderCreated,
			map[string]interface{}{
				"request_hash":  requestHash,
				"status":        idempotencydomain.StatusSucceeded,
				"order_id":      orderEntity.ID,
				"response_body": string(marshal),
				"response_code": 201,
			},
		)

		if updateIdempotencyErr != nil {
			if errors.Is(updateIdempotencyErr, dberr.ErrNotFound) {
				return fmt.Errorf("update idempotency failed: %w: %w", updateIdempotencyErr, ErrIdempotencyKeyNotFound)
			}
			return fmt.Errorf("update idempotency failed: %w", updateIdempotencyErr)
		}

		// db 예약 재고 증가
		for _, o := range dto.OrderedItems {
			UpdateErr := tx.InventoryWriters().UpdateReservedQuantity(
				ctx,
				o.ProductID,
				map[string]interface{}{"reserved_quantity": o.Quantity},
			)

			if UpdateErr != nil {
				return fmt.Errorf("update inventory failed: %w", UpdateErr)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return resource, nil
}

// cancelOrderIfNotPaidAfter 결제를 진행하지 않으면 자동으로 취소
func (os *OrderService) cancelOrderIfNotPaidAfter(
	parentCtx context.Context,
	userID uint,
	orderID uint,
	items []domain.OrderedItem,
	ttl time.Duration,
) {
	timer := time.NewTimer(ttl)
	defer timer.Stop()

	<-timer.C

	ctx, cancel := context.WithTimeoutCause(parentCtx, 5*time.Second, serviceerr.ErrTimeout)
	defer cancel()

	// ttl 이후에도 결제가 아직 pending 상태라면 주문 취소 후 재고 복구
	err := os.cancelPendingOrderAndRestoreInventory(ctx, userID, orderID, items)
	if err != nil {
		os.logger.ErrorContext(ctx, "cancel pending order failed", "err", err)
		_ = os.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body:    fmt.Sprintf("cancel pending order failed, order no: %d, err: %s", orderID, err.Error()),
		})
		return
	}
}

// cancelPendingOrderAndRestoreInventory 결제를 진행하지 않은 주문 취소
func (os *OrderService) cancelPendingOrderAndRestoreInventory(
	ctx context.Context,
	userID uint,
	orderID uint,
	items []domain.OrderedItem,
) error {
	err := os.orderStore.Tx(ctx, func(tx order.OrderTx) error {
		// 주문 취소
		ok, err := tx.OrderWriters().CancelIfPendingByOrderID(ctx, orderID)

		if err != nil {
			return err
		}

		if !ok {
			return nil
		}

		// 재고 복구
		for _, item := range items {
			err = tx.InventoryWriters().RestoreReservedQuantity(
				ctx,
				item.ProductID,
				map[string]interface{}{
					"reserved_quantity": item.Quantity,
				},
			)

			if err != nil {
				return err
			}
		}

		// 멱등키 취소
		ok, err = tx.IdempotencyWriters().CancelIfProcessingByOrderIDAndUserID(ctx, orderID, userID)

		if err != nil {
			return err
		}

		if !ok {
			return fmt.Errorf("idempotency key: %d-%d-%s, not found: %w",
				orderID, userID, idempotencydomain.StatusProcessing, serviceerr.ErrResourceNotFound)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// redis 재고 복구 로직 진행
	restoreItems := make(map[uint]int)
	for _, item := range items {
		restoreItems[item.ProductID] += item.Quantity
	}

	os.restoreReservedQuantityInRedis(ctx, "", orderID, restoreItems)

	return nil
}

// cancelledOrderAndRestoreReservedQuantity 주문 및 멱등키 취소 처리, DB 재고 복구
func (os *OrderService) cancelledOrderAndRestoreReservedQuantity(
	ctx context.Context,
	orderID uint,
	orderNo string,
	userID uint,
) error {
	var orderItems []*domain.OrderItem
	err := os.orderStore.Tx(ctx, func(tx order.OrderTx) error {
		// 주문 취소
		ok, err := tx.OrderWriters().CancelIfPendingByOrderAndUserID(ctx, orderID, orderNo, userID)

		if err != nil {
			return fmt.Errorf("order constraint: %d-%s-%d-%s, cancel order failed: %w",
				orderID, orderNo, userID, domain.StatusPending, err)
		}

		if !ok {
			return fmt.Errorf("order constraint: %d-%s-%d-%s, not found: %w",
				orderID, orderNo, userID, domain.StatusPending, serviceerr.ErrResourceNotFound)
		}

		// 멱등키 취소
		ok, err = tx.IdempotencyWriters().CancelIfProcessingByOrderIDAndUserID(ctx, orderID, userID)

		if err != nil {
			return fmt.Errorf("idempotency constraint: %d-%d-%s, cancel idempotency failed: %w",
				orderID, userID, idempotencydomain.StatusProcessing, err)
		}

		if !ok {
			return fmt.Errorf("idempotency constraint: %d-%d-%s, not found: %w",
				orderID, userID, idempotencydomain.StatusProcessing, serviceerr.ErrResourceNotFound)
		}

		// 주문 품목을 가져옴
		items, err := os.orderStore.GetOrderItems(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order items failed: %w", err)
		}
		orderItems = items
		// 예약 재고 복구
		for _, item := range items {
			err = tx.InventoryWriters().RestoreReservedQuantity(ctx, item.ProductID, map[string]interface{}{
				"reserved_quantity": item.Quantity,
			})

			if err != nil {
				return fmt.Errorf("update order item reserved quantity failed, items: %v, err: %w", items, err)
			}
		}

		return nil
	})

	if err != nil {
		os.logger.ErrorContext(ctx, "cancelled order failed", "err", err)
		_ = os.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body:    fmt.Sprintf("cancelled order failed, order id: %d, err: %s", orderID, err.Error()),
		})
		return err
	}

	// 트랜잭션이 성공했다면 레디스 재고도 복구
	restoreItems := make(map[uint]int)
	for _, item := range orderItems {
		restoreItems[item.ProductID] += item.Quantity
	}

	os.restoreReservedQuantityInRedis(ctx, "", orderID, restoreItems)

	return nil
}

// registerRedisRestoreInventoryRetry redis 예약 재고 복구 실패 재시도 등록
func (os *OrderService) registerRedisRestoreInventoryRetry(ctx context.Context, retryCtx []retryRestoreContext) {
	jobCtx, cancel := context.WithTimeoutCause(
		context.WithoutCancel(ctx),
		2*time.Second,
		serviceerr.ErrTimeout,
	)
	defer cancel()

	err := os.orderStore.Tx(jobCtx, func(tx order.OrderTx) error {
		for _, retry := range retryCtx {
			payload, _ := json.Marshal(restorePayload{
				OrderNo:   retry.OrderNo,
				OrderID:   retry.OrderID,
				ProductID: retry.ProductID,
				Quantity:  retry.Quantity,
			})

			var uniqueKey string
			if retry.OrderID != 0 {
				uniqueKey = fmt.Sprintf("%s-%s-%d-%d",
					productdomain.TargetRedis, productdomain.DecreaseReserved, retry.OrderID, retry.ProductID)
			} else {
				uniqueKey = fmt.Sprintf("%s-%s-%s-%d",
					productdomain.TargetRedis, productdomain.DecreaseReserved, retry.OrderNo, retry.ProductID)
			}

			err := tx.InventoryJobWriters().CreateJob(jobCtx, productdomain.InventoryJobCreateContext{
				Target:      productdomain.TargetRedis,
				Operation:   productdomain.DecreaseReserved,
				RetryCount:  1,
				Status:      productdomain.JobPending,
				Payload:     string(payload),
				UniqueKey:   uniqueKey,
				CreatedAt:   retry.CreatedAt,
				NextRetryAt: retry.NextRetryAt,
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		os.logger.ErrorContext(ctx, "failed to register redis restore inventory retry",
			"err", err,
			"items", retryCtx)
		_ = os.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body: fmt.Sprintf("failed to register redis restore inventory retry, err: %s, items: %v",
				err.Error(), retryCtx),
		})
	}
}
