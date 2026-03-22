package scheduler

import (
	"context"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/application/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/timer"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/logger"
)

type Worker struct {
	timerRepo         timer.Repository
	transitionUseCase *instance.TransitionInstanceUseCase
	ticker            *time.Ticker
	quit              chan struct{}
	interval          time.Duration
}

func NewWorker(
	timerRepo timer.Repository,
	transitionUseCase *instance.TransitionInstanceUseCase,
	interval time.Duration,
) *Worker {
	return &Worker{
		timerRepo:         timerRepo,
		transitionUseCase: transitionUseCase,
		interval:          interval,
		quit:              make(chan struct{}),
	}
}

func (w *Worker) Start() {
	w.ticker = time.NewTicker(w.interval)
	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.processTimers()
			case <-w.quit:
				w.ticker.Stop()
				return
			}
		}
	}()
	logger.Info("⏰ Scheduler started", "interval", w.interval)
}

func (w *Worker) Stop() {
	close(w.quit)
	logger.Info("🛑 Scheduler stopped")
}

func (w *Worker) processTimers() {
	ctx := context.Background()
	timers, err := w.timerRepo.FindPending(ctx, 10) // Batch size 10
	if err != nil {
		logger.Error("Failed to fetch pending timers", "error", err)
		return
	}

	if len(timers) == 0 {
		return
	}

	logger.Debug("Processing timers", "count", len(timers))

	for _, t := range timers {
		w.processTimer(ctx, t)
	}
}

func (w *Worker) processTimer(ctx context.Context, t *timer.Timer) {
	logger.Info("Triggering timeout", "timer_id", t.ID(), "instance_id", t.InstanceID(), "event", t.EventOnTimeout())

	// Execute Transition
	cmd := instance.TransitionInstanceCommand{
		InstanceID: t.InstanceID().String(),
		Event:      t.EventOnTimeout(),
		ActorID:    "system-scheduler", // System actor
		Reason:     "Timer Expired",
	}

	_, err := w.transitionUseCase.Execute(ctx, cmd)
	if err != nil {
		logger.Error("Failed to execute timeout transition",
			"timer_id", t.ID(),
			"instance_id", t.InstanceID(),
			"error", err,
			"retry_count", t.RetryCount(),
		)

		if t.CanRetry() {
			t.IncrementRetry(err)
			if saveErr := w.timerRepo.Save(ctx, t); saveErr != nil {
				logger.Error("Failed to save timer retry state", "timer_id", t.ID(), "error", saveErr)
			} else {
				logger.Info("Timer scheduled for retry",
					"timer_id", t.ID(),
					"retry_count", t.RetryCount(),
					"next_retry_at", t.NextRetryAt(),
				)
			}
		} else {
			t.MarkFailed()
			if saveErr := w.timerRepo.Save(ctx, t); saveErr != nil {
				logger.Error("Failed to save timer failed state", "timer_id", t.ID(), "error", saveErr)
			}
			logger.Warn("Timer retries exhausted, marked as failed",
				"timer_id", t.ID(),
				"instance_id", t.InstanceID(),
				"last_error", t.LastError(),
				"retry_count", t.RetryCount(),
			)
		}
	} else {
		// Success: Mark completed
		t.MarkCompleted()
		if err := w.timerRepo.Save(ctx, t); err != nil {
			logger.Error("Failed to update timer status", "timer_id", t.ID(), "error", err)
		}
	}
}
