package executor

import (
	"context"
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

func TestPanicRecoveryMiddleware(t *testing.T) {
	// 1. Create a task handler that panics
	panickingHandler := func(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (map[string]any, error) {
		panic("boom")
	}

	// 2. Wrap it with the middleware
	middleware := PanicRecoveryMiddleware(panickingHandler)

	// 3. Execute the handler
	ctx := context.Background()
	execCtx := NewExecutionContext("wf-1")
	task := core.NewTask("t1", "test", "desc", nil, nil)

	output, err := middleware(ctx, execCtx, task)

	// 4. Verify that it didn't crash and returned an error
	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}

	if output != nil {
		t.Errorf("expected nil output on panic, got %v", output)
	}

	expectedPrefix := "task panicked: boom"
	if len(err.Error()) < len(expectedPrefix) || err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected error message to start with '%s', got '%s'", expectedPrefix, err.Error())
	}
}
