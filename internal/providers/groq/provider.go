package groq

import (
	"context"
	"encoding/json"
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

func (p *GroqProvider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		SupportsTools:     true,
		SupportsStreaming: false,
		SupportsVision:    false,
	}
}

func (p *GroqProvider) ListModels(ctx context.Context) ([]string, error) {
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

