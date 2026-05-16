package groq

import (
	"context"
	"fmt"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/httpclient"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

type GroqProvider struct {
	client *httpclient.Client
}

func NewGroqProvider(
	apiKey string,
) *GroqProvider {
	return &GroqProvider{
		client: httpclient.NewClient(
			apiKey,
			"https://api.groq.com/openai/v1",
			30*time.Second,
		),
	}
}

func (p *GroqProvider) Name() string {
	return "groq"
}

func (p *GroqProvider) Stream(
	ctx context.Context,
	request providers.GenerateRequest,
) (<-chan providers.StreamChunk, <-chan error) {
	chunkChan := make(chan providers.StreamChunk)
	errChan := make(chan error, 1)

	// Minimal implementation: just return an error or a single chunk for now
	// Real implementation would use p.client.Post with streaming enabled
	go func() {
		defer close(chunkChan)
		defer close(errChan)
		errChan <- fmt.Errorf("streaming not implemented for groq provider")
	}()

	return chunkChan, errChan
}

