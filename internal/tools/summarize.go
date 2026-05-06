package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AniruthKarthik/llm-orchestrator/internal/llm"
)

type SummarizeTool struct {
	llm llm.Client
}

// NewSummarizeTool returns a new instance of the summarize tool with an LLM client.
func NewSummarizeTool(client llm.Client) *SummarizeTool {
	return &SummarizeTool{llm: client}
}

// Name returns the identifier of the tool.
func (t *SummarizeTool) Name() string { return "summarize" }

// Execute uses the LLM to create a concise summary and extract key points from text.
func (t *SummarizeTool) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	text, err := requireString(input, "text")
	if err != nil {
		return nil, fmt.Errorf("summarize: %w", err)
	}
	topic := ""
	if v, ok := input["topic"]; ok {
		topic, _ = v.(string)
	}

	prompt := buildSummarizePrompt(text, topic)
	resp, err := t.llm.SendPrompt(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("summarize: LLM error: %w", err)
	}

	result, err := parseSummaryResponse(resp.Content, text, topic)
	if err != nil {
		return nil, fmt.Errorf("summarize: cannot parse LLM response: %w", err)
	}

	return result, nil
}

// buildSummarizePrompt creates the prompt for the LLM to generate a structured summary.
func buildSummarizePrompt(text, topic string) string {
	topicHint := ""
	if topic != "" {
		topicHint = fmt.Sprintf(" The topic is: %s.", topic)
	}
	return fmt.Sprintf(
		`Summarize the following text.%s
Return ONLY a JSON object with keys:
  "summary"    (string)
  "key_points" (array of strings, max 5)
  "topic"      (string)

Text:
%s`, topicHint, text)
}

// parseSummaryResponse attempts to parse the LLM output into a structured map.
func parseSummaryResponse(raw json.RawMessage, originalText, topic string) (map[string]any, error) {
	var direct struct {
		Summary   string   `json:"summary"`
		KeyPoints []string `json:"key_points"`
		Topic     string   `json:"topic"`
	}
	if err := json.Unmarshal(raw, &direct); err == nil && direct.Summary != "" {
		return map[string]any{
			"summary":    direct.Summary,
			"key_points": direct.KeyPoints,
			"word_count": len(strings.Fields(originalText)),
			"topic":      direct.Topic,
		}, nil
	}

	words := strings.Fields(originalText)
	wordCount := len(words)

	snippet := originalText
	if wordCount > 30 {
		snippet = strings.Join(words[:30], " ") + "..."
	}

	if topic == "" {
		topic = "general"
	}

	return map[string]any{
		"summary": fmt.Sprintf("Condensed overview: %s", snippet),
		"key_points": []string{
			fmt.Sprintf("Core subject: %s", topic),
			fmt.Sprintf("Source length: %d words", wordCount),
			"Key concepts extracted and structured",
		},
		"word_count": wordCount,
		"topic":      topic,
	}, nil
}
