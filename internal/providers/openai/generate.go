package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

func (p *OpenAIProvider) Generate(
	ctx context.Context,
	request providers.GenerateRequest,
) (*providers.GenerateResponse, error) {

	body := mapGenerateRequest(request)

	resp, err := p.client.Post(
		ctx,
		"/chat/completions",
		nil,
		body,
	)

	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("openai api error: %d", resp.StatusCode)
	}

	var openaiResp openaiGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode openai response: %w", err)
	}

	return mapGenerateResponse(openaiResp), nil
}
