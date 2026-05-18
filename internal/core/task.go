package core

import (
	"fmt"
	"sync"
	"time"
)

type TaskStatus string

const (
	TaskPending            TaskStatus = "PENDING"
	TaskRunning            TaskStatus = "RUNNING"
	TaskCompleted          TaskStatus = "COMPLETED"
	TaskFailed             TaskStatus = "FAILED"
	TaskWaitingForApproval TaskStatus = "WAITING_FOR_APPROVAL"
)

type Task struct {
	ID          string
	WorkflowID  string
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

	Timeout      time.Duration
	OutputSchema map[string]string // Key: field name, Value: expected type (e.g., "string", "int", "bool")
	AgentID      string            // Optional: ID of the agent assigned to this task

	RequiresApproval bool // If true, task will wait for human approval before finishing or starting

	mu sync.RWMutex
}

func (t *Task) WithAgentID(id string) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.AgentID = id
	return t
}

func NewTask(
	id string,
	workflowID string,
	name string,
	desc string,
	input map[string]any,
	dependencies []string,
) *Task {
	return &Task{
		ID:            id,
		WorkflowID:    workflowID,
		Name:          name,
		Description:   desc,
		Input:         DeepCopyMap(input),
		Dependencies:  DeepCopyStringSlice(dependencies),
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

func (t *Task) WithOutputSchema(schema map[string]string) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.OutputSchema = DeepCopyStringMap(schema)
	return t
}

func (t *Task) SetAttempt(attempt int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Attempt = attempt
}

func (t *Task) ValidateOutput(output map[string]any) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.OutputSchema == nil {
		return nil
	}

	for field, expectedType := range t.OutputSchema {
		val, exists := output[field]
		if !exists {
			return fmt.Errorf("missing required output field: %s", field)
		}

		switch expectedType {
		case "string":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("field %s: expected string, got %T", field, val)
			}
		case "int":
			// Handle different int types that might come from JSON unmarshaling or direct assignment
			switch val.(type) {
			case int, int32, int64, float64:
				// float64 is common if the output came from JSON
			default:
				return fmt.Errorf("field %s: expected int, got %T", field, val)
			}
		case "bool":
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("field %s: expected bool, got %T", field, val)
			}
		case "map":
			if _, ok := val.(map[string]any); !ok {
				return fmt.Errorf("field %s: expected map, got %T", field, val)
			}
		}
	}

	return nil
}

func (t *Task) WithApproval(required bool) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.RequiresApproval = required
	return t
}

func (t *Task) WaitForApproval() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Status != TaskRunning {
		return fmt.Errorf("cannot wait for approval in %s status", t.Status)
	}
	t.Status = TaskWaitingForApproval
	return nil
}

func (t *Task) Approve() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Status != TaskWaitingForApproval {
		return fmt.Errorf("cannot approve task in %s status", t.Status)
	}
	t.Status = TaskRunning
	return nil
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

	t.Output = DeepCopyMap(output)
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

	return DeepCopyMap(t.Output)
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
