package anthropic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

func (p *AnthropicProvider) Generate(
	ctx context.Context,
	request providers.GenerateRequest,
) (*providers.GenerateResponse, error) {

	body := mapGenerateRequest(request)

	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
	}

	resp, err := p.client.Post(
		ctx,
		"/messages",
		headers,
		body,
	)

	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("anthropic api error: %d", resp.StatusCode)
	}

	var anthropicResp anthropicGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode anthropic response: %w", err)
	}

	return mapGenerateResponse(anthropicResp), nil
}
