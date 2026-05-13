package executor

import (
	"context"
	"fmt"
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/dag"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
)

type Executor struct {
	registry *WorkerRegistry
	eventBus *events.EventBus
}

func NewExecutor(
	registry *WorkerRegistry,
	eventBus *events.EventBus,
) *Executor {
	return &Executor{
		registry: registry,
		eventBus: eventBus,
	}
}

func (e *Executor) ExecuteWorkflow(
	ctx context.Context,
	workflow *core.Workflow,
	plan *dag.ExecutionPlan,
) error {
	if err := workflow.Start(); err != nil {
		return err
	}

	e.eventBus.Publish(events.NewEvent(
		events.WorkflowStarted,
		workflow.ID,
		"",
		nil,
	))

	execCtx := NewExecutionContext(workflow.ID)

	for _, stage := range plan.Stages {
		select {
		case <-ctx.Done():
			_ = workflow.Fail()

			e.eventBus.Publish(events.NewEvent(
				events.WorkflowFailed,
				workflow.ID,
				"",
				map[string]any{
					"error": ctx.Err().Error(),
				},
			))

			return ctx.Err()

		default:
		}

		result := e.executeStage(
			ctx,
			workflow,
			stage,
			execCtx,
		)

		if !result.Success {
			_ = workflow.Fail()

			var err error

			if len(result.Errors) > 0 {
				err = result.Errors[0]
			} else {
				err = fmt.Errorf(
					"stage %d failed",
					stage.Level,
				)
			}

			e.eventBus.Publish(events.NewEvent(
				events.WorkflowFailed,
				workflow.ID,
				"",
				map[string]any{
					"error": err.Error(),
				},
			))

			return err
		}
	}

	if err := workflow.Complete(); err != nil {
		return err
	}

	e.eventBus.Publish(events.NewEvent(
		events.WorkflowCompleted,
		workflow.ID,
		"",
		nil,
	))

	return nil
}

func (e *Executor) executeStage(
	ctx context.Context,
	workflow *core.Workflow,
	stage dag.ExecutionStage,
	execCtx *ExecutionContext,
) StageResult {
	e.eventBus.Publish(events.NewEvent(
		events.StageStarted,
		workflow.ID,
		"",
		map[string]any{
			"stage_level": stage.Level,
		},
	))

	var wg sync.WaitGroup

	resChan := make(
		chan TaskResult,
		len(stage.TaskIDs),
	)

	for _, taskID := range stage.TaskIDs {
		task, err := workflow.GetTask(taskID)

		if err != nil {
			resChan <- TaskResult{
				TaskID:  taskID,
				Success: false,
				Error:   err,
			}

			continue
		}

		wg.Add(1)

		go func(t *core.Task) {
			defer wg.Done()

			res := e.executeTask(
				ctx,
				execCtx,
				t,
			)

			resChan <- res
		}(task)
	}

	wg.Wait()
	close(resChan)

	var results []TaskResult
	var errors []error

	success := true

	for res := range resChan {
		results = append(results, res)

		if !res.Success {
			success = false
			errors = append(errors, res.Error)
		}
	}

	if success {
		e.eventBus.Publish(events.NewEvent(
			events.StageCompleted,
			workflow.ID,
			"",
			map[string]any{
				"stage_level": stage.Level,
			},
		))
	} else {
		e.eventBus.Publish(events.NewEvent(
			events.StageFailed,
			workflow.ID,
			"",
			map[string]any{
				"stage_level": stage.Level,
				"errors":      errors,
			},
		))
	}

	return StageResult{
		Level:   stage.Level,
		Success: success,
		Results: results,
		Errors:  errors,
	}
}

func (e *Executor) executeTask(
	ctx context.Context,
	execCtx *ExecutionContext,
	task *core.Task,
) TaskResult {
	e.eventBus.Publish(events.NewEvent(
		events.TaskStarted,
		execCtx.WorkflowID,
		task.ID,
		nil,
	))

	worker, exists := e.registry.Get(task.Name)

	if !exists {
		err := fmt.Errorf(
			"worker not found for task '%s'",
			task.Name,
		)

		task.Fail(err)

		e.eventBus.Publish(events.NewEvent(
			events.TaskFailed,
			execCtx.WorkflowID,
			task.ID,
			map[string]any{
				"error": err.Error(),
			},
		))

		return TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}
	}

	if err := task.Start(); err != nil {
		e.eventBus.Publish(events.NewEvent(
			events.TaskFailed,
			execCtx.WorkflowID,
			task.ID,
			map[string]any{
				"error": err.Error(),
			},
		))

		return TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}
	}

	type workerResult struct {
		output map[string]any
		err    error
	}

	workerResChan := make(
		chan workerResult,
		1,
	)

	go func() {
		output, err := worker.Execute(
			ctx,
			execCtx,
			task,
		)

		workerResChan <- workerResult{
			output,
			err,
		}
	}()

	select {
	case <-ctx.Done():
		task.Fail(ctx.Err())

		e.eventBus.Publish(events.NewEvent(
			events.TaskFailed,
			execCtx.WorkflowID,
			task.ID,
			map[string]any{
				"error": ctx.Err().Error(),
			},
		))

		return TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   ctx.Err(),
		}

	case res := <-workerResChan:
		if res.err != nil {
			task.Fail(res.err)

			e.eventBus.Publish(events.NewEvent(
				events.TaskFailed,
				execCtx.WorkflowID,
				task.ID,
				map[string]any{
					"error": res.err.Error(),
				},
			))

			return TaskResult{
				TaskID:  task.ID,
				Success: false,
				Error:   res.err,
			}
		}

		task.Complete(res.output)

		e.eventBus.Publish(events.NewEvent(
			events.TaskCompleted,
			execCtx.WorkflowID,
			task.ID,
			map[string]any{
				"output": res.output,
			},
		))

		return TaskResult{
			TaskID:  task.ID,
			Success: true,
			Output:  res.output,
		}
	}
}
