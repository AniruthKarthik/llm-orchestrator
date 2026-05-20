package groq

import (
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

func mapGenerateRequest(
	req providers.GenerateRequest,
) groqGenerateRequest {

	messages := make([]groqMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, groqMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return groqGenerateRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
	}
}

func mapGenerateResponse(
	resp groqGenerateResponse,
) *providers.GenerateResponse {

	if len(resp.Choices) == 0 {
		return &providers.GenerateResponse{}
	}

	return &providers.GenerateResponse{
		Content: resp.Choices[0].Message.Content,
		Model:   resp.Model,
		Usage: providers.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}
