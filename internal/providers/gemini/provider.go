package gemini

import (
	"context"
	"encoding/json"
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
		SupportsStreaming: false,
		SupportsVision:    true,
	}
}

func (p *GeminiProvider) ListModels(ctx context.Context) ([]string, error) {
	headers := map[string]string{
		"x-goog-api-key": p.apiKey,
	}
	resp, err := p.client.Get(ctx, "/models", headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		// Gemini returns "models/gemini-pro", we want just "gemini-pro"
		name := m.Name
		if len(name) > 7 && name[:7] == "models/" {
			name = name[7:]
		}
		models = append(models, name)
	}
	return models, nil
}

