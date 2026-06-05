package service

import (
	"context"
	"errors"
	"fmt"
	idempotencyDto "payment_system/internal/dto/idempotency"
	orderDto "payment_system/internal/dto/order"
	"payment_system/internal/model"
	"payment_system/internal/repository"

	"gorm.io/gorm"
)

type OrderService struct {
	orderRepo      repository.OrderRepository
	orderItemRepo  repository.OrderItemRepository
	idempotencySvc IdempotencyService
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	idempotencySvc IdempotencyService,
) OrderService {
	return OrderService{orderRepo, orderItemRepo, idempotencySvc}
}

func (os *OrderService) CreateOrder(
	ctx context.Context,
	idempotencyKey string,
	requestHash string,
	entity orderDto.CreateRequest,
) (*orderDto.Resource, error) {
	// 이미 있는 주문인지 확인하고, 기존에 있는 주문이라면 같은 응답을 리턴한다.
	// 없는 주문이라면 주문을 생성한다.
	cmd, err := entity.ToCommand()

	if err != nil {
		return nil, fmt.Errorf("convert order create request to command failed: %w", err)
	}

	request := idempotencyDto.CreateRequest{
		UserID:      cmd.UserID,
		Scope:       model.IdempotencyOrderCreated,
		Key:         idempotencyKey,
		RequestHash: requestHash,
		Status:      model.IdempotencyProcessing,
	}

	existingResponse := &orderDto.Resource{}
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
	createOrder := &model.Order{
		OrderNo:     cmd.OrderNo,
		UserID:      cmd.UserID,
		Status:      model.OrderStatusPending,
		TotalAmount: cmd.TotalAmount,
		OrderedAt:   cmd.OrderedAt,
	}
	var response *orderDto.Resource

	err = os.orderRepo.Transaction(func(tx *gorm.DB) error {
		innerErr := os.orderRepo.Create(ctx, tx, createOrder)

		if innerErr != nil {
			return innerErr
		}

		createOrderItems := make([]model.OrderItem, len(cmd.OrderedItems))

		for i, item := range cmd.OrderedItems {
			createOrderItems[i] = model.OrderItem{
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

		response = &orderDto.Resource{
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
		if errors.Is(err, repository.ErrDuplicateOrderNo) {
			return nil, fmt.Errorf("db: order_no duplicated: %w", ErrConflict)
		}
		return nil, fmt.Errorf("create order failed: %w", err)
	}

	return response, nil
}
