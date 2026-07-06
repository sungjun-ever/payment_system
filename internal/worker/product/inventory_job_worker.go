package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"order_system/internal/notification"
	orderrepository "order_system/internal/order/repository"
	"order_system/internal/pkg/apperr/dberr"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"
	"time"
)

type InventoryRestoreWorker struct {
	slackSender          notification.Sender
	inventoryJobGormRepo productrepository.InventoryJobGormRepository
	inventoryRedisRepo   productrepository.InventoryRedisRepository
	inventoryGormRepo    productrepository.InventoryGormRepository
	orderItemRepo        orderrepository.OrderItemGormRepository
	productStore         ProductStore
}

func NewInventoryRestoreWorker(
	slackSender notification.Sender,
	inventoryJobGormRepo productrepository.InventoryJobGormRepository,
	inventoryRedisRepo productrepository.InventoryRedisRepository,
	inventoryGormRepo productrepository.InventoryGormRepository,
	orderItemRepo orderrepository.OrderItemGormRepository,
	productStore ProductStore,
) InventoryRestoreWorker {
	return InventoryRestoreWorker{
		slackSender:          slackSender,
		inventoryJobGormRepo: inventoryJobGormRepo,
		inventoryRedisRepo:   inventoryRedisRepo,
		inventoryGormRepo:    inventoryGormRepo,
		orderItemRepo:        orderItemRepo,
		productStore:         productStore,
	}
}

type restoreQuantityPayload struct {
	OrderNo   string
	OrderID   uint
	ProductID uint
	Quantity  int
}

type confirmSalePayload struct {
	OrderID   uint
	ProductID uint
}

func (w *InventoryRestoreWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *InventoryRestoreWorker) process(ctx context.Context) {
	// 테이블에서 데이터를 300개씩 가져와 진행한다.
	jobs, err := w.inventoryJobGormRepo.FindDueJob(ctx, 300)

	if err != nil {
		_ = w.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body:    fmt.Sprintf("failed to find due job: %s", err.Error()),
		})
		slog.ErrorContext(ctx, "failed to find due job", "error", err)
		return
	}

	// 처리해야할 작업이 있으면 시작
	for _, job := range jobs {
		switch job.Target {
		case productdomain.TargetDB:
			w.handleDBJob(ctx, job)
		case productdomain.TargetRedis:
			w.handleRedisJob(ctx, job)
		default:
			slog.WarnContext(ctx, "unknown restore target", "job", job)
			if err = w.updateInventoryJob(ctx, job, "unknown restore target", productdomain.JobFailed); err != nil {
				w.sendNotification(ctx, err.Error())
			}
		}
	}

}

// DB job 분류
func (w *InventoryRestoreWorker) handleDBJob(ctx context.Context, job productdomain.InventoryJob) {
	switch job.Operation {
	case productdomain.ConfirmSale:
		w.confirmSaleInDB(ctx, job)
	default:
		slog.WarnContext(ctx, "unknown operation", "job", job)
		if err := w.updateInventoryJob(ctx, job, "unknown operation", productdomain.JobFailed); err != nil {
			w.sendNotification(ctx, err.Error())
		}
	}
	return
}

// Redis job 분류
func (w *InventoryRestoreWorker) handleRedisJob(ctx context.Context, job productdomain.InventoryJob) {
	switch job.Operation {
	case productdomain.DecreaseReserved:
		w.decreaseReservedQuantityInRedis(ctx, job)
	case productdomain.IncreaseReserved:
		// 필요시 구현
	default:
		slog.WarnContext(ctx, "unknown operation", "job", job)
		if err := w.updateInventoryJob(ctx, job, "unknown operation", productdomain.JobFailed); err != nil {
			w.sendNotification(ctx, err.Error())
		}
	}
}

// 레디스 예약 재고 감소
func (w *InventoryRestoreWorker) decreaseReservedQuantityInRedis(ctx context.Context, job productdomain.InventoryJob) {
	var payload restoreQuantityPayload
	var err error
	err = json.Unmarshal([]byte(job.Payload), &payload)

	if err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal payload", "err", err, "job", job)
		if err = w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed); err != nil {
			w.sendNotification(ctx, err.Error())
		}
		return
	}

	fail := w.inventoryRedisRepo.RestoreProductReservedQuantityInRedis(
		ctx,
		payload.OrderNo,
		payload.OrderID,
		productrepository.RestoreItem{
			ProductID: payload.ProductID,
			Quantity:  payload.Quantity,
		})

	/*
		3가지 상태로 변경할 수 있다.
		1. 실패가 없으면 성공으로 업데이트
		2. 실패하고 재시도 횟수 초과하면 실패로 업데이트
		3. 실패하고 재시도 횟수 남으면 재시도로 업데이트
	*/
	if fail != nil {
		status := w.resolveRedisFailedStatus(job, fail)
		if err = w.updateInventoryJob(ctx, job, fail.Err.Error(), status); err != nil {
			w.sendNotification(ctx, err.Error())
		}
		return
	}

	// 실패가 없다면 성공 처리
	slog.InfoContext(ctx, "restore reserved quantity succeeded", "job", job)
	if err = w.updateInventoryJob(ctx, job, "", productdomain.JobSucceeded); err != nil {
		w.sendNotification(ctx, err.Error())
	}
}

// 레디스 실패 상태 분류
func (w *InventoryRestoreWorker) resolveRedisFailedStatus(
	job productdomain.InventoryJob,
	fail *productrepository.RestoreFailed,
) productdomain.JobStatus {
	if errors.Is(fail.Err, productrepository.ErrRedisNoneReservedQuantity) ||
		errors.Is(fail.Err, productrepository.ErrRedisInvalidQuantity) ||
		!w.isRetryable(job) {

		return productdomain.JobFailed
	}

	return productdomain.JobRetryable
}

// 재시도 가능 여부 판단
func (w *InventoryRestoreWorker) isRetryable(job productdomain.InventoryJob) bool {
	if job.RetryCount <= 3 && (job.Status == productdomain.JobRetryable || job.Status == productdomain.JobPending) {
		return true
	}

	return false
}

func (w *InventoryRestoreWorker) updateInventoryJob(
	ctx context.Context,
	job productdomain.InventoryJob,
	error string,
	status productdomain.JobStatus,
) error {
	retryCount := job.RetryCount

	if status == productdomain.JobRetryable || status == productdomain.JobPending {
		retryCount++
	}

	err := w.inventoryJobGormRepo.UpdateJob(
		ctx,
		job.ID,
		productdomain.InventoryJobUpdateContext{
			RetryCount:  retryCount,
			Status:      status,
			NextRetryAt: time.Now().Add(time.Minute * 1),
			LastError:   error,
			UpdatedAt:   time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to update inventory restore job status, job: %v, status: %s, error: %s",
			job, status, err)
	}

	return nil
}

func (w *InventoryRestoreWorker) confirmSaleInDB(ctx context.Context, job productdomain.InventoryJob) {
	var payload confirmSalePayload
	var err error
	err = json.Unmarshal([]byte(job.Payload), &payload)

	if err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal payload", "err", err, "job", job)
		if err = w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed); err != nil {
			w.sendNotification(ctx, err.Error())
		}
		return
	}

	// order item에서 quantity 조회 -> inventory에 적용
	item, err := w.orderItemRepo.GetItemByOrderIDAndProductID(ctx, payload.OrderID, payload.ProductID)

	if err != nil {
		// order item 실패의 경우 수동 처리하도록
		slog.ErrorContext(ctx, "failed to get order item in inventory job worker",
			"err", err, "job", job)
		_ = w.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body: fmt.Sprintf("failed to get order item in inventory job worker, job: %v, error: %s",
				job, err),
		})

		if err = w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed); err != nil {
			w.sendNotification(ctx, err.Error())
		}

		return
	}

	err = w.productStore.Tx(ctx, func(tx ProductTx) error {
		inventoryMovementEntity := &productdomain.InventoryMovement{
			OrderID:   item.OrderID,
			ProductID: item.ProductID,
			Operation: productdomain.ConfirmSale,
			Quantity:  item.Quantity,
		}
		err = tx.InventoryMovementWriter().CreateInventoryMovement(ctx, inventoryMovementEntity)

		if err != nil {
			if errors.Is(err, dberr.ErrDuplicate) {
				return nil
			}
			return fmt.Errorf("failed to create inventory movement in inventory job worker, job: %v, error: %w",
				job, err)
		}

		err = tx.InventoryWriter().UpdateSoldQuantity(ctx, item.ProductID, item.Quantity)

		if err != nil {
			return fmt.Errorf("failed to update sold quantity in inventory job worker, job: %v, error: %w",
				job, err)
		}

		return nil
	})

	if err != nil {
		status := w.resolveDBFailedStatus(job, err)
		if updateErr := w.updateInventoryJob(ctx, job, err.Error(), status); updateErr != nil {
			w.sendNotification(ctx, updateErr.Error())
			return
		}
		if status == productdomain.JobFailed {
			w.sendNotification(ctx, fmt.Sprintf("inventory job failed, job: %v, error: %s", job, err.Error()))
		}
		return
	}

	slog.InfoContext(ctx, "update inventory restore job succeeded", "job", job)
	err = w.updateInventoryJob(ctx, job, "", productdomain.JobSucceeded)

	if err != nil {
		w.sendNotification(ctx, fmt.Sprintf("failed to update inventory restore job status, job: %v, status: %s, error: %s",
			job, productdomain.JobSucceeded, err))
	}
}

func (w *InventoryRestoreWorker) resolveDBFailedStatus(
	job productdomain.InventoryJob,
	err error,
) productdomain.JobStatus {
	classify := dberr.ClassifyDBError(err)
	if (classify == dberr.DBErrorRetryable || classify == dberr.DBErrorAmbiguous) && w.isRetryable(job) {
		return productdomain.JobRetryable
	}

	return productdomain.JobFailed
}

func (w *InventoryRestoreWorker) sendNotification(ctx context.Context, message string) {
	_ = w.slackSender.Send(ctx, notification.Message{
		Channel: notification.ChannelSlack,
		To:      "slack bot",
		Title:   "",
		Body:    message,
	})
}
