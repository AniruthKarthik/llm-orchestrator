package anthropic

import (
	"context"
	"fmt"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/httpclient"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

type AnthropicProvider struct {
	client *httpclient.Client
	apiKey string
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		client: httpclient.NewClient(
			"", // apiKey is passed via headers instead of Bearer token
			"https://api.anthropic.com/v1",
			30*time.Second,
		),
		apiKey: apiKey,
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		SupportsTools:     true,
		SupportsStreaming: true,
		SupportsVision:    true,
	}
}

func (p *AnthropicProvider) Stream(
	ctx context.Context,
	request providers.GenerateRequest,
) (<-chan providers.StreamChunk, <-chan error) {
	chunkChan := make(chan providers.StreamChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)
		errChan <- fmt.Errorf("streaming not implemented for anthropic provider")
	}()

	return chunkChan, errChan
}
