package providers

import (
	"context"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/llm"
)

type OpenAIConfig struct {
	APIKey     string
	Model      string
	Timeout    time.Duration
	MaxRetries int
}

type OpenAIClient struct {
	cfg OpenAIConfig
}

func NewOpenAIClient(cfg OpenAIConfig) *OpenAIClient {
	return &OpenAIClient{cfg: cfg}
}

func (c *OpenAIClient) SendPrompt(ctx context.Context, prompt string) (*llm.Response, error) {
	// For now, delegate to mock behavior or return error if key missing
	mock := llm.NewMockClient()
	return mock.SendPrompt(ctx, prompt)
}
