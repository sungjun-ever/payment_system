package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	idempotencydomain "order_system/internal/idempotency/domain"
	idempotencyrepository "order_system/internal/idempotency/repository"
	"order_system/internal/notification"
	orderdomain "order_system/internal/order/domain"
	"order_system/internal/payment"
	"order_system/internal/payment/domain"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/pkg/apperr/rediserr"
	"order_system/internal/pkg/apperr/serviceerr"
	"order_system/internal/pkg/pg"
	"order_system/internal/pkg/pg/toss"
	"order_system/internal/pkg/rediskey"
	"order_system/internal/pkg/retry"
	"order_system/internal/pkg/token"
	productdomain "order_system/internal/product/domain"
	"time"
)

var (
	ErrPaymentAmountMismatch    = errors.New("payment amount mismatch")
	ErrOwnershipMismatch        = errors.New("ownership mismatch")
	ErrPaymentAlreadyProcessed  = errors.New("payment already processed")
	ErrPaymentCompleted         = errors.New("payment completed")
	ErrIdempotencyKeyNotFound   = errors.New("idempotency key not found")
	ErrRequestHashMismatch      = errors.New("request hash mismatch")
	ErrDeletePaymentLockTimeout = errors.New("delete payment lock timeout")
	ErrPGRejected               = errors.New("pg rejected")
	ErrConstraintViolation      = errors.New("constraint violation")

	ErrInternalLogic = errors.New("internal logic error")
	ErrPGOutcome     = errors.New("pg outcome error")
	ErrUnknown       = errors.New("unknown error")
)

// updateStatusContext 상태 변경 전달용
type updateStatusContext struct {
	UserID            uint
	PaymentID         uint
	AttemptID         uint
	OrderID           uint
	ProviderPaymentID string
	idempotencyKey    string
	failureReason     string
	status            paymentStatusUpdate
}

type soldQuantityPayload struct {
	OrderID   uint
	ProductID uint
}

// paymentAttemptContext 결제 시도 정보 전달용
type paymentAttemptContext struct {
	UserID         uint
	PaymentID      uint
	AttemptID      uint
	OrderID        uint
	IdempotencyKey string
}

// paymentStatusUpdate 결제 진행 시 수정되어야하는 상태키 값들
type paymentStatusUpdate struct {
	PaymentStatus     domain.PaymentStatus
	IdempotencyStatus idempotencydomain.Status
	OrderStatus       orderdomain.Status
	AttemptStatus     domain.AttemptStatus
}

type PaymentService struct {
	logger               *slog.Logger
	paymentStore         payment.PaymentStore
	idempotencyRedisRepo payment.IdempotencyGuard
	slackSender          notification.Sender
	toss                 toss.TossProvider
}

func NewPaymentService(
	logger *slog.Logger,
	paymentStore payment.PaymentStore,
	idempotencyGuard payment.IdempotencyGuard,
	slackSender notification.Sender,
	toss toss.TossProvider,
) PaymentService {
	return PaymentService{
		logger,
		paymentStore,
		idempotencyGuard,
		slackSender,
		toss,
	}
}

func (ps *PaymentService) CreatePayment(
	parentCtx context.Context,
	dto domain.CreateRequest,
	claims *token.AccessClaims,
	idempotencyKey string,
	requestHash string,
) (*domain.Resource, error) {
	// 요청 타임아웃 10초 지정
	ctx, cancel := context.WithTimeoutCause(parentCtx, 10*time.Second, serviceerr.ErrTimeout)
	defer cancel()

	// 락 습득
	unlock, lockErr := ps.acquirePaymentLock(ctx, idempotencyKey)
	if lockErr != nil {
		return nil, lockErr
	}
	defer unlock()

	// 결제 요청을 처리하기 전에 요청이 처리 됐는지 확인한다.
	processedPaymentErr := ps.checkIsProcessedPayment(ctx, claims.UserID, idempotencyKey, requestHash)
	if processedPaymentErr != nil {
		return nil, processedPaymentErr
	}

	// 결제 요청 전송 전에 실제 주문과 결제가 맞는지 확인
	order, validatePaymentErr := ps.validatePaymentAndReturnOrder(ctx, dto, claims.UserID)
	if validatePaymentErr != nil {
		return nil, validatePaymentErr
	}

	// 결제 생성
	paymentCtx, savedTxErr := ps.preparePaymentAttempt(ctx, claims.UserID, idempotencyKey, order.ID, dto)
	if savedTxErr != nil {
		return nil, savedTxErr
	}

	// 결제 요청 전송
	pgResponse := ps.confirmPayment(ctx, dto, order)

	return ps.handleConfirmResult(ctx, dto, order, paymentCtx, pgResponse)
}

// acquirePaymentLock 결제 시작 전 락 획득
func (ps *PaymentService) acquirePaymentLock(
	ctx context.Context,
	idempotencyKey string,
) (func(), error) {
	lockKey := rediskey.IdempotencyLockKey(idempotencyKey)
	lockToken := rediskey.IdempotencyLockToken()

	if err := ps.getLock(ctx, lockKey, lockToken); err != nil {
		return nil, err
	}

	return func() {
		ps.deleteLock(ctx, lockKey, lockToken)
	}, nil
}

// getLock 중복 요청 막는용도 락 획득
func (ps *PaymentService) getLock(ctx context.Context, lockKey string, lockToken string) error {
	// 중복 요청을 막기 위해 락 획득
	err := ps.idempotencyRedisRepo.GetLock(ctx, lockKey, lockToken)

	if err != nil {
		if errors.Is(err, rediserr.ErrLockExists) {
			return fmt.Errorf("create payment: %w: %w", err, ErrPaymentAlreadyProcessed)
		}

		return fmt.Errorf("create payment: %w", err)
	}

	return nil
}

// deleteLock 요청 종료 후 락 해제
func (ps *PaymentService) deleteLock(ctx context.Context, lockKey string, lockToken string) {
	defer func() {
		cleanUpCtx, cleanUpCancel := context.WithTimeoutCause(
			context.WithoutCancel(ctx),
			2*time.Second,
			ErrDeletePaymentLockTimeout,
		)
		defer cleanUpCancel()

		deleteLockErr := ps.idempotencyRedisRepo.DeleteLock(cleanUpCtx, lockKey, lockToken)

		if deleteLockErr != nil {
			switch {
			case errors.Is(deleteLockErr, rediserr.ErrLockNotOwned):
				ps.logger.ErrorContext(
					cleanUpCtx,
					"idempotency lock not owned",
					"lockKey", lockKey,
					"err", deleteLockErr,
				)
			case errors.Is(cleanUpCtx.Err(), context.DeadlineExceeded):
				ps.logger.ErrorContext(
					cleanUpCtx,
					"delete idempotency lock timeout",
					"lockKey", lockKey,
					"err", context.Cause(cleanUpCtx),
				)
			default:
				ps.logger.ErrorContext(
					cleanUpCtx,
					"delete idempotency lock failed",
					"lockKey", lockKey,
					"err", deleteLockErr,
				)
			}
		}
	}()
}

// checkIsProcessedPayment 이미 처리된 결제 요청인지 확인한다.
func (ps *PaymentService) checkIsProcessedPayment(
	ctx context.Context,
	userID uint,
	idempotencyKey string,
	requestHash string,
) error {
	// DB 확인 전에 레디스 확인
	status, err := ps.idempotencyRedisRepo.GetIdempotencyStatus(ctx, idempotencyKey)

	if err != nil {
		return fmt.Errorf("get idempotency status error: %w", err)
	}

	// 상태가 존재한다면 이미 처리된 결제 요청
	if status != "" {
		return fmt.Errorf("payment already processed: %w", ErrPaymentCompleted)
	}

	savedIdempotency, err := ps.paymentStore.ValidateIdempotency(
		ctx,
		userID,
		idempotencyKey,
		requestHash,
	)

	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			return fmt.Errorf("idempotency key not found: %w : %w", err, ErrIdempotencyKeyNotFound)
		}

		if errors.Is(err, idempotencyrepository.ErrIdempotencyHashMismatch) {
			return fmt.Errorf("request hash mismatch: %w : %w", err, ErrRequestHashMismatch)
		}

		return fmt.Errorf("get idempotency key error %w", err)
	}

	// 처리 중 상태가 아닌 경우 이미 처리 완료 리턴
	if savedIdempotency.Status != idempotencydomain.StatusProcessing {
		return fmt.Errorf("payment already processed: %w", ErrPaymentCompleted)
	}

	return nil
}

// validatePaymentAndReturnOrder 실제 주문과 결제가 일치하는지 확인
func (ps *PaymentService) validatePaymentAndReturnOrder(
	ctx context.Context,
	dto domain.CreateRequest,
	userID uint,
) (*orderdomain.Order, error) {
	// 사용자 결제 요청 금액이 주문과 맞는지 확인
	order, err := ps.paymentStore.FindOrderForPayment(ctx, dto.OrderID)

	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			return nil, fmt.Errorf("order not found: %w", err)
		}
		return nil, fmt.Errorf("find order error: %w", err)
	}

	// 주문 금액과 결제 요청 금액이 맞지 않는 경우
	if order.TotalAmount != dto.Amount {
		return nil, fmt.Errorf("order total amount is not equal to payment amount: %w", ErrPaymentAmountMismatch)
	}

	// 주문과 결제자의 소유권이 다르면 에러
	if order.UserID != userID {
		return nil, fmt.Errorf("order owner and payment owner is not equal: %w", ErrOwnershipMismatch)
	}

	return order, nil
}

// preparePaymentAttempt 결제 요청 전 관련 데이터 생성
func (ps *PaymentService) preparePaymentAttempt(
	ctx context.Context,
	userID uint,
	idempotencyKey string,
	orderID uint,
	dto domain.CreateRequest,
) (paymentAttemptContext, error) {
	paymentID, attemptID, err := ps.createPaymentAndMapIdempotencyTx(ctx, userID, idempotencyKey, dto)
	if err != nil {
		return paymentAttemptContext{}, err
	}

	return paymentAttemptContext{
		UserID:         userID,
		PaymentID:      paymentID,
		AttemptID:      attemptID,
		OrderID:        orderID,
		IdempotencyKey: idempotencyKey,
	}, nil
}

// createPaymentAndMapIdempotencyTx 결제 생성 및 멱등키와 결제 id 연결
func (ps *PaymentService) createPaymentAndMapIdempotencyTx(
	ctx context.Context,
	userID uint,
	idempotencyKey string,
	dto domain.CreateRequest,
) (uint, uint, error) {
	paymentID := uint(0)
	attemptID := uint(0)
	err := ps.paymentStore.Tx(ctx, func(tx payment.PayTx) error {
		// 생성된 payment가 있는지 확인 후 없으면 payment 생성
		exist, err := tx.PaymentsReader().FindByUserAndOrderID(ctx, userID, dto.OrderID)

		if err != nil {
			return err
		}

		// 기존 payment가 존재하지 않는 경우에만 생성
		if exist != nil {
			paymentID = exist.ID
		} else {
			createPaymentEntity := dto.ToCreatePaymentEntity(userID)

			saved, createErr := tx.PaymentsWriter().Create(ctx, createPaymentEntity)
			if createErr != nil {
				return createErr
			}

			paymentID = saved.ID
		}

		// payment attempt 생성
		createAttemptEntity := dto.ToCreateAttemptEntity(paymentID, idempotencyKey)

		attempt, createErr := tx.AttemptsWriter().Create(ctx, createAttemptEntity)
		if createErr != nil {
			return createErr
		}
		attemptID = attempt.ID

		// 멱등성 상태 업데이트
		updateIdempotencyErr := tx.IdempotenciesWriter().Update(
			ctx,
			userID,
			idempotencyKey,
			idempotencydomain.ScopePayOrder,
			map[string]interface{}{
				"status":     idempotencydomain.StatusProcessing,
				"payment_id": paymentID,
			})

		if updateIdempotencyErr != nil {
			if errors.Is(updateIdempotencyErr, dberr.ErrNotFound) {
				return fmt.Errorf(
					"update idempotency failed: %w: %w",
					updateIdempotencyErr,
					serviceerr.ErrResourceNotFound,
				)
			}
			return updateIdempotencyErr
		}

		return nil

	})

	if err != nil {
		return 0, 0, err
	}

	return paymentID, attemptID, nil
}

// confirmPayment 결제 요청 전송
func (ps *PaymentService) confirmPayment(
	ctx context.Context,
	dto domain.CreateRequest,
	order *orderdomain.Order,
) toss.ResponseDto {
	tossDto := toss.ToConfirmationDTO(dto.PaymentNo, order.OrderNo, dto.Amount)
	return ps.toss.Confirm(ctx, tossDto)
}

// handleConfirmResult 결제 요청 결과 별 행동
func (ps *PaymentService) handleConfirmResult(
	ctx context.Context,
	dto domain.CreateRequest,
	order *orderdomain.Order,
	paymentCtx paymentAttemptContext,
	pgResponse toss.ResponseDto,
) (*domain.Resource, error) {
	statusContext := ps.newUpdateStatusContext(paymentCtx, pgResponse)

	// 결제 성공과 결제 거절은 요청 성공
	// 발송 요청 오류, PG 오류, 미식별 오류는 요청 실패
	switch pgResponse.Response {
	case pg.Succeeded:
		ps.applyConfirmStatus(ctx, statusContext, dto)
		return domain.NewResource(true, pgResponse.Reason, false), nil
	case pg.Rejected:
		ps.applyConfirmStatus(ctx, statusContext, dto)
		return domain.NewResource(false, pgResponse.Reason, false), nil
	case pg.Completed:
		inquiryResponse := ps.toss.Inquiry(ctx, order.OrderNo, dto.PaymentNo)
		ps.compareInquiryResult(ctx, inquiryResponse, paymentCtx, dto)
		return nil, fmt.Errorf("payment already processed: %w", ErrPaymentCompleted)
	case pg.ServerFailed:
		ps.applyConfirmStatus(ctx, statusContext, dto)
		return domain.NewResource(false, pgResponse.Reason, false),
			fmt.Errorf("data: %v, failed reason: %s, server request to pg error: %w ",
				dto, pgResponse.Reason, ErrInternalLogic)
	case pg.PGFailed:
		ps.applyConfirmStatus(ctx, statusContext, dto)
		return domain.NewResource(false, pgResponse.Reason, false),
			fmt.Errorf("data: %v, failed reason: %s, pg response error: %w",
				dto, pgResponse.Reason, ErrPGOutcome)
	default:
		return ps.handleUnknownConfirmResult(ctx, dto, pgResponse)
	}
}

// newUpdateStatusContext 상태 업데이트 context 생성
func (ps *PaymentService) newUpdateStatusContext(
	paymentCtx paymentAttemptContext,
	pgResponse toss.ResponseDto,
) updateStatusContext {
	return updateStatusContext{
		UserID:            paymentCtx.UserID,
		PaymentID:         paymentCtx.PaymentID,
		AttemptID:         paymentCtx.AttemptID,
		OrderID:           paymentCtx.OrderID,
		ProviderPaymentID: pgResponse.PaymentID,
		idempotencyKey:    paymentCtx.IdempotencyKey,
		failureReason:     pgResponse.Reason,
		status:            ps.buildStatusUpdate(pgResponse.Response),
	}
}

// applyConfirmStatus 상태 업데이트 적용 및 실패 시 fallback
func (ps *PaymentService) applyConfirmStatus(
	ctx context.Context,
	statusContext updateStatusContext,
	dto domain.CreateRequest,
) {
	txErr := ps.updateStatusTx(ctx, statusContext)
	if txErr != nil {
		if err := ps.updateStatusTxFailedFallback(ctx, statusContext, dto, txErr); err != nil {
			return
		}
	}

	if statusContext.status.PaymentStatus == domain.Succeeded {
		ps.applySoldQuantity(ctx, statusContext)
	}

}

// handleUnknownConfirmResult 미식별 결과
func (ps *PaymentService) handleUnknownConfirmResult(
	ctx context.Context,
	dto domain.CreateRequest,
	pgResponse toss.ResponseDto,
) (*domain.Resource, error) {
	// TODO 결제 재시도 구현하기
	ps.logger.ErrorContext(ctx, "payment unknown failed", "data", dto, "reason", pgResponse.Reason)
	ps.sendNotification(ctx, fmt.Sprintf(
		"unknown payment failed: %v, reason: %s", dto, pgResponse.Reason))

	return nil, fmt.Errorf("data: %v, failed reason: %s, unknown error: %w", dto, pgResponse.Reason, ErrUnknown)
}

// compareInquiryResult 결제 조회 결과와 현재 상태 비교
func (ps *PaymentService) compareInquiryResult(
	ctx context.Context,
	pgResponse toss.ResponseDto,
	paymentCtx paymentAttemptContext,
	dto domain.CreateRequest,
) {
	// 업데이트할 context 생성
	updateCtx := ps.newUpdateStatusContext(paymentCtx, pgResponse)
	isPaymentStatusSame := false
	isAttemptStatusSame := false
	isOrderStatusSame := false
	isIdempotencyStatusSame := false

	// 결제, 결제 시도, 주문, 멱등성을 조회한다.
	err := ps.paymentStore.Tx(ctx, func(tx payment.PayTx) error {
		getPayment, paymentErr := tx.PaymentsReader().Find(ctx, paymentCtx.PaymentID)

		if paymentErr != nil {
			if errors.Is(paymentErr, dberr.ErrNotFound) {
				ps.logger.ErrorContext(ctx, "after payment inquiry, payment not found",
					"payment", paymentCtx,
					"error", paymentErr)
				ps.sendNotification(ctx,
					fmt.Sprintf("after payment inquiry, payment not found: %v, error: %s",
						paymentCtx, paymentErr.Error()),
				)

				return fmt.Errorf(
					"after payment inquiry, payment not found: %w: %w",
					paymentErr,
					ErrConstraintViolation,
				)
			}

			return fmt.Errorf("find payment error in inquiry: %w", paymentErr)
		}

		isPaymentStatusSame = getPayment.Status == updateCtx.status.PaymentStatus
		getAttempt, attemptErr := tx.AttemptsReader().Find(ctx, paymentCtx.AttemptID)

		if attemptErr != nil {
			if errors.Is(attemptErr, dberr.ErrNotFound) {
				ps.logger.ErrorContext(ctx, "after payment inquiry, attempt not found",
					"attempt", paymentCtx,
					"error", attemptErr)
				ps.sendNotification(ctx, fmt.Sprintf(
					"after payment inquiry, attempt not found: %v, error: %s",
					paymentCtx, attemptErr.Error()),
				)

				return fmt.Errorf(
					"after payment inquiry, attempt not found: %w: %w",
					paymentErr,
					ErrConstraintViolation,
				)

			}
			return fmt.Errorf("find attempt error  in inquiry: %w", paymentErr)
		}

		isAttemptStatusSame = getAttempt.Status == updateCtx.status.AttemptStatus

		getOrder, orderErr := tx.OrdersReader().Find(ctx, paymentCtx.OrderID)

		if orderErr != nil {
			if errors.Is(orderErr, dberr.ErrNotFound) {
				ps.logger.ErrorContext(ctx, "after payment inquiry, order not found",
					"order", paymentCtx,
					"error", orderErr)
				ps.sendNotification(ctx, fmt.Sprintf(
					"after payment inquiry, order not found: %v, error: %s",
					paymentCtx, orderErr.Error()),
				)
				return fmt.Errorf(
					"after payment inquiry, order not found: %w: %w",
					orderErr,
					ErrConstraintViolation,
				)
			}
			return fmt.Errorf("find order error in inquiry: %w", orderErr)
		}

		isOrderStatusSame = getOrder.Status == updateCtx.status.OrderStatus

		getIdempotency, idempotencyErr := tx.IdempotenciesReader().FindByConstraint(
			ctx,
			paymentCtx.UserID,
			idempotencydomain.ScopePayOrder,
			paymentCtx.IdempotencyKey,
		)

		if idempotencyErr != nil {
			if errors.Is(idempotencyErr, dberr.ErrNotFound) {
				ps.logger.ErrorContext(ctx, "after payment inquiry, idempotency not found",
					"order", paymentCtx,
					"error", orderErr)
				ps.sendNotification(ctx, fmt.Sprintf(
					"after payment inquiry, idempotency not found: %v, error: %s",
					paymentCtx, idempotencyErr.Error()),
				)
				return fmt.Errorf(
					"after payment inquiry, idempotencyErr: %w: %w",
					orderErr,
					ErrConstraintViolation,
				)
			}
			return fmt.Errorf("find idempotency error in inquiry: %w", idempotencyErr)
		}

		isIdempotencyStatusSame = getIdempotency.Status == updateCtx.status.IdempotencyStatus

		return nil
	})

	if err == nil {
		if !(isPaymentStatusSame && isAttemptStatusSame && isOrderStatusSame && isIdempotencyStatusSame) {
			ps.applyConfirmStatus(ctx, updateCtx, dto)
		}
	}
}

// buildStatusUpdate pg response 별 상태 정의
func (ps *PaymentService) buildStatusUpdate(response pg.PGResponse) paymentStatusUpdate {
	switch response {
	case pg.Succeeded:
		return paymentStatusUpdate{
			PaymentStatus:     domain.Succeeded,
			IdempotencyStatus: idempotencydomain.StatusSucceeded,
			OrderStatus:       orderdomain.StatusCompleted,
			AttemptStatus:     domain.AttemptStatusSucceeded,
		}
	case pg.Rejected:
		return paymentStatusUpdate{
			PaymentStatus:     domain.Rejected,
			IdempotencyStatus: idempotencydomain.StatusFailed,
			OrderStatus:       orderdomain.StatusFailed,
			AttemptStatus:     domain.AttemptStatusRejected,
		}
	default:
		return paymentStatusUpdate{
			PaymentStatus:     domain.Failed,
			IdempotencyStatus: idempotencydomain.StatusFailed,
			OrderStatus:       orderdomain.StatusFailed,
			AttemptStatus:     domain.AttemptStatusFailed,
		}
	}
}

// updateStatusTx 상태 변경 트랜잭션
func (ps *PaymentService) updateStatusTx(
	ctx context.Context,
	statusContext updateStatusContext,
) error {
	err := ps.paymentStore.Tx(ctx, func(tx payment.PayTx) error {
		paymentStatusField := map[string]interface{}{
			"status": statusContext.status.PaymentStatus,
		}

		if statusContext.status.PaymentStatus == domain.Succeeded {
			paymentStatusField["paid_at"] = time.Now()
		}

		// payment 업데이트
		paymentStatusErr := tx.PaymentsWriter().Update(ctx, statusContext.PaymentID, paymentStatusField)

		if paymentStatusErr != nil {
			if errors.Is(paymentStatusErr, dberr.ErrNotFound) {
				return fmt.Errorf(
					"update payment failed: %w: %w",
					paymentStatusErr,
					serviceerr.ErrResourceNotFound,
				)
			}
			return paymentStatusErr
		}

		attemptStatusField := map[string]interface{}{
			"status":         statusContext.status.AttemptStatus,
			"failure_reason": statusContext.failureReason,
		}
		if statusContext.ProviderPaymentID != "" {
			attemptStatusField["provider_payment_id"] = statusContext.ProviderPaymentID
		}

		// attempt 업데이트
		attemptStatusErr := tx.AttemptsWriter().Update(ctx, statusContext.AttemptID, attemptStatusField)
		if attemptStatusErr != nil {
			if errors.Is(attemptStatusErr, dberr.ErrNotFound) {
				return fmt.Errorf(
					"update attempt failed: %w: %w",
					attemptStatusErr,
					serviceerr.ErrResourceNotFound,
				)
			}
			return attemptStatusErr
		}

		// idempotency 업데이트
		updateIdempotencyErr := tx.IdempotenciesWriter().Update(
			ctx,
			statusContext.UserID,
			statusContext.idempotencyKey,
			idempotencydomain.ScopePayOrder,
			map[string]interface{}{
				"status": statusContext.status.IdempotencyStatus,
			},
		)

		if updateIdempotencyErr != nil {
			if errors.Is(updateIdempotencyErr, dberr.ErrNotFound) {
				return fmt.Errorf(
					"update idempotency failed: %w: %w",
					updateIdempotencyErr,
					serviceerr.ErrResourceNotFound,
				)
			}
			return updateIdempotencyErr
		}

		// order 업데이트
		updateOrderErr := tx.OrdersWriter().Update(ctx, statusContext.OrderID, map[string]interface{}{
			"status": statusContext.status.OrderStatus,
		})

		if updateOrderErr != nil {
			if errors.Is(updateOrderErr, dberr.ErrNotFound) {
				return fmt.Errorf(
					"update order failed: %w: %w",
					updateOrderErr,
					serviceerr.ErrResourceNotFound,
				)
			}
			return updateOrderErr
		}
		return nil
	})

	return err
}

// updateStatusTxFailedFallback 상태 업데이트 처리 실패 시
func (ps *PaymentService) updateStatusTxFailedFallback(
	ctx context.Context,
	statusContext updateStatusContext,
	dto domain.CreateRequest,
	txErr error,
) error {
	// 상태 업데이트 실패 시 재시도
	retryErr := retry.Retry(ctx, ps.logger, retry.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}, func() error {
		return ps.updateStatusTx(ctx, statusContext)
	})

	// retry 성공시 종료
	if retryErr == nil {
		return nil
	}

	// retry 실패시 기록
	ps.logger.ErrorContext(ctx, "update payment status retry failed",
		"tx err", txErr,
		"retry err", retryErr)
	ps.sendNotification(ctx, fmt.Sprintf(
		"update payment status retry failed: %s, info: %v, context: %v",
		retryErr.Error(), dto, statusContext),
	)

	// retry 실패시 idempotency status 등록
	setErr := ps.idempotencyRedisRepo.SetIdempotencyStatus(
		ctx,
		statusContext.idempotencyKey,
		statusContext.status.IdempotencyStatus,
	)

	if setErr != nil {
		ps.logger.ErrorContext(ctx, "update payment status, set idempotency status failed",
			"tx err", txErr,
			"idempotency err", setErr)
		ps.sendNotification(ctx,
			fmt.Sprintf("update payment status failed: %s, set idempotency status failed: %s, info: %v",
				txErr.Error(), setErr.Error(), dto),
		)

		return setErr
	}

	return nil
}

// applySoldQuantity 판매 재고 반영
func (ps *PaymentService) applySoldQuantity(ctx context.Context, statusContext updateStatusContext) {
	items, err := ps.getOrderItems(ctx, statusContext.OrderID)

	if err != nil {
		return
	}

	ps.confirmSoldQuantity(ctx, items)
}

// getOrderItems 주문 아이템 조회
func (ps *PaymentService) getOrderItems(ctx context.Context, orderID uint) ([]*orderdomain.OrderItem, error) {
	items, err := ps.paymentStore.GetItemsByOrderID(ctx, orderID)

	if err != nil {
		retryErr := retry.Retry(
			ctx,
			ps.logger,
			retry.RetryPolicy{
				MaxAttempts: 3,
				BaseDelay:   100 * time.Millisecond,
				MaxDelay:    1 * time.Second,
			},
			func() error {
				items, err = ps.paymentStore.GetItemsByOrderID(ctx, orderID)
				return err
			},
		)

		if retryErr != nil {
			ps.logger.ErrorContext(ctx, "apply sold quantity, get item retry failed",
				"error", retryErr.Error())
			ps.sendNotification(ctx,
				fmt.Sprintf("apply sold quantity, get item retry failed, orderID: %d, error: %s",
					orderID, retryErr.Error()),
			)
			return nil, retryErr
		}
	}

	return items, nil
}

// confirmSoldQuantity 판매 재고 반영
func (ps *PaymentService) confirmSoldQuantity(
	ctx context.Context,
	items []*orderdomain.OrderItem,
) {
	var err error
	for _, item := range items {
		// 판매 재고 반영 트랜잭션
		err = ps.paymentStore.Tx(ctx, func(tx payment.PayTx) error {
			inventoryMovementCreateContext := &productdomain.InventoryMovement{
				OrderID:   item.OrderID,
				ProductID: item.ProductID,
				Operation: productdomain.ConfirmSale,
				Quantity:  item.Quantity,
			}
			err = tx.InventoryMovementWriter().CreateInventoryMovement(ctx, inventoryMovementCreateContext)

			if err != nil {
				// 중복이라면 이미 처리된 상태
				if errors.Is(err, dberr.ErrDuplicate) {
					return nil
				}
				return fmt.Errorf("create inventory movement failed, err: %w", err)
			}

			err = tx.InventoryWriter().UpdateSoldQuantity(ctx, item.ProductID, item.Quantity)

			if err != nil {
				return fmt.Errorf("update sold quantity failed, err: %w", err)
			}

			return nil
		})

		if err != nil {
			if fallbackErr := ps.updateSoldQuantityFallback(ctx, item, err); fallbackErr != nil {
				ps.logger.ErrorContext(ctx, "update sold quantity failed", "err", fallbackErr, "info", item)
				ps.sendNotification(ctx, fmt.Sprintf(
					"update sold quantity failed, err: %s, order id: %d, product id: %d",
					fallbackErr.Error(), item.OrderID, item.ProductID))
			}
		}
	}
}

// updateSoldQuantityFallback 판매 재고 반영 실패 후처리
func (ps *PaymentService) updateSoldQuantityFallback(
	ctx context.Context,
	item *orderdomain.OrderItem,
	err error,
) error {
	// db 에러 분류
	classify := dberr.ClassifyDBError(err)
	if classify == dberr.DBErrorRetryable || classify == dberr.DBErrorAmbiguous {
		ps.logger.WarnContext(ctx, "apply sold quantity failed, update sold quantity failed",
			"err", err,
			"info", item,
		)
		payload, _ := json.Marshal(soldQuantityPayload{
			OrderID:   item.OrderID,
			ProductID: item.ProductID,
		})
		uniqueKey := fmt.Sprintf("%s-%s-%d-%d",
			productdomain.TargetDB, productdomain.ConfirmSale, item.OrderID, item.ProductID)
		err = ps.paymentStore.CreateJob(ctx, productdomain.InventoryJobCreateContext{
			Target:      productdomain.TargetDB,
			Operation:   productdomain.ConfirmSale,
			RetryCount:  1,
			Status:      productdomain.JobPending,
			Payload:     string(payload),
			UniqueKey:   uniqueKey,
			CreatedAt:   time.Now(),
			NextRetryAt: time.Now(),
		})

		if err != nil {
			// 유니크 중복은 이미 생성된 job
			if errors.Is(err, dberr.ErrDuplicate) {
				return nil
			}

			return fmt.Errorf("create job failed, err: %w", err)
		}
	} else {
		return fmt.Errorf("update sold quantity failed, not retryable error, err: %w", err)
	}

	return nil
}

func (ps *PaymentService) sendNotification(ctx context.Context, message string) {
	_ = ps.slackSender.Send(ctx, notification.Message{
		Channel: notification.ChannelSlack,
		To:      "slack bot",
		Title:   "",
		Body:    message,
	})
}
