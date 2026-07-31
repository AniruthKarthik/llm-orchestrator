package executor

import (
	"context"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

// TaskHandler defines the function signature for task execution.
type TaskHandler func(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (map[string]any, error)

// TaskMiddleware is a function that wraps a TaskHandler.
type TaskMiddleware func(next TaskHandler) TaskHandler

// WorkflowHandler defines the function signature for workflow execution.
type WorkflowHandler func(ctx context.Context, workflow *core.Workflow) error

// WorkflowMiddleware is a function that wraps a WorkflowHandler.
type WorkflowMiddleware func(next WorkflowHandler) WorkflowHandler

// ApplyTaskMiddleware chains multiple middlewares into a single handler.
func ApplyTaskMiddleware(handler TaskHandler, middlewares ...TaskMiddleware) TaskHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// ApplyWorkflowMiddleware chains multiple middlewares into a single handler.
func ApplyWorkflowMiddleware(handler WorkflowHandler, middlewares ...WorkflowMiddleware) WorkflowHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// OutputValidationMiddleware validates task output against its schema.
func OutputValidationMiddleware(next TaskHandler) TaskHandler {
	return func(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (map[string]any, error) {
		output, err := next(ctx, execCtx, task)
		if err != nil {
			return output, err
		}

		if valErr := task.ValidateOutput(output); valErr != nil {
			return output, valErr
		}

		return output, nil
	}
}
