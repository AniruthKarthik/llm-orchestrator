package models

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Payload   any        `json:"payload"`
	Status    TaskStatus `json:"status"`
	DependsOn []string   `json:"depends_on"`
	Result    any        `json:"result,omitempty"`
	Retries   int        `json:"retries"`
}
