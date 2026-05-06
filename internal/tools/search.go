package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type SearchTool struct{}

func NewSearchTool() *SearchTool { return &SearchTool{} }

func (t *SearchTool) Name() string { return "search" }

func (t *SearchTool) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	query, err := requireString(input, "query")
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("search: context cancelled: %w", ctx.Err())
	case <-time.After(30 * time.Millisecond):
	}

	results := buildMockSearchResults(query)

	return map[string]any{
		"query":   query,
		"results": results,
		"total":   len(results),
	}, nil
}

func buildMockSearchResults(query string) []map[string]any {
	keywords := strings.Fields(strings.ToLower(query))
	topic := "general"
	if len(keywords) > 0 {
		topic = keywords[0]
	}

	return []map[string]any{
		{
			"title":   fmt.Sprintf("Comprehensive guide to %s", topic),
			"url":     fmt.Sprintf("https://docs.example.com/%s/overview", topic),
			"snippet": fmt.Sprintf("An in-depth overview of %s covering key concepts, best practices, and real-world use cases.", topic),
			"rank":    1,
		},
		{
			"title":   fmt.Sprintf("%s: getting started", topic),
			"url":     fmt.Sprintf("https://wiki.example.com/%s/quickstart", topic),
			"snippet": fmt.Sprintf("Step-by-step quickstart guide for %s with code examples.", topic),
			"rank":    2,
		},
		{
			"title":   fmt.Sprintf("Latest %s research and trends", topic),
			"url":     fmt.Sprintf("https://news.example.com/%s", topic),
			"snippet": fmt.Sprintf("Recent developments and emerging trends in %s as of 2025.", topic),
			"rank":    3,
		},
	}
}

func requireString(input map[string]any, key string) (string, error) {
	v, ok := input[key]
	if !ok {
		return "", fmt.Errorf("missing required field %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("field %q must be a string, got %T", key, v)
	}
	if strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("field %q must not be empty", key)
	}
	return s, nil
}
