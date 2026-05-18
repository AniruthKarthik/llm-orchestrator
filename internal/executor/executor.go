package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/dag"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/plugin"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
)

type Executor struct {
	registry         *WorkerRegistry
	agentRegistry    *agents.AgentRegistry
	artifactRegistry *core.ArtifactRegistry
	memoryRegistry   *core.MemoryRegistry
	toolRegistry     *core.ToolRegistry
	toolPolicy       *core.ToolPolicy
	pluginRegistry   plugin.Registry
	eventBus         *events.EventBus
	store            store.Store
	router           Router

	taskMiddlewares     []TaskMiddleware
	workflowMiddlewares []WorkflowMiddleware

	concurrencyLimiter *ConcurrencyLimiter
	mu                 sync.RWMutex
}

func NewExecutor(
	registry *WorkerRegistry,
	agentRegistry *agents.AgentRegistry,
	artifactRegistry *core.ArtifactRegistry,
	memoryRegistry *core.MemoryRegistry,
	toolRegistry *core.ToolRegistry,
	toolPolicy *core.ToolPolicy,
	eventBus *events.EventBus,
	store store.Store,
) *Executor {
	return &Executor{
		registry:            registry,
		agentRegistry:       agentRegistry,
		artifactRegistry:    artifactRegistry,
		memoryRegistry:      memoryRegistry,
		toolRegistry:        toolRegistry,
		toolPolicy:          toolPolicy,
		pluginRegistry:      plugin.NewDefaultRegistry(),
		eventBus:            eventBus,
		store:               store,
		router:              NewCapabilityAwareRouter(),
		taskMiddlewares:     make([]TaskMiddleware, 0),
		workflowMiddlewares: make([]WorkflowMiddleware, 0),
		concurrencyLimiter:  NewConcurrencyLimiter(100), // Default limit
	}
}

func (e *Executor) WithConcurrencyLimit(limit int) *Executor {
	e.concurrencyLimiter = NewConcurrencyLimiter(limit)
	return e
}

func (e *Executor) UseTaskMiddleware(m ...TaskMiddleware) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.taskMiddlewares = append(e.taskMiddlewares, m...)
}

func (e *Executor) UseWorkflowMiddleware(m ...WorkflowMiddleware) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workflowMiddlewares = append(e.workflowMiddlewares, m...)
}

func (e *Executor) WithPluginRegistry(r plugin.Registry) *Executor {
	e.pluginRegistry = r
	return e
}

func (e *Executor) WithRouter(r Router) *Executor {
	e.router = r
	return e
}

func (e *Executor) Execute(
	workflow *core.Workflow,
) error {
	e.mu.RLock()
	middlewares := e.workflowMiddlewares
	e.mu.RUnlock()
	handler := ApplyWorkflowMiddleware(e.execute, middlewares...)
	return handler(workflow)
}

func (e *Executor) execute(
	workflow *core.Workflow,
) error {
	// Use a workflow-level context that we can cancel if any task fails critically
	var (
		workflowCtx    context.Context
		workflowCancel context.CancelFunc
	)

	if workflow.Timeout > 0 {
		workflowCtx, workflowCancel = context.WithTimeout(context.Background(), workflow.Timeout)
	} else {
		workflowCtx, workflowCancel = context.WithCancel(context.Background())
	}
	defer workflowCancel()

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

	execCtx := NewExecutionContext(workflow.ID, e.artifactRegistry, e.memoryRegistry, e.toolRegistry, e.toolPolicy)

	for _, stage := range plan.Stages {
		// Check if workflow was cancelled by a previous stage failure
		select {
		case <-workflowCtx.Done():
			return workflowCtx.Err()
		default:
		}

		var wg sync.WaitGroup
		errChan := make(chan error, len(stage.TaskIDs))

		for _, taskID := range stage.TaskIDs {
			task, err := workflow.GetTask(taskID)
			if err != nil {
				workflowCancel()
				return err
			}

			if task.Status == core.TaskCompleted {
				continue
			}

			wg.Add(1)
			go func(t *core.Task) {
				defer wg.Done()

				// Apply task-specific timeout if set
				taskCtx := workflowCtx
				var taskCancel context.CancelFunc
				if t.Timeout > 0 {
					taskCtx, taskCancel = context.WithTimeout(workflowCtx, t.Timeout)
					defer taskCancel()
				}

				if err := e.executeTask(taskCtx, execCtx, workflow, t); err != nil {
					errChan <- err
					// If we should fail fast, cancel the entire workflow
					if workflow.GetFailurePolicy() != core.FailurePolicyContinueOnFailure {
						workflowCancel()
					}
				}
			}(task)
		}

		wg.Wait()
		close(errChan)

		var stageErr error
		for err := range errChan {
			if err != nil {
				stageErr = err
				break // We just need the first error to decide what to do
			}
		}

		if stageErr != nil {
			wfFailurePolicy := workflow.GetFailurePolicy()
			if wfFailurePolicy == core.FailurePolicyContinueOnFailure {
				continue
			}

			if failErr := workflow.Fail(); failErr != nil {
				return fmt.Errorf("failed to fail workflow: %v (original error: %v)", failErr, stageErr)
			}

			_ = e.store.UpdateWorkflow(
				store.WorkflowToRecord(workflow),
			)

			return stageErr
		}

		// Create a checkpoint after each successful stage
		if cpData, err := workflow.CreateCheckpoint(); err == nil {
			_ = e.store.SaveCheckpoint(store.CheckpointRecord{
				WorkflowID: workflow.ID,
				StateData:  cpData,
				Timestamp:  time.Now(),
			})
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

func (e *Executor) Resume(
	workflowID string,
) error {
	record, err := e.store.GetWorkflow(workflowID)
	if err != nil {
		return err
	}

	taskRecords, err := e.store.GetWorkflowTasks(workflowID)
	if err != nil {
		return err
	}

	workflow := store.RecordToWorkflow(record, taskRecords)

	checkpoint, err := e.store.GetLatestCheckpoint(workflowID)
	if err == nil {
		if err := workflow.RestoreFromCheckpoint(checkpoint.StateData); err != nil {
			return fmt.Errorf("failed to restore from checkpoint: %w", err)
		}
	}

	return e.Execute(workflow)
}

func (e *Executor) executeTask(
	ctx context.Context,
	execCtx *ExecutionContext,
	workflow *core.Workflow,
	task *core.Task,
) error {
	if err := e.concurrencyLimiter.Acquire(ctx); err != nil {
		return err
	}
	defer e.concurrencyLimiter.Release()

	var handler TaskHandler

	agentIDs := []string{}
	if task.AgentID != "" {
		agentIDs = append(agentIDs, task.AgentID)
	} else {
		// Use router to select agent
		var err error
		agentIDs, err = e.router.Route(ctx, task, e.agentRegistry.List())
		if err != nil {
			return fmt.Errorf("failed to route task: %w", err)
		}
	}

	if len(agentIDs) > 0 {
		// Agent-based execution with fallback
		agentExec := agents.NewAgentExecutor(e.agentRegistry, e.artifactRegistry)
		handler = func(ctx context.Context, execCtx *ExecutionContext, t *core.Task) (map[string]any, error) {
			var lastErr error
			for _, id := range agentIDs {
				output, err := agentExec.Execute(ctx, id, t)
				if err == nil {
					return output, nil
				}
				lastErr = err
				fmt.Printf("[Executor] Fallback: Agent %s failed, trying next... error: %v\n", id, err)
			}
			return nil, fmt.Errorf("all agents failed: %w", lastErr)
		}
	} else {
		// Worker-based execution
		worker, exists := e.registry.Get(task.Name)
		if !exists {
			err := fmt.Errorf("worker not found: %s", task.Name)
			_ = task.Fail(err)
			_ = e.store.UpdateTask(store.TaskToRecord(workflow.ID, task))
			return err
		}
		retryWorker := NewRetryWorkerWrapper(worker, e.eventBus)
		handler = retryWorker.Execute
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

	// Execution Interceptors (Before)
	for _, p := range e.pluginRegistry.List(plugin.PluginTypeObserver) {
		if interceptor, ok := p.(plugin.ExecutionInterceptor); ok {
			if err := interceptor.BeforeTask(ctx, task); err != nil {
				return err
			}
		}
	}

	e.mu.RLock()
	middlewares := e.taskMiddlewares
	e.mu.RUnlock()

	finalHandler := ApplyTaskMiddleware(handler, middlewares...)
	output, err := finalHandler(ctx, execCtx, task)


	// Execution Interceptors (After)
	for _, p := range e.pluginRegistry.List(plugin.PluginTypeObserver) {
		if interceptor, ok := p.(plugin.ExecutionInterceptor); ok {
			_ = interceptor.AfterTask(ctx, task, output, err)
		}
	}

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

		failurePolicy := task.GetFailurePolicy()

		if failurePolicy == core.FailurePolicyContinueOnFailure {
			return nil // Pretend it succeeded to continue workflow
		}
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
