package core

import (
	"errors"
	"sync"
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

	mu sync.RWMutex
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
	w.mu.Lock()
	defer w.mu.Unlock()

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
	w.mu.RLock()
	defer w.mu.RUnlock()

	task, exists := w.Tasks[id]
	if !exists {
		return nil, errors.New("task not found")
	}

	return task, nil
}

func (w *Workflow) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Status != WorkflowPending {
		return errors.New("cannot start workflow that is not pending")
	}

	w.Status = WorkflowRunning
	t := time.Now()
	w.StartedAt = &t
	return nil
}

func (w *Workflow) Complete() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Status != WorkflowRunning {
		return errors.New("cannot complete workflow that is not running")
	}

	w.Status = WorkflowCompleted
	t := time.Now()
	w.FinishedAt = &t
	return nil
}

func (w *Workflow) Fail() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Status == WorkflowCompleted || w.Status == WorkflowFailed {
		return errors.New("cannot fail workflow that is already finished")
	}

	w.Status = WorkflowFailed
	t := time.Now()
	w.FinishedAt = &t
	return nil
}

func (w *Workflow) ReadyTasks() []*Task {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var ready []*Task

	completed := make(map[string]bool)

	for _, task := range w.Tasks {
		if task.GetStatus() == TaskCompleted {
			completed[task.ID] = true
		}
	}

	for _, task := range w.Tasks {
		if task.GetStatus() != TaskPending {
			continue
		}

		if task.CanRun(completed) {
			ready = append(ready, task)
		}
	}

	return ready
}

func (w *Workflow) AllTasksFinished() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, task := range w.Tasks {
		if !task.IsFinished() {
			return false
		}
	}

	return true
}

func (w *Workflow) HasFailures() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, task := range w.Tasks {
		if task.GetStatus() == TaskFailed {
			return true
		}
	}

	return false
}

func (w *Workflow) GetTasks() map[string]*Task {
	w.mu.RLock()
	defer w.mu.RUnlock()

	tasks := make(map[string]*Task, len(w.Tasks))
	for k, v := range w.Tasks {
		tasks[k] = v
	}

	return tasks
}
