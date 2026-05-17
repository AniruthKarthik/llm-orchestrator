package gemini

import (
	"context"
	"fmt"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/httpclient"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

type GeminiProvider struct {
	client *httpclient.Client
	apiKey string
}

func NewGeminiProvider(apiKey string) *GeminiProvider {
	return &GeminiProvider{
		client: httpclient.NewClient(
			"",
			"https://generativelanguage.googleapis.com/v1beta",
			30*time.Second,
		),
		apiKey: apiKey,
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		SupportsTools:     true,
		SupportsStreaming: true,
		SupportsVision:    true,
	}
}

func (p *GeminiProvider) Stream(
	ctx context.Context,
	request providers.GenerateRequest,
) (<-chan providers.StreamChunk, <-chan error) {
	chunkChan := make(chan providers.StreamChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)
		errChan <- fmt.Errorf("streaming not implemented for gemini provider")
	}()

	return chunkChan, errChan
}
