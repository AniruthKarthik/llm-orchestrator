package llm

import (
	"context"
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

type LLMWorker struct {
	provider providers.Provider
}

func NewLLMWorker(
	provider providers.Provider,
) *LLMWorker {
	return &LLMWorker{
		provider: provider,
	}
}

func (w *LLMWorker) Name() string {
	return "llm_worker"
}

func (w *LLMWorker) Execute(
	ctx context.Context,
	input map[string]any,
) (map[string]any, error) {

	model, ok := input["model"].(string)
	if !ok {
		return nil, fmt.Errorf(
			"invalid model",
		)
	}

	messages, ok := input["messages"].([]providers.Message)

	if !ok {
		return nil, fmt.Errorf(
			"invalid messages",
		)
	}

	req := providers.GenerateRequest{
		Model:    model,
		Messages: messages,
	}

	resp, err := w.provider.Generate(
		ctx,
		req,
	)
	if err != nil {
		return nil, err
	}

	output := map[string]any{
		"response": resp.Content,
		"model":    model,
	}

	return output, nil
}
