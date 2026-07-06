package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"order_system/internal/notification"
	orderrepository "order_system/internal/order/repository"
	"order_system/internal/pkg/apperr/dberr"
	"order_system/internal/pkg/apperr/rediserr"
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
}

func NewInventoryRestoreWorker(
	slackSender notification.Sender,
	inventoryJobGormRepo productrepository.InventoryJobGormRepository,
	inventoryRedisRepo productrepository.InventoryRedisRepository,
	inventoryGormRepo productrepository.InventoryGormRepository,
	orderItemRepo orderrepository.OrderItemGormRepository,
) InventoryRestoreWorker {
	return InventoryRestoreWorker{
		slackSender:          slackSender,
		inventoryJobGormRepo: inventoryJobGormRepo,
		inventoryRedisRepo:   inventoryRedisRepo,
		inventoryGormRepo:    inventoryGormRepo,
		orderItemRepo:        orderItemRepo,
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
			w.updateInventoryJob(ctx, job, "unknown restore target", productdomain.JobFailed)
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
		w.updateInventoryJob(ctx, job, "unknown operation", productdomain.JobFailed)
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
		w.updateInventoryJob(ctx, job, "unknown operation", productdomain.JobFailed)
	}
}

// 레디스 예약 재고 감소
func (w *InventoryRestoreWorker) decreaseReservedQuantityInRedis(ctx context.Context, job productdomain.InventoryJob) {
	var payload restoreQuantityPayload
	var err error
	err = json.Unmarshal([]byte(job.Payload), &payload)

	if err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal payload", "err", err, "job", job)
		w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed)
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
		w.updateInventoryJob(ctx, job, fail.Err.Error(), status)
		return
	}

	// 실패가 없다면 성공 처리
	slog.InfoContext(ctx, "restore reserved quantity succeeded", "job", job)
	w.updateInventoryJob(ctx, job, "", productdomain.JobSucceeded)
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
) {
	retryCount := job.RetryCount
	// 실패 처리의 경우 알림 보내서 직접 처리하도록
	if status == productdomain.JobFailed {
		slog.ErrorContext(ctx, "inventory job failed",
			"job", job,
			"error", error,
		)

		_ = w.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body:    fmt.Sprintf("inventory job failed, job: %v, error: %s", job, error),
		})
	}

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
		slog.ErrorContext(ctx, "update inventory restore job failed",
			"info", job,
			"error", err,
		)
		_ = w.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body:    fmt.Sprintf("update inventory restore job failed, job: %v, error: %s", job, err),
		})
	}
}

func (w *InventoryRestoreWorker) confirmSaleInDB(ctx context.Context, job productdomain.InventoryJob) {
	var payload confirmSalePayload
	var err error
	err = json.Unmarshal([]byte(job.Payload), &payload)

	if err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal payload", "err", err, "job", job)
		w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed)
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
		w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed)
		return
	}

	result, err := w.inventoryRedisRepo.GetConfirmSaleDoneKey(ctx, item.OrderID, item.ProductID)

	if err != nil && !errors.Is(err, rediserr.ErrNotFound) {
		slog.ErrorContext(ctx, "failed to get confirm sale done key in inventory job worker",
			"err", err, "job", job)
		_ = w.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body: fmt.Sprintf("failed to get confirm sale done key in inventory job worker, job: %v, error: %s",
				job, err),
		})
		w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed)
		return
	}

	if result != "" {
		w.updateInventoryJob(ctx, job, "", productdomain.JobSucceeded)
		return
	}

	err = w.inventoryGormRepo.UpdateSoldQuantity(ctx, item.ProductID, item.Quantity)

	if err != nil {
		classify := dberr.ClassifyDBError(err)

		if (classify == dberr.DBErrorRetryable || classify == dberr.DBErrorAmbiguous) && w.isRetryable(job) {
			w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobRetryable)
		} else {
			w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed)
		}
		return
	}

	err = w.inventoryRedisRepo.SetConfirmSaleDoneKey(ctx, item.OrderID, item.ProductID)

	if err != nil {
		slog.ErrorContext(ctx, "failed to set confirm sale done key in inventory job worker",
			"err", err, "job", job)
		_ = w.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body: fmt.Sprintf("failed to set confirm sale done key in inventory job worker, job: %v, error: %s",
				job, err),
		})
		w.updateInventoryJob(ctx, job, err.Error(), productdomain.JobFailed)
		return
	}

	slog.InfoContext(ctx, "update inventory restore job succeeded", "job", job)
	w.updateInventoryJob(ctx, job, "", productdomain.JobSucceeded)
}
