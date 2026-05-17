package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

func (p *GeminiProvider) Generate(
	ctx context.Context,
	request providers.GenerateRequest,
) (*providers.GenerateResponse, error) {

	body := mapGenerateRequest(request)

	headers := map[string]string{
		"x-goog-api-key": p.apiKey,
	}

	// Gemini API endpoint format: /models/{model}:generateContent
	endpoint := fmt.Sprintf("/models/%s:generateContent", request.Model)

	resp, err := p.client.Post(
		ctx,
		endpoint,
		headers,
		body,
	)

	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gemini api error: %d", resp.StatusCode)
	}

	var geminiResp geminiGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w", err)
	}

	return mapGenerateResponse(geminiResp), nil
}
