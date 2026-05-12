package core

import "time"

type TaskStatus string

const (
	TaskPending   TaskStatus = "PENDING"
	TaskRunning   TaskStatus = "RUNNING"
	TaskCompleted TaskStatus = "COMPLETED"
	TaskFailed    TaskStatus = "FAILED"
)

type Task struct {
	ID          string
	Name        string
	Description string
	Input       map[string]any
	Output      map[string]any
	Status      TaskStatus
	Error       string
	CreatedAt   time.Time
	StartedAt   time.Time
	FinishedAt  time.Time

	Dependencies []string
}

func NewTask(
	id string,
	name string,
	desc string,
	input map[string]any,
	dependencies []string,
) *Task {
	newtask := &Task{
		ID:           id,
		Name:         name,
		Description:  desc,
		Input:        input,
		Dependencies: dependencies,
		Status:       TaskPending,
		Error:        "",
		CreatedAt:    time.Now(),
	}

	return newtask
}

func (t *Task) Start() {
	t.Status = TaskRunning
	t.StartedAt = time.Now()
}

func (t *Task) Complete(output map[string]any) {
	t.Output = output
	t.Status = TaskCompleted
	t.FinishedAt = time.Now()
}

func (t *Task) Fail(err error) {
	t.Error = "task failed"
	t.Status = TaskFailed
	t.FinishedAt = time.Now()
}

func (t *Task) IsFinished() bool {
	if t.Status == TaskCompleted {
		return true
	}
	return false
}

func (t *Task) CanRun(completed map[string]bool) bool {
	for _, depId := range t.Dependencies {
		if !completed[depId] {
			return false
		}
	}

	return true
}
