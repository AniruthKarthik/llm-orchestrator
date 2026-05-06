package orchestrator

import (
	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
)

type EventType string

const (
	EventTaskCompleted EventType = "task_completed"
	EventTaskFailed    EventType = "task_failed"
	EventJobQueued     EventType = "job_queued"
)

type Event struct {
	Type   EventType
	JobID  string
	TaskID string
}

type ReadyTask struct {
	Task  models.Task
	JobID string
}

type jobPhase int

const (
	phaseRunning jobPhase = iota
	phaseCompleted
	phaseFailed
)
