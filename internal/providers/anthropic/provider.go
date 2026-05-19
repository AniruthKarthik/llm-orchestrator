package anthropic

import (
	"context"
	"encoding/json"
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
		SupportsStreaming: false,
		SupportsVision:    true,
	}
}

func (p *AnthropicProvider) ListModels(ctx context.Context) ([]string, error) {
	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
	}
	resp, err := p.client.Get(ctx, "/models", headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Fallback for Anthropic if /models is not available
		return []string{"claude-3-5-sonnet-20240620", "claude-3-opus-20240229", "claude-3-haiku-20240307"}, nil
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

