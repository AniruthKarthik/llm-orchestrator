package openai

import (
	"context"
	"fmt"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/httpclient"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

type OpenAIProvider struct {
	client *httpclient.Client
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		client: httpclient.NewClient(
			apiKey,
			"https://api.openai.com/v1",
			30*time.Second,
		),
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		SupportsTools:     true,
		SupportsStreaming: true,
		SupportsVision:    true,
	}
}

func (p *OpenAIProvider) Stream(
	ctx context.Context,
	request providers.GenerateRequest,
) (<-chan providers.StreamChunk, <-chan error) {
	chunkChan := make(chan providers.StreamChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)
		errChan <- fmt.Errorf("streaming not implemented for openai provider")
	}()

	return chunkChan, errChan
}
