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
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf(
			"groq api error:%d",
			resp.StatusCode,
		)
	}

	var groqResp groqGenerateResponse

	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return nil, err
	}

	return mapGenerateResponse(groqResp), nil
}
