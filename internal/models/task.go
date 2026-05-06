package models

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusWaiting   TaskStatus = "waiting"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID        string     `json:"id"`
	JobID     string     `json:"job_id"`
	Type      string     `json:"type"`
	Payload   any        `json:"payload"`
	Status    TaskStatus `json:"status"`
	DependsOn []string   `json:"depends_on"`
	Result    any        `json:"result,omitempty"`
	Retries   int        `json:"retries"`
}
