package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
)

const (
	maxRetries     = 3
	defaultTimeout = 30 * time.Second
)

type Response struct {
	Content json.RawMessage `json:"content"`
}

type Client interface {
	SendPrompt(ctx context.Context, prompt string) (*Response, error)
}

type MockClient struct {
	Timeout time.Duration
}

// NewMockClient returns a client that simulates LLM responses for testing.
func NewMockClient() *MockClient {
	return &MockClient{Timeout: defaultTimeout}
}

// SendPrompt simulates sending a prompt to an LLM with retries and backoff.
func (c *MockClient) SendPrompt(ctx context.Context, prompt string) (*Response, error) {
	var lasterr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[llm] attempt %d/%d ", attempt, maxRetries)

		attemptCtx, cancel := context.WithTimeout(ctx, c.Timeout)
		raw, err := c.callOnce(attemptCtx, prompt)
		cancel()

		if err != nil {
			lasterr = err
			log.Printf("[llm] attempt %d failed: %v", attempt, err)

			if ctx.Err() != nil {
				return nil, fmt.Errorf("context cancelled before LLM response: %w", ctx.Err())
			}

			backoff := time.Duration(attempt*attempt) * 200 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during back-off: %w", ctx.Err())
			case <-time.After(backoff):
			}
			continue
		}

		if !json.Valid(raw) {
			lasterr = fmt.Errorf("LLM returned non JSON payload on attempt %d", attempt)
			log.Printf("[llm] %v", lasterr)
			continue
		}

		return &Response{Content: raw}, nil
	}

	return nil, fmt.Errorf("LLM client exhausted %d retries, last error: %w", maxRetries, lasterr)
}

// callOnce performs a single simulated LLM call with random latency.
func (c *MockClient) callOnce(ctx context.Context, _ string) ([]byte, error) {
	latency := time.Duration(50+rand.Intn(150)) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for LLM: %w", ctx.Err())
	case <-time.After(latency):
	}
	payload := `[
		{
			"id": "task-001",
			"type": "research",
			"payload": {"query": "gather requirements"},
			"status": "pending",
			"depends_on": [],
			"result": null,
			"retries": 0
		},
		{
			"id": "task-002",
			"type": "generate",
			"payload": {"template": "outline"},
			"status": "pending",
			"depends_on": ["task-001"],
			"result": null,
			"retries": 0
		},
		{
			"id": "task-003",
			"type": "validate",
			"payload": {"checks": ["schema", "completeness"]},
			"status": "pending",
			"depends_on": ["task-002"],
			"result": null,
			"retries": 0
		}
	]`

	return []byte(payload), nil
}
