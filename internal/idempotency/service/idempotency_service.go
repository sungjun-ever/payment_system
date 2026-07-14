package service

import (
	"context"
	"fmt"
	"order_system/internal/idempotency/domain"
	"order_system/internal/idempotency/repository"
	"order_system/internal/pkg/apperr/serviceerr"
	"order_system/internal/pkg/token"

	"github.com/google/uuid"
)

type IdempotencyService struct {
	idempotencyRepo repository.IdempotencyGormRepository
}

func NewIdempotencyService(idempotencyRepo repository.IdempotencyGormRepository) IdempotencyService {
	return IdempotencyService{idempotencyRepo: idempotencyRepo}
}

func (s *IdempotencyService) CreateKey(
	ctx context.Context,
	request domain.CreateRequest,
	claims *token.AccessClaims,
) (*domain.Resource, error) {
	// scope, status 멥핑
	scope, status := s.mapScopeAndStatus(request.Origin, request.Action)

	if scope == nil || status == nil {
		return nil, fmt.Errorf("invalid idempotency origin and action: %w", serviceerr.ErrInvalidArgument)
	}

	idempotencyKey := &domain.IdempotencyKey{
		UserID: claims.UserID,
		Scope:  *scope,
		Key:    s.generateKey(),
		Status: *status,
	}

	err := s.idempotencyRepo.Create(ctx, idempotencyKey)

	if err != nil {
		return nil, fmt.Errorf("create idempotency key failed: %w", err)
	}

	return domain.NewResource(idempotencyKey.Key), nil
}

func (s *IdempotencyService) generateKey() string {
	key := uuid.New().String()
	return "idempotency_" + key
}

func (s *IdempotencyService) mapScopeAndStatus(origin string, action string) (*domain.Scope, *domain.Status) {
	if origin == "order" {
		if action == "create" {
			scope := domain.ScopeOrderCreated
			status := domain.StatusProcessing
			return &scope, &status
		} else if action == "cancel" {
			scope := domain.ScopeOrderCancelled
			status := domain.StatusProcessing
			return &scope, &status
		}
	} else if origin == "payment" {
		if action == "create" {
			scope := domain.ScopePayOrder
			status := domain.StatusProcessing
			return &scope, &status
		} else if action == "refund" {
			scope := domain.ScopeRefundOrder
			status := domain.StatusProcessing
			return &scope, &status
		}
	}

	return nil, nil
}
