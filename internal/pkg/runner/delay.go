package runner

import (
	"context"
	"time"
)

type DelayedTaskRunner interface {
	RunAfter(
		parentCtx context.Context,
		delay time.Duration,
		task func(ctx context.Context),
	)
}

type timerTaskRunner struct{}

func NewTimerTaskRunner() DelayedTaskRunner {
	return timerTaskRunner{}
}

func (timerTaskRunner) RunAfter(parentCtx context.Context, delay time.Duration, task func(ctx context.Context)) {
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		<-timer.C
		task(context.WithoutCancel(parentCtx))
	}()
}
