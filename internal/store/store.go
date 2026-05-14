package store

import (
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

type Store interface {
	SaveWorkflow(
		workflow WorkflowRecord,
	) error

	UpdateWorkflow(
		workflow WorkflowRecord,
	) error

	GetWorkflow(
		workflowID string,
	) (WorkflowRecord, error)

	SaveTask(
		task TaskRecord,
	) error

	UpdateTask(
		task TaskRecord,
	) error

	GetTask(
		workflowID string,
		taskID string,
	) (TaskRecord, error)

	GetWorkflowTasks(
		workflowID string,
	) ([]TaskRecord, error)

	SaveCheckpoint(
		checkpoint CheckpointRecord,
	) error

	GetLatestCheckpoint(
		workflowID string,
	) (CheckpointRecord, error)
}

func WorkflowToRecord(
	workflow *core.Workflow,
) WorkflowRecord {
	if workflow == nil {
		return WorkflowRecord{}
	}

	return WorkflowRecord{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Status:      string(workflow.Status),
		CreatedAt:   workflow.CreatedAt,
		FinishedAt:  workflow.FinishedAt,
		StartedAt:   workflow.StartedAt,
	}
}

func TaskToRecord(
	workflowID string,
	task *core.Task,
) TaskRecord {
	dependencies := make([]string, len(task.Dependencies))
	copy(dependencies, task.Dependencies)

	input := make(map[string]any)
	for k, v := range task.Input {
		input[k] = v
	}

	output := make(map[string]any)
	for k, v := range task.Output {
		output[k] = v
	}

	return TaskRecord{
		ID:           task.ID,
		WorkflowID:   workflowID,
		Name:         task.Name,
		Description:  task.Description,
		Status:       string(task.Status),
		Error:        task.Error,
		Input:        input,
		Output:       output,
		Dependencies: dependencies,
		CreatedAt:    task.CreatedAt,
		StartedAt:    task.StartedAt,
		FinishedAt:   task.FinishedAt,
	}
}

func RecordToWorkflow(
	workflow WorkflowRecord,
	tasks []TaskRecord,
) *core.Workflow {
	runtimeTasks := make(map[string]*core.Task)

	for _, task := range tasks {
		dependencies := make([]string, len(task.Dependencies))
		copy(dependencies, task.Dependencies)

		input := make(map[string]any)
		for k, v := range task.Input {
			input[k] = v
		}

		output := make(map[string]any)
		for k, v := range task.Output {
			output[k] = v
		}

		runtimeTasks[task.ID] = &core.Task{
			ID:           task.ID,
			Name:         task.Name,
			Description:  task.Description,
			Status:       core.TaskStatus(task.Status),
			Error:        task.Error,
			Input:        input,
			Output:       output,
			Dependencies: dependencies,
			CreatedAt:    task.CreatedAt,
			StartedAt:    task.StartedAt,
			FinishedAt:   task.FinishedAt,
		}
	}

	return &core.Workflow{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Status:      core.WorkflowStatus(workflow.Status),
		Tasks:       runtimeTasks,
		CreatedAt:   workflow.CreatedAt,
		StartedAt:   workflow.StartedAt,
		FinishedAt:  workflow.FinishedAt,
	}
}
