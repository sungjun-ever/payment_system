package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	idempotencydomain "order_system/internal/idempotency/domain"
	idempotencyrepository "order_system/internal/idempotency/repository"
	"order_system/internal/notification"
	orderdomain "order_system/internal/order/domain"
	orderrepository "order_system/internal/order/repository"
	"order_system/internal/payment"
	"order_system/internal/payment/domain"
	"order_system/internal/payment/repository"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/pkg/apperr/rediserr"
	"order_system/internal/pkg/apperr/serviceerr"
	"order_system/internal/pkg/pg"
	"order_system/internal/pkg/pg/toss"
	"order_system/internal/pkg/rediskey"
	"order_system/internal/pkg/token"
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

	ErrInternalLogic = errors.New("internal logic error")
	ErrPGOutcome     = errors.New("pg outcome error")
	ErrUnknown       = errors.New("unknown error")
)

type UpdateStatusVo struct {
	UserID         uint
	PaymentID      uint
	AttemptID      uint
	OrderID        uint
	idempotencyKey string
	failureReason  string
	statusMap      map[string]interface{}
}

type PaymentService struct {
	logger               *slog.Logger
	paymentUow           payment.PaymentUnitOfWork
	paymentGormRepo      repository.PaymentGormRepository
	idempotencyGormRepo  idempotencyrepository.IdempotencyGormRepository
	idempotencyRedisRepo idempotencyrepository.IdempotencyRedisRepository
	orderGormRepo        orderrepository.OrderGormRepository
	slackSender          notification.Sender
	toss                 toss.TossProvider
}

func NewPaymentService(
	logger *slog.Logger,
	paymentUow payment.PaymentUnitOfWork,
	paymentGormRepo repository.PaymentGormRepository,
	idempotencyGormRepo idempotencyrepository.IdempotencyGormRepository,
	idempotencyRedisRepo idempotencyrepository.IdempotencyRedisRepository,
	orderGormRepo orderrepository.OrderGormRepository,
	slackSender notification.Sender,
	toss toss.TossProvider,
) PaymentService {
	return PaymentService{
		logger,
		paymentUow,
		paymentGormRepo,
		idempotencyGormRepo,
		idempotencyRedisRepo,
		orderGormRepo,
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

	// 락 획득
	lockKey := rediskey.IdempotencyLockKey(idempotencyKey)
	lockToken := rediskey.IdempotencyLockToken()

	getLockErr := ps.getLock(ctx, lockKey, lockToken)
	if getLockErr != nil {
		return nil, getLockErr
	}

	// 락을 획득했다면 요청이 끝나면 락을 풀어줌
	defer ps.deleteLock(ctx, lockKey, lockToken)

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
	paymentID, attemptID, savedTxErr := ps.createPaymentAndMapIdempotencyTx(ctx, claims.UserID, idempotencyKey, dto)
	if savedTxErr != nil {
		return nil, savedTxErr
	}

	// 결제 요청 전송
	tossDto := toss.ToConfirmationDTO(dto.PaymentNo, order.OrderNo, dto.Amount)
	confirmResult := ps.toss.Confirm(ctx, tossDto)
	// 결제 성공과 결제 거절은 요청 성공
	// 발송 요청 오류, PG 오류, 미식별 오류는 요청 실패

	statusVo := UpdateStatusVo{
		UserID:         claims.UserID,
		PaymentID:      paymentID,
		AttemptID:      attemptID,
		OrderID:        order.ID,
		idempotencyKey: idempotencyKey,
		failureReason:  confirmResult.Reason,
		statusMap:      ps.mappingUpdateStatus(confirmResult.Response),
	}

	if confirmResult.Response == pg.Succeeded || confirmResult.Response == pg.Rejected {
		succeeded := true

		if confirmResult.Response == pg.Rejected {
			succeeded = false
		}

		// 상태 업데이트
		txErr := ps.updateStatusTx(ctx, statusVo)
		if txErr != nil {
			idempotencyStatus := idempotencydomain.StatusSucceeded
			if confirmResult.Response == pg.Rejected {
				idempotencyStatus = idempotencydomain.StatusFailed
			}
			ps.updateStatusTxFailedFallback(ctx, idempotencyKey, idempotencyStatus, dto, txErr)
		}

		return domain.NewResource(succeeded, confirmResult.Reason, false), nil
	}

	// 이미 처리된 결제는 Conflict 에러
	// TODO 이미 처리된 결제인 경우, 결제 조회 후 DB에 결제 결과가 반영되어있는지 확인하는 로직이 있으면 좋을 것 같음
	if confirmResult.Response == pg.Completed {
		ps.toss.Inquiry(ctx, order.OrderNo, dto.PaymentNo)
		return nil, fmt.Errorf("payment already processed: %w", ErrPaymentCompleted)
	}

	// 결제 요청 인풋 오류는 내부 로직 문제로 -> 알림 전송, 서버 오류 리턴
	if confirmResult.Response == pg.ServerFailed {
		txErr := ps.updateStatusTx(ctx, statusVo)
		if txErr != nil {
			ps.updateStatusTxFailedFallback(ctx, idempotencyKey, idempotencydomain.StatusFailed, dto, txErr)
		}

		return domain.NewResource(false, confirmResult.Reason, false),
			fmt.Errorf("data: %v, failed reason: %s, server request to pg error: %w ",
				dto, confirmResult.Reason, ErrInternalLogic)
	}

	// PG 사 오류의 경우
	if confirmResult.Response == pg.PGFailed {
		txErr := ps.updateStatusTx(ctx, statusVo)
		if txErr != nil {
			ps.updateStatusTxFailedFallback(ctx, idempotencyKey, idempotencydomain.StatusFailed, dto, txErr)
		}

		return domain.NewResource(false, confirmResult.Reason, false),
			fmt.Errorf("data: %v, failed reason: %s, pg response error: %w",
				dto, confirmResult.Reason, ErrPGOutcome)
	}

	// 미식별 오류는 -> 알림 전송, 서버 오류 리턴
	// TODO 결제 재시도 구현하기
	ps.logger.ErrorContext(ctx, "payment unknown failed", "data", dto, "reason", confirmResult.Reason)
	ps.slackSender.Send(ctx, notification.Message{
		Channel: notification.ChannelSlack,
		To:      "slack bot",
		Title:   "",
		Body: fmt.Sprintf(
			"unknown payment failed: %v, reason: %s", dto, confirmResult.Reason),
	})

	return nil, fmt.Errorf("data: %v, failed reason: %s, unknown error: %w", dto, confirmResult.Reason, ErrUnknown)
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

	savedIdempotency, err := ps.idempotencyGormRepo.Validate(
		ctx,
		userID,
		idempotencydomain.ScopePayOrder,
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
	var order *orderdomain.Order
	err := ps.paymentUow.Tx(ctx, func(tx payment.PayTx) error {
		getOrder, getOrderErr := tx.OrderReader().Find(ctx, dto.OrderID)

		if getOrderErr != nil {
			return getOrderErr
		}

		order = getOrder

		return nil
	})

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

// createPaymentAndMapIdempotencyTx 결제 생성 및 멱등키와 결제 id 연결
func (ps *PaymentService) createPaymentAndMapIdempotencyTx(
	ctx context.Context,
	userID uint,
	idempotencyKey string,
	dto domain.CreateRequest,
) (uint, uint, error) {
	paymentID := uint(0)
	attemptID := uint(0)
	err := ps.paymentUow.Tx(ctx, func(tx payment.PayTx) error {
		// 생성된 payment가 있는지 확인 후 없으면 payment 생성
		exist, err := tx.PaymentReader().FindByUserAndOrderID(ctx, userID, dto.OrderID)

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

// mappingUpdateStatus 업데이트 상태 맵
func (ps *PaymentService) mappingUpdateStatus(response pg.PGResponse) map[string]interface{} {
	switch response {
	case pg.Succeeded:
		return map[string]interface{}{
			"payment_status":     domain.Succeeded,
			"idempotency_status": idempotencydomain.StatusSucceeded,
			"order_status":       orderdomain.StatusCompleted,
			"attempt_status":     domain.AttemptStatusSucceeded,
		}
	case pg.Rejected:
		return map[string]interface{}{
			"payment_status":     domain.Rejected,
			"idempotency_status": idempotencydomain.StatusFailed,
			"order_status":       orderdomain.StatusFailed,
			"attempt_status":     domain.AttemptStatusRejected,
		}
	default:
		return map[string]interface{}{
			"payment_status":     domain.Failed,
			"idempotency_status": idempotencydomain.StatusFailed,
			"order_status":       orderdomain.StatusFailed,
			"attempt_status":     domain.AttemptStatusFailed,
		}
	}
}

// updateStatusTx 상태 변경 트랜잭션
func (ps *PaymentService) updateStatusTx(
	ctx context.Context,
	vo UpdateStatusVo,
) error {
	err := ps.paymentUow.Tx(ctx, func(tx payment.PayTx) error {
		paymentStatusField := map[string]interface{}{
			"status": vo.statusMap["payment_status"].(domain.PaymentStatus),
		}

		if vo.statusMap["payment_status"].(domain.PaymentStatus) == domain.Succeeded {
			paymentStatusField["paid_at"] = time.Now()
		}

		// payment 업데이트
		paymentStatusErr := tx.PaymentsWriter().Update(ctx, vo.PaymentID, paymentStatusField)

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

		// attempt 업데이트
		attemptStatusErr := tx.AttemptsWriter().Update(ctx, vo.AttemptID, map[string]interface{}{
			"status":         vo.statusMap["attempt_status"].(domain.AttemptStatus),
			"failure_reason": vo.failureReason,
		})
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
			vo.UserID,
			vo.idempotencyKey,
			idempotencydomain.ScopePayOrder,
			map[string]interface{}{
				"status": vo.statusMap["idempotency_status"].(idempotencydomain.Status),
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
		updateOrderErr := tx.OrdersWriter().Update(ctx, vo.OrderID, map[string]interface{}{
			"status": vo.statusMap["order_status"].(orderdomain.Status),
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
	idempotencyKey string,
	idempotencyStatus idempotencydomain.Status,
	dto domain.CreateRequest,
	txErr error,
) {
	// 재시도 로직 같은 것이 현재 별도로 없기 때문에, 재시도 구현 후에 레디스로 멱등키 상태 저장하는 부분은 미사용으로
	// 아니면 있어도 나쁘지 않을지도?
	// TODO 재시도 로직 구현 후에 코드 수정
	setErr := ps.idempotencyRedisRepo.SetIdempotencyStatus(ctx, idempotencyKey, idempotencyStatus)

	if setErr != nil {
		ps.logger.ErrorContext(ctx, "update payment status, set idempotency status failed",
			"tx err", txErr,
			"idempotency err", setErr)
		_ = ps.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body: fmt.Sprintf(
				"update payment status failed: %s, set idempotency status failed: %s, info: %v",
				txErr.Error(), setErr.Error(), dto),
		})
		return
	}

	ps.logger.ErrorContext(ctx, "update payment status failed", "tx err", txErr)
	_ = ps.slackSender.Send(ctx, notification.Message{
		Channel: notification.ChannelSlack,
		To:      "slack bot",
		Title:   "",
		Body:    fmt.Sprintf("update payment status failed: %s, info: %v", txErr.Error(), dto),
	})
}
