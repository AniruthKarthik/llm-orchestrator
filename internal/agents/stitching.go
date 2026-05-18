package agents

import (
	"fmt"
	"strings"
)

// ContextStitcher handles data that might exceed context window limits.
type ContextStitcher struct {
	MaxChars int
}

func NewContextStitcher(maxChars int) *ContextStitcher {
	return &ContextStitcher{
		MaxChars: maxChars,
	}
}

// Stitch simplifies or truncates content if it exceeds the limit.
func (s *ContextStitcher) Stitch(content string) string {
	if len(content) <= s.MaxChars {
		return content
	}

	// Simple truncation with a note for now.
	// In a real implementation, we could use an LLM to summarize.
	return fmt.Sprintf("%s\n\n[CONTENT TRUNCATED DUE TO CONTEXT LIMITS]", content[:s.MaxChars])
}

// StitchArtifacts combines artifacts into a single string, respecting limits.
func (s *ContextStitcher) StitchArtifacts(artifacts []string) string {
	var sb strings.Builder
	currentLen := 0

	for _, a := range artifacts {
		if currentLen+len(a) > s.MaxChars {
			sb.WriteString("\n[FURTHER ARTIFACTS OMITTED DUE TO CONTEXT LIMITS]")
			break
		}
		sb.WriteString(a)
		sb.WriteString("\n")
		currentLen += len(a) + 1
	}

	return sb.String()
}
