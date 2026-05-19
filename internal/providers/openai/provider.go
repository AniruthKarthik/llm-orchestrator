package openai

import (
	"context"
	"encoding/json"
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
		SupportsStreaming: false,
		SupportsVision:    true,
	}
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	resp, err := p.client.Get(ctx, "/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

