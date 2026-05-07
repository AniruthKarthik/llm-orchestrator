package tools

import (
	"context"
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/reliability"
)

type CircuitBreakerTool struct {
	inner tool
	cb    *reliability.CircuitBreaker
}

type tool = Tool

// NewCircuitBreakerTool wraps an existing tool with a circuit breaker.
func NewCircuitBreakerTool(inner Tool, cb *reliability.CircuitBreaker) *CircuitBreakerTool {
	return &CircuitBreakerTool{
		inner: inner,
		cb:    cb,
	}
}

func (t *CircuitBreakerTool) Name() string { return t.inner.Name() }

func (t *CircuitBreakerTool) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	var result map[string]any
	err := t.cb.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		result, execErr = t.inner.Execute(ctx, input)

		return execErr
	})

	if err != nil {
		return nil, fmt.Errorf("too; %s :%w ", t.Name(), err)
	}

	return result, nil
}
