package anthropic

import (
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

func mapGenerateRequest(req providers.GenerateRequest) anthropicGenerateRequest {
	msgs := []anthropicMessage{}
	system := ""

	for _, m := range req.Messages {
		if m.Role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
		} else {
			msgs = append(msgs, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024 // Anthropic requires max_tokens
	}

	return anthropicGenerateRequest{
		Model:       req.Model,
		Messages:    msgs,
		System:      system,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}
}

func mapGenerateResponse(resp anthropicGenerateResponse) *providers.GenerateResponse {
	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	return &providers.GenerateResponse{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      content,
		FinishReason: resp.StopReason,
		Usage: providers.TokenUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}
