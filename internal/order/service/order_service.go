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

	idempotencyDomain "order_system/internal/idempotency/domain"
	idempotencyRepository "order_system/internal/idempotency/repository"
	productDomain "order_system/internal/product/domain"
	productRepository "order_system/internal/product/repository"

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
	lockKey := rediskey.IdempotencyLockKey(idempotencyKey)
	lockToken := rediskey.IdempotencyLockToken()

	err := os.idempotencyRedisLock.GetLock(ctx, lockKey, lockToken)

	if err != nil {
		// 이미 락이 있는 경우 "처리 중" 반환
		if errors.Is(err, rediserr.ErrLockExists) {
			return nil, fmt.Errorf("create order: %w: %w", err, ErrOrderAlreadyProcessed)
		}

		return nil, fmt.Errorf("create order: %w", err)
	}

	// 락을 획득했다면 종료 시점에 락을 해제
	defer func() {
		// 본 ctx와 같이 쓰면 제대로 동작하지 않기 때문에, ctx 분리
		cleanUpCtx, cleanUpCancel := context.WithTimeoutCause(
			context.WithoutCancel(parentCtx),
			2*time.Second,
			ErrDeleteOrderLockTimeout,
		)
		defer cleanUpCancel()

		deleteLockErr := os.idempotencyRedisLock.DeleteLock(cleanUpCtx, lockKey, lockToken)

		if deleteLockErr != nil {
			switch {
			case errors.Is(deleteLockErr, rediserr.ErrLockNotOwned):
				os.slackSender.Send(cleanUpCtx, notification.Message{
					Channel: notification.ChannelSlack,
					To:      "slack bot",
					Title:   "",
					Body:    fmt.Sprintf("idempotency lock not owned, lockKey:%s, err: %s", lockKey, deleteLockErr.Error()),
				})
				os.logger.ErrorContext(cleanUpCtx, "idempotency lock not owned", "lockKey", lockKey, "err", deleteLockErr)
			case errors.Is(cleanUpCtx.Err(), context.DeadlineExceeded):
				os.slackSender.Send(cleanUpCtx, notification.Message{
					Channel: notification.ChannelSlack,
					To:      "slack bot",
					Title:   "",
					Body:    fmt.Sprintf("delete idempotency lock timeout, lockKey:%s, err: %s", lockKey, context.Cause(cleanUpCtx).Error()),
				})
				os.logger.ErrorContext(cleanUpCtx, "delete idempotency lock timeout", "lockKey", lockKey, "err", context.Cause(cleanUpCtx))
			default:
				os.slackSender.Send(cleanUpCtx, notification.Message{
					Channel: notification.ChannelSlack,
					To:      "slack bot",
					Title:   "",
					Body:    fmt.Sprintf("delete idempotency lock failed, lockKey:%s, err: %s", lockKey, deleteLockErr.Error()),
				})
				os.logger.ErrorContext(cleanUpCtx, "delete idempotency lock failed", "lockKey", lockKey, "err", deleteLockErr)
			}
		}
	}()

	// 멱등성 검사, 재고 반영 및 기존 응답 반환
	var response *domain.Resource
	response, err = os.validateIdempotencyAndReturnResponse(
		ctx,
		claims.UserID,
		idempotencyDomain.ScopeOrderCreated,
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

			os.restoreReservedQuantity(cleanUpCtx, dto.OrderedItems)
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

	// db에도 예약 재고 업데이트
	// 오류 발생 시에 로그
	os.updateInventoryReservedQuantity(ctx, dto.OrderedItems)

	return resource, nil
}

type requestedProduct struct {
	name      string
	unitPrice uint64
	quantity  int
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

		if product.Status != productDomain.StatusActive {
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
	scope idempotencyDomain.Scope,
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
		if errors.Is(err, idempotencyRepository.ErrIdempotencyHashMismatch) {
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

		if errors.Is(err, productRepository.ErrRedisInvalidQuantity) {
			return fmt.Errorf(
				"product id: %d, invalid input quantity: %w : %w",
				productID,
				err,
				serviceerr.ErrInvalidArgument,
			)
		}

		if errors.Is(err, productRepository.ErrRedisInsufficientQuantity) {
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

// restoreReservedQuantity 예약 재고 복구
func (os *OrderService) restoreReservedQuantity(ctx context.Context, orderItems []domain.OrderedItem) {
	var keys []string
	var args []interface{}

	for _, o := range orderItems {
		keys = append(keys, rediskey.ProductInventoryKey(o.ProductID))
		args = append(args, -o.Quantity)
	}

	err := os.inventoryReservation.UpdateReservedQuantityInRedis(ctx, keys, args)

	if err != nil {
		switch {
		case errors.Is(err, rediserr.ErrNotFound):
			os.logger.ErrorContext(
				ctx,
				"can not found updated product",
				"err", err,
				"items", orderItems,
			)
		case errors.Is(err, context.DeadlineExceeded):
			os.logger.ErrorContext(
				ctx,
				"restore reserved quantity timeout",
				"err", err,
			)
		default:
			os.logger.ErrorContext(
				ctx,
				"restore reserved quantity failed",
				"err", err,
				"items", orderItems,
			)
		}
	}
}

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
		createOrderErr := tx.Orders().Create(ctx, orderEntity)

		if createOrderErr != nil {
			if errors.Is(createOrderErr, repository.ErrDuplicateOrderNo) {
				return fmt.Errorf("create order: %w: %w", createOrderErr, serviceerr.ErrConflict)
			}

			return fmt.Errorf("create order: %w", createOrderErr)
		}

		// 주문 품목 저장
		orderItemsEntity := dto.ToCreateOrderItemsEntity(orderEntity.ID)

		createOrderItemsEntity := tx.OrderItems().CreateRows(ctx, orderItemsEntity)

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
		updateIdempotencyErr := tx.Idempotencies().Update(
			ctx,
			userID,
			idempotencyKey,
			idempotencyDomain.ScopeOrderCreated,
			map[string]interface{}{
				"request_hash":  requestHash,
				"status":        idempotencyDomain.StatusSucceeded,
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

		return nil
	})

	if err != nil {
		return nil, err
	}

	return resource, nil
}

// updateInventoryReservedQuantity 예약 재고 수정
func (os *OrderService) updateInventoryReservedQuantity(ctx context.Context, orderItems []domain.OrderedItem) {
	for _, o := range orderItems {
		UpdateErr := os.orderStore.UpdateInventoryReservedQuantity(
			ctx,
			o.ProductID,
			map[string]interface{}{"reserved_quantity": o.Quantity},
		)

		if UpdateErr != nil {
			_ = os.slackSender.Send(ctx, notification.Message{
				Channel: notification.ChannelSlack,
				To:      "slack bot",
				Title:   "",
				Body:    fmt.Sprintf("update inventory failed: %s", UpdateErr.Error()),
			})
			os.logger.ErrorContext(ctx, "update inventory failed", "msg", UpdateErr.Error())
		}
	}
}
