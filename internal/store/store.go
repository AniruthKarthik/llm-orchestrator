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

	ListWorkflows() ([]WorkflowRecord, error)
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
		Timeout:     workflow.Timeout,
	}
}

func TaskToRecord(
	workflowID string,
	task *core.Task,
) TaskRecord {
	return TaskRecord{
		ID:           task.ID,
		WorkflowID:   workflowID,
		Name:         task.Name,
		Description:  task.Description,
		Status:       string(task.Status),
		Error:        task.Error,
		Input:        core.DeepCopyMap(task.Input),
		Output:       core.DeepCopyMap(task.Output),
		Dependencies: core.DeepCopyStringSlice(task.Dependencies),
		CreatedAt:    task.CreatedAt,
		StartedAt:    task.StartedAt,
		FinishedAt:   task.FinishedAt,
		Timeout:      task.Timeout,
	}
}

func RecordToWorkflow(
	workflow WorkflowRecord,
	tasks []TaskRecord,
) *core.Workflow {
	runtimeTasks := make(map[string]*core.Task)

	for _, task := range tasks {
		runtimeTasks[task.ID] = &core.Task{
			ID:           task.ID,
			Name:         task.Name,
			Description:  task.Description,
			Status:       core.TaskStatus(task.Status),
			Error:        task.Error,
			Input:        core.DeepCopyMap(task.Input),
			Output:       core.DeepCopyMap(task.Output),
			Dependencies: core.DeepCopyStringSlice(task.Dependencies),
			CreatedAt:    task.CreatedAt,
			StartedAt:    task.StartedAt,
			FinishedAt:   task.FinishedAt,
			Timeout:      task.Timeout,
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
		Timeout:     workflow.Timeout,
	}
}
