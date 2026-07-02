package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"order_system/internal/notification"
	productdomain "order_system/internal/product/domain"
	productrepository "order_system/internal/product/repository"
	"time"
)

type InventoryRestoreWorker struct {
	slackSender        notification.Sender
	inventoryGormRepo  productrepository.InventoryJobGormRepository
	inventoryRedisRepo productrepository.InventoryRedisRepository
}

func NewInventoryRestoreWorker(
	slackSender notification.Sender,
	inventoryGormRepo productrepository.InventoryJobGormRepository,
	inventoryRedisRepo productrepository.InventoryRedisRepository,
) InventoryRestoreWorker {
	return InventoryRestoreWorker{
		slackSender:        slackSender,
		inventoryGormRepo:  inventoryGormRepo,
		inventoryRedisRepo: inventoryRedisRepo,
	}
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
	jobs, err := w.inventoryGormRepo.FindDueJob(ctx, 300)

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
		case productdomain.RestoreTargetDB:
			w.handleDBJob(ctx, job)
		case productdomain.RestoreTargetRedis:
			w.handleRedisJob(ctx, job)
		default:
			slog.WarnContext(ctx, "unknown restore target", "job", job)
			w.updateInventoryRestore(ctx, job, "unknown restore target", productdomain.JobFailed)
		}
	}

}

func (w *InventoryRestoreWorker) handleDBJob(ctx context.Context, job productdomain.InventoryRestoreJob) {
	// TODO DB 재고 복구 로직 필요시 구현, 현재는 db job이 별도로 없음
	return
}

func (w *InventoryRestoreWorker) handleRedisJob(ctx context.Context, job productdomain.InventoryRestoreJob) {
	switch job.Operation {
	case productdomain.DecreaseReserved:
		w.decreaseReservedQuantity(ctx, job)
	case productdomain.IncreaseReserved:
		// TODO 필요시 구현
	default:
		slog.WarnContext(ctx, "unknown operation", "job", job)
	}
}

func (w *InventoryRestoreWorker) decreaseReservedQuantity(ctx context.Context, job productdomain.InventoryRestoreJob) {
	fail := w.inventoryRedisRepo.RestoreProductReservedQuantityInRedis(
		ctx,
		job.OrderNo,
		productrepository.RestoreItem{
			ProductID: job.ProductID,
			Quantity:  job.Quantity,
		})

	/*
		3가지 상태로 변경할 수 있다.
		1. 실패가 없으면 성공으로 업데이트
		2. 실패하고 재시도 횟수 초과하면 실패로 업데이트
		3. 실패하고 재시도 횟수 남으면 재시도로 업데이트
	*/
	if fail != nil {
		status := w.resolveFailedStatus(job, fail)
		w.updateInventoryRestore(ctx, job, fail.Err.Error(), status)
		return
	}

	// 실패가 없다면 성공 처리
	slog.InfoContext(ctx, "restore reserved quantity succeeded", "info", job)
	w.updateInventoryRestore(ctx, job, "", productdomain.JobSucceeded)
}

func (w *InventoryRestoreWorker) resolveFailedStatus(
	job productdomain.InventoryRestoreJob,
	fail *productrepository.RestoreFailed,
) productdomain.JobStatus {
	if errors.Is(fail.Err, productrepository.ErrRedisNoneReservedQuantity) ||
		errors.Is(fail.Err, productrepository.ErrRedisInvalidQuantity) ||
		!w.isRetryable(job) {

		return productdomain.JobFailed
	}

	return productdomain.JobRetryable
}

func (w *InventoryRestoreWorker) isRetryable(job productdomain.InventoryRestoreJob) bool {
	if job.RetryCount <= 3 && (job.Status == productdomain.JobRetryable || job.Status == productdomain.JobPending) {
		return true
	}

	return false
}

func (w *InventoryRestoreWorker) updateInventoryRestore(
	ctx context.Context,
	job productdomain.InventoryRestoreJob,
	error string,
	status productdomain.JobStatus,
) {
	retryCount := job.RetryCount
	// 실패 처리의 경우 알림 보내서 직접 처리하도록
	if status == productdomain.JobFailed {
		slog.ErrorContext(ctx, "restore product reserved quantity failed",
			"info", job,
			"error", error,
		)

		_ = w.slackSender.Send(ctx, notification.Message{
			Channel: notification.ChannelSlack,
			To:      "slack bot",
			Title:   "",
			Body:    fmt.Sprintf("restore product reserved quantity failed, info: %v, error: %s", job, error),
		})
	}

	if status == productdomain.JobRetryable || status == productdomain.JobPending {
		retryCount++
	}

	err := w.inventoryGormRepo.UpdateJob(
		ctx,
		productdomain.InventoryRestoreJobFindConstraint{
			OrderNo:   job.OrderNo,
			ProductID: job.ProductID,
			Target:    productdomain.RestoreTargetRedis,
			Operation: productdomain.DecreaseReserved,
		},
		productdomain.InventoryRestoreJobUpdateContext{
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
			Body:    fmt.Sprintf("update inventory restore job failed, info: %v, error: %s", job, err),
		})
	}
}
