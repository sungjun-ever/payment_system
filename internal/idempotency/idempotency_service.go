package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"payment_system/internal/pkg/apperr/serviceerr"

	"gorm.io/gorm"
)

type IdempotencyService struct {
	idempotencyRepo IdempotencyKeyRepository
}

func NewIdempotencyService(idempotencyRepo IdempotencyKeyRepository) IdempotencyService {
	return IdempotencyService{idempotencyRepo: idempotencyRepo}
}

func (s *IdempotencyService) CheckExistingResponse(
	ctx context.Context,
	request CreateRequest,
	response interface{},
) (bool, int, error) {
	key, err := s.idempotencyRepo.Get(ctx, request.UserID, request.Scope, request.Key)

	if err != nil {
		return false, 0, fmt.Errorf("get idempotency key failed: %w", err)
	}

	if key == nil {
		return false, 0, nil
	}

	if key.RequestHash != request.RequestHash {
		return false, 0, fmt.Errorf("request hash is conflict: %w", serviceerr.ErrConflict)
	}

	if key.ResponseBody == nil {
		return false, 0, fmt.Errorf("idempotency response body is empty: %w", serviceerr.ErrConflict)
	}

	if err := json.Unmarshal([]byte(*key.ResponseBody), response); err != nil {
		return false, 0, fmt.Errorf("unmarshal idempotency response failed: %w", err)
	}

	return true, key.ResponseCode, nil
}

func (s *IdempotencyService) CreateRecord(
	ctx context.Context,
	tx *gorm.DB,
	request CreateRequest,
) error {
	responseBody, err := json.Marshal(request.Response)

	if err != nil {
		return fmt.Errorf("marshal idempotency response failed: %w", err)
	}

	responseBodyString := string(responseBody)
	createIdempotencyKey := &IdempotencyKey{
		UserID:       request.UserID,
		Scope:        request.Scope,
		Key:          request.Key,
		RequestHash:  request.RequestHash,
		Status:       request.Status,
		OrderID:      request.OrderID,
		PaymentID:    request.PaymentID,
		ResponseCode: request.ResponseCode,
		ResponseBody: &responseBodyString,
	}

	if err := s.idempotencyRepo.Create(ctx, tx, createIdempotencyKey); err != nil {
		return fmt.Errorf("create idempotency key failed: %w", err)
	}

	return nil
}
