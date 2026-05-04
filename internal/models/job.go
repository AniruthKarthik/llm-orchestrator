package models

import "time"

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusPlanning  JobStatus = "planning"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

type Job struct {
	ID        string    `json:"id"`
	Goal      string    `json:"goal"`
	Status    JobStatus `json:"status"`
	Tasks     []Task    `json:"tasks"`
	Result    any       `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
