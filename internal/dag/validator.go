package dag

import (
	"fmt"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// Validate performs all validation checks on a workflow
func (v *Validator) Validate(workflow *core.Workflow) error {
	tasks := workflow.GetTasks()
	if err := v.validateTaskReferences(tasks); err != nil {
		return err
	}
	if err := v.validateSelfDependencies(tasks); err != nil {
		return err
	}
	if err := v.validateCycles(tasks); err != nil {
		return err
	}
	return nil
}

func (v *Validator) validateTaskReferences(
	tasks map[string]*core.Task,
) error {
	for id, task := range tasks {
		for _, dep := range task.Dependencies {
			if _, exists := tasks[dep]; !exists {
				return fmt.Errorf(
					"task '%s' references unknown dependency '%s': %w",
					id,
					dep,
					ErrTaskNotFound,
				)
			}
		}
	}

	return nil
}

func (v *Validator) validateSelfDependencies(
	tasks map[string]*core.Task,
) error {
	for id, task := range tasks {
		for _, dep := range task.Dependencies {
			if dep == id {
				return fmt.Errorf(
					"task '%s' cannot depend on itself: %w",
					id,
					ErrSelfDependency,
				)
			}
		}
	}

	return nil
}

func (v *Validator) validateCycles(
	tasks map[string]*core.Task,
) error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for id := range tasks {
		if !visited[id] {
			if v.detectCycleDFS(
				id,
				tasks,
				visited,
				recursionStack,
			) {
				return fmt.Errorf(
					"cycle detected involving task '%s': %w",
					id,
					ErrCircularDependency,
				)
			}
		}
	}

	return nil
}

func (v *Validator) detectCycleDFS(
	taskID string,
	tasks map[string]*core.Task,
	visited map[string]bool,
	recursionStack map[string]bool,
) bool {
	if recursionStack[taskID] {
		return true
	}

	if visited[taskID] {
		return false
	}

	visited[taskID] = true
	recursionStack[taskID] = true

	task := tasks[taskID]
	if task == nil {
		return false
	}

	for _, dep := range task.Dependencies {
		if v.detectCycleDFS(
			dep,
			tasks,
			visited,
			recursionStack,
		) {
			return true
		}
	}

	recursionStack[taskID] = false

	return false
}
