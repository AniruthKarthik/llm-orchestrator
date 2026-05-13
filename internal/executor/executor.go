package executor

import (
	"context"
	"fmt"
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/dag"
)

type Executor struct {
	registry *WorkerRegistry
}

func NewExecutor(
	registry *WorkerRegistry,
) *Executor {
	return &Executor{
		registry: registry,
	}
}

func (e *Executor) ExecuteWorkflow(
	ctx context.Context,
	workflow *core.Workflow,
	plan *dag.ExecutionPlan,
) error {
	workflow.Start()

	execCtx := NewExecutionContext(workflow.ID)

	for _, stage := range plan.Stages {
		select {
		case <-ctx.Done():
			workflow.Fail()
			return ctx.Err()
		default:
		}

		result := e.executeStage(ctx, workflow, stage, execCtx)
		if !result.Success {
			workflow.Fail()
			if len(result.Errors) > 0 {
				return result.Errors[0]
			}
			return fmt.Errorf("stage %d failed", stage.Level)
		}
	}

	workflow.Complete()

	return nil
}

func (e *Executor) executeStage(
	ctx context.Context,
	workflow *core.Workflow,
	stage dag.ExecutionStage,
	execCtx *ExecutionContext,
) StageResult {
	var wg sync.WaitGroup
	resChan := make(chan TaskResult, len(stage.TaskIDs))

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

			res := e.executeTask(ctx, execCtx, t)
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
	worker, exists := e.registry.Get(task.Name)
	if !exists {
		err := fmt.Errorf("worker not found for task '%s'", task.Name)
		task.Fail(err)
		return TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}
	}

	if err := task.Start(); err != nil {
		return TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}
	}

	// Create a channel to catch the result of worker execution
	type workerResult struct {
		output map[string]any
		err    error
	}
	workerResChan := make(chan workerResult, 1)

	go func() {
		output, err := worker.Execute(ctx, execCtx, task)
		workerResChan <- workerResult{output, err}
	}()

	select {
	case <-ctx.Done():
		task.Fail(ctx.Err())
		return TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   ctx.Err(),
		}
	case res := <-workerResChan:
		if res.err != nil {
			task.Fail(res.err)
			return TaskResult{
				TaskID:  task.ID,
				Success: false,
				Error:   res.err,
			}
		}

		task.Complete(res.output)
		return TaskResult{
			TaskID:  task.ID,
			Success: true,
			Output:  res.output,
		}
	}
}


