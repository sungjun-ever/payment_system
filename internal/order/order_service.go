package order

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/idempotency"
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/rediskey"
	"payment_system/internal/product"

	"gorm.io/gorm"
)

type OrderService struct {
	orderRepo      OrderRepository
	orderItemRepo  OrderItemRepository
	inventoryRepo  product.InventoryRepository
	idempotencySvc idempotency.IdempotencyService
}

func NewOrderService(
	orderRepo OrderRepository,
	orderItemRepo OrderItemRepository,
	inventoryRepo product.InventoryRepository,
	idempotencySvc idempotency.IdempotencyService,
) OrderService {
	return OrderService{orderRepo, orderItemRepo, inventoryRepo, idempotencySvc}
}

func (os *OrderService) CreateOrder(
	ctx context.Context,
	idempotencyKey string,
	requestHash string,
	entity CreateRequest,
) (*Resource, error) {
	cmd, err := entity.ToCommand()

	if err != nil {
		return nil, fmt.Errorf("convert order create request to command failed: %w", err)
	}

	// 멱등키 request 생성
	request := os.newIdempotencyRequest(cmd, idempotencyKey, requestHash)

	// 이미 있는 주문인지 확인하고, 기존에 있는 주문이라면 같은 응답을 리턴한다.
	existingResponse, err := os.findExistingOrderResponse(ctx, request)

	if err != nil {
		return nil, fmt.Errorf("find existing order response failed: %w", err)
	}

	if existingResponse != nil {
		return existingResponse, nil
	}

	// 재고 검증 로직
	err = os.reserveInventories(ctx, *cmd)

	if err != nil {
		return nil, fmt.Errorf("reserve inventories failed: %w", err)
	}

	response, err := os.createOrderTransaction(ctx, *cmd, request)
	if err != nil {
		return nil, fmt.Errorf("create order transaction failed: %w", err)
	}

	return response, nil
}

func (os *OrderService) newIdempotencyRequest(
	cmd *CreateCommand,
	idempotencyKey string,
	requestHash string,
) idempotency.CreateRequest {
	request := idempotency.CreateRequest{
		UserID:      cmd.UserID,
		Scope:       idempotency.ScopeOrderCreated,
		Key:         idempotencyKey,
		RequestHash: requestHash,
		Status:      idempotency.StatusProcessing,
	}

	return request
}

func (os *OrderService) findExistingOrderResponse(
	ctx context.Context,
	request idempotency.CreateRequest,
) (*Resource, error) {
	existingResponse := &Resource{}
	exists, _, err := os.idempotencySvc.CheckExistingResponse(ctx, request, existingResponse)

	if err != nil {
		return nil, err
	}

	// 있으면 기존 정보 리턴
	if exists {
		return existingResponse, nil
	}

	return nil, nil
}

func (os *OrderService) reserveInventories(ctx context.Context, cmd CreateCommand) error {

	// 상품과 예약 수량을 맵 형태로 변환
	productQuantities := make(map[uint]uint)
	for _, item := range cmd.OrderedItems {
		productQuantities[item.ProductID] += item.Quantity
	}

	keys := make([]string, 0, len(productQuantities))
	args := make([]interface{}, 0, len(productQuantities))

	for k, v := range productQuantities {
		keys = append(keys, rediskey.ProductKey(k))
		args = append(args, v)
	}

	// 레디스에서 남은 재고 확인 및 예약 재고 증가
	err := os.inventoryRepo.ValidateAndUpdateReservedQuantity(ctx, keys, args...)

	// 재고 확인이 안되는 경우 DB에서 정보를 가져와 redis에 넣는다.
	if err != nil && errors.Is(err, product.ErrRedisInventoryNotFound) {
		// lock을 획득
		// 로직을 실행하기전 마지막으로 redis에 정보가 없는지 확인하고
		// DB 조회 -> redis에 적재
		for _, item := range cmd.OrderedItems {
			//lockKey := rediskey.InventoryLockKey(item.ProductID)
			//newUUID, err := uuid.NewV7()
			//
			//if err != nil {
			//	return fmt.Errorf("create uuid failed: %w", err)
			//}

			//lockToken := "inventory-" + newUUID.String()
			//exist, err := os.inventoryRepo.GetInventoryLock(ctx, lockKey, lockToken)
			//
			//if err != nil {
			//	return fmt.Errorf("get inventory lock failed: %w", err)
			//}
			//
			//// 다른 락이 이미 있는 경우
			//if exist {
			//
			//}

			// 락을 획득한 경우
			// 다시 redis를 확인한다.
			inventory, err := os.inventoryRepo.FindByProductID(ctx, item.ProductID)

			if err != nil {
				if errors.Is(err, product.ErrInventoryNotFound) {
					return fmt.Errorf("inventory not found: %w", err)
				}

				return fmt.Errorf("find inventory by product id failed: %w", err)
			}

			err = os.inventoryRepo.StoreInRedis(ctx, rediskey.ProductKey(item.ProductID), map[string]interface{}{
				"total_quantity":    inventory.TotalQuantity,
				"reserved_quantity": inventory.ReservedQuantity,
				"sold_quantity":     inventory.SoldQuantity,
			})

			if err != nil {
				if errors.Is(err, product.ErrRedisInventoryAlreadyExists) {

				}
			}

		}
	} else if err != nil {
		// 재고 부족으로 불가능
		if errors.Is(err, product.ErrRedisInsufficientQuantity) {
			return fmt.Errorf("insufficient quantity: %w", err)
		}

		// 요청 재고 수량이 유효하지 않은 경우
		if errors.Is(err, product.ErrRedisInvalidQuantity) {
			return fmt.Errorf("invalid quantity: %w", err)
		}

		return fmt.Errorf("valid and update inventory failed: %w", err)
	}

	return nil
}

func (os *OrderService) createOrderTransaction(ctx context.Context, cmd CreateCommand, request idempotency.CreateRequest) (*Resource, error) {
	// 없으면 생성 작업
	createOrder := &Order{
		OrderNo:     cmd.OrderNo,
		UserID:      cmd.UserID,
		Status:      StatusPending,
		TotalAmount: cmd.TotalAmount,
		OrderedAt:   cmd.OrderedAt,
	}
	var response *Resource

	err := os.orderRepo.Transaction(func(tx *gorm.DB) error {
		innerErr := os.orderRepo.Create(ctx, tx, createOrder)

		if innerErr != nil {
			return innerErr
		}

		createOrderItems := make([]OrderItem, len(cmd.OrderedItems))

		for i, item := range cmd.OrderedItems {
			createOrderItems[i] = OrderItem{
				OrderID:     createOrder.ID,
				ProductID:   item.ProductID,
				ProductName: item.ProductName,
				UnitPrice:   item.UnitPrice,
				Quantity:    item.Quantity,
				TotalPrice:  item.TotalPrice,
			}
		}

		innerErr = os.orderItemRepo.Create(ctx, tx, createOrderItems)

		if innerErr != nil {
			return innerErr
		}

		response = &Resource{
			ID:          createOrder.ID,
			OrderNo:     createOrder.OrderNo,
			Status:      createOrder.Status,
			TotalAmount: createOrder.TotalAmount,
			OrderedAt:   createOrder.OrderedAt,
		}

		request.OrderID = &createOrder.ID
		request.ResponseCode = 201
		request.Response = response

		innerErr = os.idempotencySvc.CreateRecord(ctx, tx, request)

		if innerErr != nil {
			return innerErr
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrDuplicateOrderNo) {
			return nil, fmt.Errorf("order_no duplicated: %w", apperr.ErrConflict)
		}
		return nil, fmt.Errorf("create order failed: %w", err)
	}

	return response, nil
}
