package groq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

func (p *GroqProvider) Generate(
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
		return nil, fmt.Errorf("groq request failed: %w", err)
	}

	// Always close the body
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			// This could be used by a retry middleware to decide whether to retry
			return nil, fmt.Errorf("groq api retryable error: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("groq api error: %d", resp.StatusCode)
	}

	var groqResp groqGenerateResponse

	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return nil, fmt.Errorf("failed to decode groq response: %w", err)
	}

	return mapGenerateResponse(groqResp), nil
}
