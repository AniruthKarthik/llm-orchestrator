package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type FetchTool struct{}

func NewFetchTool() *FetchTool { return &FetchTool{} }

func (t *FetchTool) Name() string { return "fetch" }

func (t *FetchTool) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	url, err := requireString(input, "url")
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("fetch: context cancelled: %w", ctx.Err())
	case <-time.After(50 * time.Millisecond):
	}

	body := buildMockBody(url)
	words := len(strings.Fields(body))

	return map[string]any{
		"url":          url,
		"status_code":  200,
		"content_type": "text/html; charset=utf-8",
		"body":         body,
		"word_count":   words,
	}, nil
}

func buildMockBody(url string) string {
	parts := strings.Split(strings.Trim(url, "/"), "/")
	topic := "this topic"
	if len(parts) > 0 {
		topic = strings.ReplaceAll(parts[len(parts)-1], "-", " ")
	}

	return fmt.Sprintf(
		"Introduction to %[1]s. "+
			"%[1]s is a broad subject encompassing many sub-topics. "+
			"This page provides a structured overview including definitions, "+
			"key principles, implementation patterns, and references to further reading. "+
			"Section 1: Core Concepts. The foundational ideas behind %[1]s include "+
			"modularity, observability, and fault tolerance. "+
			"Section 2: Practical Applications. Real-world uses of %[1]s span industries "+
			"ranging from finance to healthcare and e-commerce. "+
			"Section 3: Further Reading. See the official documentation and community "+
			"resources for deeper exploration of %[1]s.",
		topic,
	)
}
