package executor

import (
	"context"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
)

// RetryWorkerWrapper wraps a Worker to provide retry capabilities based on the task's RetryPolicy.
type RetryWorkerWrapper struct {
	inner    Worker
	eventBus *events.EventBus
}

// NewRetryWorkerWrapper creates a new wrapper around a worker.
func NewRetryWorkerWrapper(inner Worker, eventBus *events.EventBus) *RetryWorkerWrapper {
	return &RetryWorkerWrapper{
		inner:    inner,
		eventBus: eventBus,
	}
}

// Execute performs the task execution, applying the retry policy if execution fails.
func (r *RetryWorkerWrapper) Execute(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (map[string]any, error) {
	var lastErr error
	var output map[string]any

	policy := task.RetryPolicy
	if policy == nil {
		// Default to no retries if policy is missing
		policy = &core.RetryPolicy{
			MaxRetries: 0,
			Strategy:   core.RetryStrategyImmediate,
		}
	}

	for attempt := 1; attempt <= policy.MaxRetries+1; attempt++ {
		task.SetAttempt(attempt)

		output, lastErr = r.inner.Execute(ctx, execCtx, task)
		if lastErr == nil {
			return output, nil // Success
		}

		if attempt > policy.MaxRetries {
			break
		}

		delay := policy.CalculateDelay(attempt)

		r.eventBus.Publish(events.Event{
			Type:       events.TaskRetried,
			TaskID:     task.ID,
			WorkflowID: execCtx.WorkflowID,
			Timestamp:  time.Now(),
			Payload: map[string]any{
				"attempt": attempt,
				"error":   lastErr.Error(),
				"delay":   delay.String(),
			},
		})

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Retry after delay
		}
	}

	return nil, lastErr
}
