package llm

import (
	"context"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
)

// LLMWorker implements the executor.Worker interface for LLM tasks.
type LLMWorker struct {
	provider Provider
}

func NewLLMWorker(p Provider) *LLMWorker {
	return &LLMWorker{provider: p}
}

func (w *LLMWorker) Execute(ctx context.Context, execCtx *executor.ExecutionContext, task *core.Task) (map[string]any, error) {
	// Extract request from task input
	var req Request
	// Mapping task input to Request... (Simplified for this phase)
	req.Model = task.Input["model"].(string)

	resp, err := w.provider.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"content":      resp.Content,
		"total_tokens": resp.Usage.TotalTokens,
	}, nil
}
