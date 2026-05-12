package core

import (
	"errors"
	"fmt"
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
	if input == nil {
		input = make(map[string]any)
	}
	if dependencies == nil {
		dependencies = []string{}
	}
	newtask := &Task{
		ID:           id,
		Name:         name,
		Description:  desc,
		Input:        input,
		Dependencies: dependencies,
		Status:       TaskPending,
		Error:        "",
		CreatedAt:    time.Now(),
		Output:       make(map[string]any),
	}

	return newtask
}

func (t *Task) Start() error {
	if t.Status != TaskPending {
		return fmt.Errorf("cannot start task in %s status", t.Status)
	}
	t.Status = TaskRunning
	t.StartedAt = time.Now()
	return nil
}

func (t *Task) Complete(output map[string]any) error {
	if t.Status != TaskRunning {
		return fmt.Errorf("cannot complete task in %s status", t.Status)
	}
	if output == nil {
		output = make(map[string]any)
	}
	t.Output = output
	t.Status = TaskCompleted
	t.FinishedAt = time.Now()
	return nil
}

func (t *Task) Fail(err error) error {
	if t.Status == TaskCompleted || t.Status == TaskFailed {
		return fmt.Errorf("cannot fail task in %s status", t.Status)
	}
	if err != nil {
		t.Error = err.Error()
	} else {
		t.Error = "unknown error"
	}
	t.Status = TaskFailed
	t.FinishedAt = time.Now()
	return nil
}

func (t *Task) IsFinished() bool {
	return t.Status == TaskCompleted || t.Status == TaskFailed
}

func (t *Task) CanRun(completed map[string]bool) bool {
	for _, depId := range t.Dependencies {
		if !completed[depId] {
			return false
		}
	}

	return true
}
