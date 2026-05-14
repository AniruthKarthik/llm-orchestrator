package core

import (
	"fmt"
	"sync"
	"time"
)

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
	StartedAt   *time.Time
	FinishedAt  *time.Time

	Dependencies []string

	RetryPolicy   *RetryPolicy
	FailurePolicy FailurePolicy
	Attempt       int

	mu sync.RWMutex
}

func NewTask(
	id string,
	name string,
	desc string,
	input map[string]any,
	dependencies []string,
) *Task {
	// Copy input map to avoid external mutation
	inputCopy := make(map[string]any)
	if input != nil {
		for k, v := range input {
			inputCopy[k] = v
		}
	}

	// Copy dependencies slice to avoid external mutation
	depsCopy := make([]string, 0)
	if dependencies != nil {
		depsCopy = append(depsCopy, dependencies...)
	}

	return &Task{
		ID:            id,
		Name:          name,
		Description:   desc,
		Input:         inputCopy,
		Dependencies:  depsCopy,
		Status:        TaskPending,
		Error:         "",
		CreatedAt:     time.Now(),
		Output:        make(map[string]any),
		FailurePolicy: FailurePolicyFailFast,
	}
}

func (t *Task) WithRetryPolicy(p RetryPolicy) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.RetryPolicy = &p
	return t
}

func (t *Task) WithFailurePolicy(p FailurePolicy) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.FailurePolicy = p
	return t
}

func (t *Task) GetFailurePolicy() FailurePolicy {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.FailurePolicy
}

func (t *Task) SetAttempt(attempt int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Attempt = attempt
}

func (t *Task) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Status != TaskPending {
		return fmt.Errorf("cannot start task in %s status", t.Status)
	}
	t.Status = TaskRunning
	now := time.Now()
	t.StartedAt = &now
	return nil
}

func (t *Task) Complete(output map[string]any) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Status != TaskRunning {
		return fmt.Errorf("cannot complete task in %s status", t.Status)
	}

	// Copy output map to avoid external mutation
	t.Output = make(map[string]any)
	if output != nil {
		for k, v := range output {
			t.Output[k] = v
		}
	}

	t.Status = TaskCompleted
	now := time.Now()
	t.FinishedAt = &now
	return nil
}

func (t *Task) Fail(err error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Status == TaskCompleted || t.Status == TaskFailed {
		return fmt.Errorf("cannot fail task in %s status", t.Status)
	}
	if err != nil {
		t.Error = err.Error()
	} else {
		t.Error = "unknown error"
	}
	t.Status = TaskFailed
	now := time.Now()
	t.FinishedAt = &now
	return nil
}

func (t *Task) IsFinished() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status == TaskCompleted || t.Status == TaskFailed
}

func (t *Task) CanRun(completed map[string]bool) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, depId := range t.Dependencies {
		if !completed[depId] {
			return false
		}
	}

	return true
}

func (t *Task) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

func (t *Task) GetOutput() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	outputCopy := make(map[string]any)
	for k, v := range t.Output {
		outputCopy[k] = v
	}
	return outputCopy
}

func (t *Task) GetError() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Error
}

func (t *Task) GetStartedAt() *time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.StartedAt == nil {
		return nil
	}
	cp := *t.StartedAt
	return &cp
}

func (t *Task) GetFinishedAt() *time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.FinishedAt == nil {
		return nil
	}
	cp := *t.FinishedAt
	return &cp
}
