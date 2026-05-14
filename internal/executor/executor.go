package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/dag"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

type Executor struct {
	registry *WorkerRegistry
	eventBus *events.EventBus
	store    store.Store
}

func NewExecutor(
	registry *WorkerRegistry,
	eventBus *events.EventBus,
	store store.Store,
) *Executor {
	return &Executor{
		registry: registry,
		eventBus: eventBus,
		store:    store,
	}
}

func (e *Executor) Execute(
	workflow *core.Workflow,
) error {
	if err := e.store.SaveWorkflow(
		store.WorkflowToRecord(workflow),
	); err != nil {
		return err
	}

	for _, task := range workflow.GetTasks() {
		if err := e.store.SaveTask(
			store.TaskToRecord(workflow.ID, task),
		); err != nil {
			return err
		}
	}

	if err := workflow.Start(); err != nil {
		return err
	}

	if err := e.store.UpdateWorkflow(
		store.WorkflowToRecord(workflow),
	); err != nil {
		return err
	}

	planner := dag.NewTopologicalPlanner()
	plan, err := planner.BuildExecutionPlan(workflow)
	if err != nil {
		return err
	}

	execCtx := NewExecutionContext(workflow.ID)

	for _, stage := range plan.Stages {
		var wg sync.WaitGroup
		errChan := make(chan error, len(stage.TaskIDs))

		for _, taskID := range stage.TaskIDs {
			task, err := workflow.GetTask(taskID)
			if err != nil {
				return err
			}

			wg.Add(1)
			go func(t *core.Task) {
				defer wg.Done()
				if err := e.executeTask(context.Background(), execCtx, workflow, t); err != nil {
					errChan <- err
				}
			}(task)
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			if err != nil {
				if failErr := workflow.Fail(); failErr != nil {
					return fmt.Errorf("failed to fail workflow: %v (original error: %v)", failErr, err)
				}

				_ = e.store.UpdateWorkflow(
					store.WorkflowToRecord(workflow),
				)

				return err
			}
		}
	}

	if err := workflow.Complete(); err != nil {
		return err
	}

	if err := e.store.UpdateWorkflow(
		store.WorkflowToRecord(workflow),
	); err != nil {
		return err
	}

	return nil
}

func (e *Executor) executeTask(
	ctx context.Context,
	execCtx *ExecutionContext,
	workflow *core.Workflow,
	task *core.Task,
) error {
	worker, exists := e.registry.Get(task.Name)
	if !exists {
		err := fmt.Errorf("worker not found: %s", task.Name)
		_ = task.Fail(err)
		_ = e.store.UpdateTask(store.TaskToRecord(workflow.ID, task))
		return err
	}

	if err := task.Start(); err != nil {
		return err
	}

	if err := e.store.UpdateTask(
		store.TaskToRecord(workflow.ID, task),
	); err != nil {
		return err
	}

	e.eventBus.Publish(events.Event{
		Type:       events.TaskStarted,
		TaskID:     task.ID,
		WorkflowID: workflow.ID,
		Timestamp:  time.Now(),
	})

	output, err := worker.Execute(ctx, execCtx, task)
	if err != nil {
		_ = task.Fail(err)
		_ = e.store.UpdateTask(
			store.TaskToRecord(workflow.ID, task),
		)

		e.eventBus.Publish(events.Event{
			Type:       events.TaskFailed,
			TaskID:     task.ID,
			WorkflowID: workflow.ID,
			Timestamp:  time.Now(),
			Payload: map[string]any{
				"error": err.Error(),
			},
		})
		return err
	}

	if err := task.Complete(output); err != nil {
		return err
	}

	if err := e.store.UpdateTask(
		store.TaskToRecord(workflow.ID, task),
	); err != nil {
		return err
	}

	e.eventBus.Publish(events.Event{
		Type:       events.TaskCompleted,
		TaskID:     task.ID,
		WorkflowID: workflow.ID,
		Timestamp:  time.Now(),
	})

	return nil
}
