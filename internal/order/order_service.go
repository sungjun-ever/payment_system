package order

import (
	"context"
	"errors"
	"fmt"
	"payment_system/internal/idempotency"
	"payment_system/internal/pkg/apperr/serviceerr"

	"gorm.io/gorm"
)

type OrderService struct {
	orderRepo      OrderRepository
	orderItemRepo  OrderItemRepository
	idempotencySvc idempotency.IdempotencyService
}

func NewOrderService(
	orderRepo OrderRepository,
	orderItemRepo OrderItemRepository,
	idempotencySvc idempotency.IdempotencyService,
) OrderService {
	return OrderService{orderRepo, orderItemRepo, idempotencySvc}
}

func (os *OrderService) CreateOrder(
	ctx context.Context,
	idempotencyKey string,
	requestHash string,
	entity CreateRequest,
) (*Resource, error) {
	// 이미 있는 주문인지 확인하고, 기존에 있는 주문이라면 같은 응답을 리턴한다.
	// 없는 주문이라면 주문을 생성한다.
	cmd, err := entity.ToCommand()

	if err != nil {
		return nil, fmt.Errorf("convert order create request to command failed: %w", err)
	}

	request := idempotency.CreateRequest{
		UserID:      cmd.UserID,
		Scope:       idempotency.ScopeOrderCreated,
		Key:         idempotencyKey,
		RequestHash: requestHash,
		Status:      idempotency.StatusProcessing,
	}

	existingResponse := &Resource{}
	exists, responseCode, err := os.idempotencySvc.CheckExistingResponse(ctx, request, existingResponse)

	// 있으면 기존 정보 리턴
	if err != nil {
		return nil, err
	}

	if exists {
		if responseCode == 0 {
			responseCode = 200
		}

		return existingResponse, nil
	}

	// 없으면 생성 작업
	createOrder := &Order{
		OrderNo:     cmd.OrderNo,
		UserID:      cmd.UserID,
		Status:      StatusPending,
		TotalAmount: cmd.TotalAmount,
		OrderedAt:   cmd.OrderedAt,
	}
	var response *Resource

	err = os.orderRepo.Transaction(func(tx *gorm.DB) error {
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
			return nil, fmt.Errorf("db: order_no duplicated: %w", serviceerr.ErrConflict)
		}
		return nil, fmt.Errorf("create order failed: %w", err)
	}

	return response, nil
}
