package executor

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

// RecoverMiddleware handles panics in task execution.
func RecoverMiddleware(next TaskHandler) TaskHandler {
	return func(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (output map[string]any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("task execution panicked: %v\n%s", r, debug.Stack())
			}
		}()
		return next(ctx, execCtx, task)
	}
}
