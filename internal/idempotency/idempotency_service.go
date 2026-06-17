package idempotency

import (
	"context"
	"fmt"
	"payment_system/internal/pkg/apperr/serviceerr"
	"payment_system/internal/pkg/token"

	"github.com/google/uuid"
)

type IdempotencyService struct {
	idempotencyRepo IdempotencyKeyRepository
}

func NewIdempotencyService(idempotencyRepo IdempotencyKeyRepository) IdempotencyService {
	return IdempotencyService{idempotencyRepo: idempotencyRepo}
}

// CheckExistingResponse 멱등키와 요청해시가 일치하는지 확인
func (s *IdempotencyService) CheckExistingResponse(
	ctx context.Context,
	request UpdateRequest,
	response interface{},
) (bool, int, error) {
	// 멱등키와 내용을 가져온다.
	//key, err := s.idempotencyRepo.Get(ctx, request.UserID, request.Scope, request.Key)
	//
	//if err != nil {
	//	return false, 0, fmt.Errorf("get idempotency key failed: %w", err)
	//}
	//
	//if key == nil {
	//	return false, 0, nil
	//}
	//
	//if key.RequestHash != request.RequestHash {
	//	return false, 0, fmt.Errorf("request hash is conflict: %w", serviceerr.ErrConflict)
	//}
	//
	//if key.ResponseBody == nil {
	//	return false, 0, fmt.Errorf("idempotency response body is empty: %w", serviceerr.ErrConflict)
	//}
	//
	//if err = json.Unmarshal([]byte(*key.ResponseBody), response); err != nil {
	//	return false, 0, fmt.Errorf("unmarshal idempotency response failed: %w", err)
	//}
	//
	//return true, key.ResponseCode, nil
	return false, 0, nil
}

func (s *IdempotencyService) CreateKey(
	ctx context.Context,
	request CreateRequest,
	claims *token.AccessClaims,
) (*Resource, error) {
	// scope, status 멥핑
	scope, status := s.mapScopeAndStatus(request.Origin, request.Action)

	if scope == nil || status == nil {
		return nil, fmt.Errorf("invalid idempotency origin and action: %w", serviceerr.ErrInvalidArgument)
	}

	idempotencyKey := &IdempotencyKey{
		UserID: claims.UserID,
		Scope:  *scope,
		Key:    s.generateKey(),
		Status: *status,
	}

	err := s.idempotencyRepo.Create(ctx, idempotencyKey)

	if err != nil {
		return nil, fmt.Errorf("create idempotency key failed: %w", err)
	}

	return NewResource(idempotencyKey.Key), nil
	//responseBody, err := json.Marshal(request.Response)
	//
	//if err != nil {
	//	return fmt.Errorf("marshal idempotency response failed: %w", err)
	//}
	//
	//responseBodyString := string(responseBody)
	//createIdempotencyKey := &IdempotencyKey{
	//	UserID:       request.UserID,
	//	Scope:        request.Scope,
	//	Key:          request.Key,
	//	RequestHash:  request.RequestHash,
	//	Status:       request.Status,
	//	OrderID:      request.OrderID,
	//	PaymentID:    request.PaymentID,
	//	ResponseCode: request.ResponseCode,
	//	ResponseBody: &responseBodyString,
	//}
	//
	//if err := s.idempotencyRepo.CreateRows(ctx, tx, createIdempotencyKey); err != nil {
	//	return fmt.Errorf("create idempotency key failed: %w", err)
	//}
}

func (s *IdempotencyService) generateKey() string {
	key := uuid.New().String()
	return "idempotency_" + key
}

func (s *IdempotencyService) mapScopeAndStatus(origin string, action string) (*Scope, *Status) {
	if origin == "order" {
		if action == "create" {
			scope := ScopeOrderCreated
			status := StatusProcessing
			return &scope, &status
		}
	} else if origin == "payment" {

	}

	return nil, nil
}
