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
}
