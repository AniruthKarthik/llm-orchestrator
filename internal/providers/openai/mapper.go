package openai

import (
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

func mapGenerateRequest(req providers.GenerateRequest) openaiGenerateRequest {
	msgs := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openaiMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return openaiGenerateRequest{
		Model:       req.Model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
}

func mapGenerateResponse(resp openaiGenerateResponse) *providers.GenerateResponse {
	content := ""
	finishReason := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
		finishReason = resp.Choices[0].FinishReason
	}

	return &providers.GenerateResponse{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      content,
		FinishReason: finishReason,
		Usage: providers.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}
