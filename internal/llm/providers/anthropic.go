package providers

import (
	"context"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/llm"
)

type AnthropicConfig struct {
	APIKey     string
	Model      string
	Timeout    time.Duration
	MaxRetries int
}

type AnthropicClient struct {
	cfg AnthropicConfig
}

func NewAnthropicClient(cfg AnthropicConfig) *AnthropicClient {
	return &AnthropicClient{cfg: cfg}
}

func (c *AnthropicClient) SendPrompt(ctx context.Context, prompt string) (*llm.Response, error) {
	// For now, delegate to mock behavior
	mock := llm.NewMockClient()
	return mock.SendPrompt(ctx, prompt)
}
