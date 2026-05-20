package gemini

import (
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

func mapGenerateRequest(req providers.GenerateRequest) geminiGenerateRequest {
	contents := []geminiContent{}
	var combinedSystemInstruction string

	for _, m := range req.Messages {
		if m.Role == "system" {
			if combinedSystemInstruction != "" {
				combinedSystemInstruction += "\n\n"
			}
			combinedSystemInstruction += m.Content
		} else {
			role := m.Role
			if role == "user" {
				role = "user"
			} else if role == "assistant" {
				role = "model"
			}

			contents = append(contents, geminiContent{
				Role:  role,
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}

	var systemInstruction *geminiInstruction
	if combinedSystemInstruction != "" {
		systemInstruction = &geminiInstruction{
			Parts: []geminiPart{{Text: combinedSystemInstruction}},
		}
	}

	return geminiGenerateRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig: &geminiConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		},
	}
}

func mapGenerateResponse(resp geminiGenerateResponse) *providers.GenerateResponse {
	content := ""
	finishReason := ""
	if len(resp.Candidates) > 0 {
		if len(resp.Candidates[0].Content.Parts) > 0 {
			content = resp.Candidates[0].Content.Parts[0].Text
		}
		finishReason = resp.Candidates[0].FinishReason
	}

	return &providers.GenerateResponse{
		Model:        "gemini", // Response doesn't always include model name in candidates
		Content:      content,
		FinishReason: finishReason,
		Usage: providers.TokenUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}
}
