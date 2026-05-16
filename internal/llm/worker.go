package llm

import (
	"context"
	"fmt"

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

	model, ok := task.Input["model"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'model' in task input")
	}
	req.Model = model

	if messages, ok := task.Input["messages"].([]any); ok {
		for _, msg := range messages {
			if m, ok := msg.(map[string]any); ok {
				role, _ := m["role"].(string)
				content, _ := m["content"].(string)
				req.Messages = append(req.Messages, Message{
					Role:    role,
					Content: content,
				})
			}
		}
	} else if messages, ok := task.Input["messages"].([]Message); ok {
		req.Messages = messages
	}

	if temp, ok := task.Input["temperature"].(float64); ok {
		req.Temperature = temp
	}

	if maxTokens, ok := task.Input["max_tokens"].(float64); ok {
		req.MaxTokens = int(maxTokens)
	} else if maxTokens, ok := task.Input["max_tokens"].(int); ok {
		req.MaxTokens = maxTokens
	}

	resp, err := w.provider.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"content":      resp.Content,
		"total_tokens": resp.Usage.TotalTokens,
	}, nil
}
