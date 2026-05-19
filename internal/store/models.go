package store

import (
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

type WorkflowRecord struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      string        `json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
	StartedAt   *time.Time    `json:"startedAt,omitempty"`
	FinishedAt  *time.Time    `json:"finishedAt,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`

	// TaskCount is populated only on list responses (not stored in DB).
	TaskCount int `json:"taskCount,omitempty"`
}

type TaskRecord struct {
	ID         string `json:"id"`
	WorkflowID string `json:"workflowId"`

	Name        string `json:"name"`
	Description string `json:"description"`

	Status string `json:"status"`
	Error  string `json:"error,omitempty"`

	Input  map[string]any `json:"input,omitempty"`
	Output map[string]any `json:"output,omitempty"`

	Dependencies []string `json:"dependencies,omitempty"`

	Timeout          time.Duration     `json:"timeout,omitempty"`
	OutputSchema     map[string]string `json:"outputSchema,omitempty"`
	RetryPolicy      *core.RetryPolicy `json:"retryPolicy,omitempty"`
	Attempt          int               `json:"attempt,omitempty"`
	AgentID          string            `json:"agentId,omitempty"`
	Provider         string            `json:"provider,omitempty"`
	Model            string            `json:"model,omitempty"`
	RequiresApproval bool              `json:"requiresApproval,omitempty"`

	CreatedAt  time.Time  `json:"createdAt"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

type AgentRecord struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Role         string         `json:"role"`
	SystemPrompt string         `json:"systemPrompt,omitempty"`
	Model        string         `json:"model"`
	Provider     string         `json:"provider"`
	Tools        []string       `json:"tools,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
}

type ArtifactRecord struct {
	ID         string         `json:"id"`
	WorkflowID string         `json:"workflowId"`
	TaskID     string         `json:"taskId"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Data       any            `json:"data,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// CheckpointRecord represents a saved state of a workflow execution for recovery.
type CheckpointRecord struct {
	WorkflowID string    `json:"workflowId"`
	StateData  []byte    `json:"stateData"` // JSON serialized workflow and task states
	Timestamp  time.Time `json:"timestamp"`
}
