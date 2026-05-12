package core

import (
	"errors"
	"time"
)

type WorkflowStatus string

const (
	WorkflowPending   WorkflowStatus = "PENDING"
	WorkflowRunning   WorkflowStatus = "RUNNING"
	WorkflowCompleted WorkflowStatus = "COMPLETED"
	WorkflowFailed    WorkflowStatus = "FAILED"
)

type Workflow struct {
	ID          string
	Name        string
	Description string

	Tasks  map[string]*Task
	Status WorkflowStatus

	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

func NewWorkflow(
	id string,
	name string,
	description string,
) *Workflow {
	newWorkflow := &Workflow{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      WorkflowPending,
		CreatedAt:   time.Now(),
		Tasks:       map[string]*Task{},
	}

	return newWorkflow
}

func (w *Workflow) AddTask(task *Task) error {
	if task == nil {
		return errors.New("task is nil")
	}

	if _, exists := w.Tasks[task.ID]; exists {
		return errors.New("task already exists")
	}

	w.Tasks[task.ID] = task

	return nil
}

func (w *Workflow) GetTask(id string) (*Task, error) {
	task, exists := w.Tasks[id]
	if !exists {
		return nil, errors.New("task not found")
	}

	return task, nil
}

func (w *Workflow) Start() {
	w.Status = WorkflowPending
	t := time.Now()
	w.StartedAt = &t
}

func (w *Workflow) Complete() {
	w.Status = WorkflowCompleted
	t := time.Now()
	w.FinishedAt = &t
}

func (w *Workflow) Fail() {
	w.Status = WorkflowFailed
	t := time.Now()
	w.FinishedAt = &t
}

func (w *Workflow) ReadyTasks() []*Task {
	var ready []*Task

	completed := make(map[string]bool)

	for _, task := range w.Tasks {
		if task.Status == TaskCompleted {
			completed[task.ID] = true
		}
	}

	for _, task := range w.Tasks {
		if task.Status == TaskPending {
			continue
		}

		if task.CanRun(completed) {
			ready = append(ready, task)
		}
	}

	return ready
}

func (w *Workflow) AllTasksFinished() bool {
	for _, task := range w.Tasks {
		if task.Status != TaskCompleted {
			return false
		}
	}

	return true
}

func (w *Workflow) HasFailures() bool {
	for _, task := range w.Tasks {
		if task.Status == TaskFailed {
			return true
		}
	}

	return false
}
