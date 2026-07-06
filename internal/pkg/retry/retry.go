package retry

import (
	"context"
	"log/slog"
	"order_system/internal/pkg/apperr/dberr"
	"time"
)

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

var defaultRetryPolicy = RetryPolicy{
	MaxAttempts: 3,
	BaseDelay:   100 * time.Millisecond,
	MaxDelay:    1 * time.Second,
}

func Retry(ctx context.Context, logger *slog.Logger, policy RetryPolicy, fn func() error) error {
	var lastErr error

	policy = validatedRetryPolicy(policy)

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		err := fn()

		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryable(err) {
			return err
		}

		if attempt == policy.MaxAttempts {
			break
		}

		delay := calculateDelay(policy, attempt)

		logger.Warn("retry",
			"attempt", attempt,
			"max_attempts", policy.MaxAttempts,
			"delay", delay,
			"error", err,
		)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

func isRetryable(err error) bool {
	return dberr.ClassifyDBError(err) == dberr.DBErrorRetryable
}

func validatedRetryPolicy(policy RetryPolicy) RetryPolicy {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}

	if policy.BaseDelay <= 0 {
		policy.BaseDelay = 100 * time.Millisecond
	}

	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 1 * time.Second
	}

	return policy
}

func calculateDelay(policy RetryPolicy, attempt int) time.Duration {
	delay := policy.BaseDelay * time.Duration(1<<(attempt-1))

	if delay > policy.MaxDelay {
		return policy.MaxDelay
	}

	return delay
}
