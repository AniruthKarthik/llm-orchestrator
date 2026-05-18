package executor

import (
	"context"
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

func TestOutputValidationMiddleware(t *testing.T) {
	// 1. Create a task with a schema
	schema := map[string]string{
		"score": "int",
		"name":  "string",
	}
	task := core.NewTask("t1", "wf-1", "test", "desc", nil, nil).WithOutputSchema(schema)

	// 2. Create a handler that returns valid output
	validHandler := func(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (map[string]any, error) {
		return map[string]any{
			"score": 100,
			"name":  "test",
		}, nil
	}

	// 3. Create a handler that returns invalid output (missing field)
	missingFieldHandler := func(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (map[string]any, error) {
		return map[string]any{
			"score": 100,
		}, nil
	}

	// 4. Create a handler that returns invalid output (wrong type)
	wrongTypeHandler := func(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (map[string]any, error) {
		return map[string]any{
			"score": "100", // should be int
			"name":  "test",
		}, nil
	}

	middleware := OutputValidationMiddleware(validHandler)
	ctx := context.Background()
	art := core.NewArtifactRegistry()
	mem := core.NewMemoryRegistry()
	tr := core.NewToolRegistry()
	tp := core.NewToolPolicy()
	execCtx := NewExecutionContext("wf-1", art, mem, tr, tp)

	// Test valid output
	_, err := middleware(ctx, execCtx, task)
	if err != nil {
		t.Errorf("expected no error for valid output, got %v", err)
	}

	// Test missing field
	middleware = OutputValidationMiddleware(missingFieldHandler)
	_, err = middleware(ctx, execCtx, task)
	if err == nil {
		t.Error("expected error for missing field, got nil")
	}

	// Test wrong type
	middleware = OutputValidationMiddleware(wrongTypeHandler)
	_, err = middleware(ctx, execCtx, task)
	if err == nil {
		t.Error("expected error for wrong type, got nil")
	}
}
