package executor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/dag"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/plugin"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
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

	// workflowCancels tracks cancel functions for all in flight workflows.
	workflowCancels map[string]context.CancelFunc
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
		workflowCancels:     make(map[string]context.CancelFunc),
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

// CancelWorkflow cancels the context of an in flight workflow.
func (e *Executor) CancelWorkflow(workflowID string) {
	e.mu.RLock()
	cancel, ok := e.workflowCancels[workflowID]
	e.mu.RUnlock()
	if ok {
		cancel()
	}
}

func (e *Executor) GetAgentRegistry() *agents.AgentRegistry {
	return e.agentRegistry
}

func (e *Executor) GetArtifactRegistry() *core.ArtifactRegistry {
	return e.artifactRegistry
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

	// Register the cancel func so the Supervisor can abort this workflow.
	e.mu.Lock()
	e.workflowCancels[workflow.ID] = workflowCancel
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.workflowCancels, workflow.ID)
		e.mu.Unlock()
	}()

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

	e.eventBus.Publish(events.Event{
		Type:       events.WorkflowStarted,
		WorkflowID: workflow.ID,
		Timestamp:  time.Now(),
	})

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

		e.eventBus.Publish(events.Event{
			Type:       events.StageStarted,
			WorkflowID: workflow.ID,
			Timestamp:  time.Now(),
			Payload: map[string]any{
				"level":   stage.Level,
				"taskIds": stage.TaskIDs,
			},
		})

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
				e.eventBus.Publish(events.Event{
					Type:       events.StageFailed,
					WorkflowID: workflow.ID,
					Timestamp:  time.Now(),
					Payload: map[string]any{
						"level":   stage.Level,
						"taskIds": stage.TaskIDs,
						"error":   stageErr.Error(),
					},
				})
				continue
			}

			if failErr := workflow.Fail(); failErr != nil {
				return fmt.Errorf("failed to fail workflow: %v (original error: %v)", failErr, stageErr)
			}

			_ = e.store.UpdateWorkflow(
				store.WorkflowToRecord(workflow),
			)

			e.eventBus.Publish(events.Event{
				Type:       events.StageFailed,
				WorkflowID: workflow.ID,
				Timestamp:  time.Now(),
				Payload: map[string]any{
					"level":   stage.Level,
					"taskIds": stage.TaskIDs,
					"error":   stageErr.Error(),
				},
			})

			e.eventBus.Publish(events.Event{
				Type:       events.WorkflowFailed,
				WorkflowID: workflow.ID,
				Timestamp:  time.Now(),
				Payload: map[string]any{
					"error": stageErr.Error(),
				},
			})

			return stageErr
		}

		e.eventBus.Publish(events.Event{
			Type:       events.StageCompleted,
			WorkflowID: workflow.ID,
			Timestamp:  time.Now(),
			Payload: map[string]any{
				"level":   stage.Level,
				"taskIds": stage.TaskIDs,
			},
		})

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

	e.eventBus.Publish(events.Event{
		Type:       events.WorkflowCompleted,
		WorkflowID: workflow.ID,
		Timestamp:  time.Now(),
	})

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

	// Inject outputs from all dependency tasks into this task's Input.
	// This gives the downstream agent structured access to upstream results
	// under the key "dep_outputs" (a map of taskID -> output) as well as
	// individual keys "dep_<taskID>_<field>" for flat access.
	//
	// SetInput acquires the task lock so this is safe when other
	// components (e.g. the Supervisor or the API layer) read the task's
	// input map concurrently.
	if len(task.Dependencies) > 0 {
		depOutputs := make(map[string]any, len(task.Dependencies))
		for _, depID := range task.Dependencies {
			depTask, err := workflow.GetTask(depID)
			if err != nil {
				continue
			}
			depOut := depTask.GetOutput()
			if len(depOut) == 0 {
				continue
			}
			depOutputs[depID] = depOut
			// Also inject a human-readable label using task name if available
			depOutputs[depTask.Name] = depOut
		}
		if len(depOutputs) > 0 {
			task.SetInput("dep_outputs", depOutputs)
		}
	}

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
		agentExec := agents.NewAgentExecutor(e.agentRegistry, e.artifactRegistry, e.eventBus)
		handler = func(ctx context.Context, execCtx *ExecutionContext, t *core.Task) (map[string]any, error) {
			var lastErr error
			for _, id := range agentIDs {
				output, err := agentExec.Execute(ctx, id, t)
				if err == nil {
					return output, nil
				}
				lastErr = err
				slog.Warn("agent fallback", "agent_id", id, "error", err)
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
		handler = worker.Execute
	}

	if err := task.Start(); err != nil {
		return err
	}

	if err := e.store.UpdateTask(
		store.TaskToRecord(workflow.ID, task),
	); err != nil {
		return err
	}

	if task.RequiresApproval {
		if err := task.WaitForApproval(); err != nil {
			return err
		}
		if err := e.store.UpdateTask(store.TaskToRecord(workflow.ID, task)); err != nil {
			return err
		}

		slog.Info("task waiting for approval", "task_id", task.ID, "workflow_id", workflow.ID)

		e.eventBus.Publish(events.Event{
			Type:       events.TaskWaitingForApproval,
			TaskID:     task.ID,
			WorkflowID: workflow.ID,
			Timestamp:  time.Now(),
		})

		// Wait for approval via store update (e.g., from API)
		// Using a ticker so ctx.Done() is checked immediately on every
		// iteration rather than only after a blocking time.Sleep
		pollInterval := 1 * time.Second
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

	approvalLoop:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				rec, err := e.store.GetTask(workflow.ID, task.ID)
				if err != nil {
					slog.Warn("approval poll: failed to get task",
						"task_id", task.ID,
						"workflow_id", workflow.ID,
						"error", err,
					)
					continue
				}
				if rec.Status == string(core.TaskRunning) {
					// Approved externally so update local state
					_ = task.Approve()
					break approvalLoop
				}
			}
		}
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
	output, err := e.executeWithRetry(ctx, execCtx, workflow.ID, task, finalHandler)

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

	// Save output as an artifact automatically
	artifactID := fmt.Sprintf("art-%s", task.ID)
	artifact := &core.Artifact{
		ID:         artifactID,
		WorkflowID: workflow.ID,
		TaskID:     task.ID,
		Name:       fmt.Sprintf("%s Output", task.Name),
		Type:       core.ArtifactTypeJSON,
		Data:       output,
		CreatedAt:  time.Now(),
	}
	_ = e.artifactRegistry.Register(artifact)
	_ = e.store.SaveArtifact(store.ArtifactToRecord(artifact))

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

func (e *Executor) executeWithRetry(
	ctx context.Context,
	execCtx *ExecutionContext,
	workflowID string,
	task *core.Task,
	handler TaskHandler,
) (map[string]any, error) {
	policy := task.RetryPolicy
	if policy == nil {
		policy = &core.RetryPolicy{
			MaxRetries: 0,
			Strategy:   core.RetryStrategyImmediate,
		}
	}

	var lastErr error
	for attempt := 1; attempt <= policy.MaxRetries+1; attempt++ {
		task.SetAttempt(attempt)
		_ = e.store.UpdateTask(store.TaskToRecord(workflowID, task))

		output, err := handler(ctx, execCtx, task)
		if err == nil {
			return output, nil
		}
		lastErr = err

		if attempt > policy.MaxRetries {
			break
		}

		delay := policy.CalculateDelay(attempt)
		e.eventBus.Publish(events.Event{
			Type:       events.TaskRetried,
			TaskID:     task.ID,
			WorkflowID: workflowID,
			Timestamp:  time.Now(),
			Payload: map[string]any{
				"attempt":     attempt,
				"nextAttempt": attempt + 1,
				"maxRetries":  policy.MaxRetries,
				"error":       err.Error(),
				"delay":       delay.String(),
			},
		})

		slog.Warn("task retry scheduled",
			"workflow_id", workflowID,
			"task_id", task.ID,
			"attempt", attempt,
			"max_retries", policy.MaxRetries,
			"delay", delay.String(),
			"error", err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil, lastErr
}
