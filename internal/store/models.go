package store

import (
	"time"
)

type WorkflowRecord struct {
	ID          string
	Name        string
	Description string

	Status string

	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time

	Timeout time.Duration
}

type TaskRecord struct {
	ID         string
	WorkflowID string

	Name        string
	Description string

	Status string
	Error  string

	Input  map[string]any
	Output map[string]any

	Dependencies []string

	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time

	Timeout time.Duration
}

// CheckpointRecord represents a saved state of a workflow execution for recovery.
type CheckpointRecord struct {
	WorkflowID string
	StateData  []byte // JSON serialized workflow and task states
	Timestamp  time.Time
}
