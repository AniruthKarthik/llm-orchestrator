package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/memory"
	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/orchestrator"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/tools"
)

const (
	maxTaskRetries    = 3
	toolTimeout       = 30 * time.Second
	requeueBackoffMax = 2 * time.Second
)

type Executor struct {
	store     store.Store
	queue     queue.Queue
	tools     map[string]tools.Tool
	retriever *memory.Retriever
	orch      *orchestrator.Orchestrator
	obs       *observability.Obs
}

func New(
	s store.Store,
	q queue.Queue,
	registry []tools.Tool,
	retriever *memory.Retriever,
	obs *observability.Obs,
) *Executor {
	toolMap := make(map[string]tools.Tool, len(registry))
	for _, t := range registry {
		toolMap[t.Name()] = t
	}
	return &Executor{
		store:     s,
		queue:     q,
		tools:     toolMap,
		retriever: retriever,
		obs:       obs,
	}
}

func (e *Executor) SetOrchestrator(o *orchestrator.Orchestrator) {
	e.orch = o
}

// Run processes one task.  Called by each worker in a tight loop.
func (e *Executor) Run(ctx context.Context, task models.Task) {
	spanCtx, span := e.obs.Tracer.Start(ctx, "executor.Run")
	defer span.End(spanCtx)

	e.obs.Log.Info(spanCtx, "executor received task",
		observability.F("task_type", task.Type),
		observability.F("task_retries", task.Retries),
	)

	// ── 1. Idempotency guard ─────────────────────────────────────────────────
	current, err := e.store.GetTask(task.JobID, task.ID)
	if err != nil {
		e.obs.Log.Error(spanCtx, "executor: cannot fetch task", observability.F("err", err.Error()))
		return
	}
	switch current.Status {
	case models.TaskStatusCompleted:
		e.obs.Log.Info(spanCtx, "executor: task already completed — skipping (idempotent)")
		return
	case models.TaskStatusFailed:
		e.obs.Log.Info(spanCtx, "executor: task already failed — skipping")
		return
	case models.TaskStatusRunning:
		e.obs.Log.Info(spanCtx, "executor: task already running — dropping duplicate")
		return
	}

	// ── 2. Dependency check ──────────────────────────────────────────────────
	ds, err := e.checkDeps(task.JobID, task.ID)
	if err != nil {
		e.obs.Log.Error(spanCtx, "executor: dep check failed — requeuing", observability.F("err", err.Error()))
		span.SetError(err)
		e.requeueWithBackoff(task, 200*time.Millisecond)
		return
	}
	switch ds {
	case depStateFailed:
		e.obs.Log.Warn(spanCtx, "executor: task has a failed dependency — cascade failing")
		e.failTask(spanCtx, task, "upstream dependency failed")
		e.publishEvent(orchestrator.EventTaskFailed, task.JobID, task.ID)
		return
	case depStateWaiting:
		e.obs.Log.Info(spanCtx, "executor: task waiting for dependencies — requeuing")
		_ = e.store.UpdateTaskStatus(task.JobID, task.ID, models.TaskStatusWaiting)
		e.requeueWithBackoff(task, 300*time.Millisecond)
		return
	}

	// ── 3. Mark running ──────────────────────────────────────────────────────
	if err := e.store.UpdateTaskStatus(task.JobID, task.ID, models.TaskStatusRunning); err != nil {
		e.obs.Log.Error(spanCtx, "executor: cannot mark running", observability.F("err", err.Error()))
		return
	}

	// ── 4. Fetch job + memory context ────────────────────────────────────────
	job, err := e.store.GetJob(task.JobID)
	if err != nil {
		e.obs.Log.Error(spanCtx, "executor: job lookup failed", observability.F("err", err.Error()))
		e.failTask(spanCtx, task, fmt.Sprintf("job lookup failed: %v", err))
		e.publishEvent(orchestrator.EventTaskFailed, task.JobID, task.ID)
		return
	}

	memContext, err := e.retriever.RetrieveContext(spanCtx, job.Goal+" "+task.Type)
	if err != nil {
		// Memory retrieval is best-effort; log and continue.
		e.obs.Log.Warn(spanCtx, "executor: memory retrieval failed",
			observability.F("err", err.Error()))
		memContext = ""
	} else if memContext != "" {
		e.obs.Log.Debug(spanCtx, "executor: retrieved memory context",
			observability.F("context_len", len(memContext)))
	}

	// ── 5. Tool lookup ───────────────────────────────────────────────────────
	tool, err := e.resolveTool(task.Type)
	if err != nil {
		e.obs.Log.Error(spanCtx, "executor: tool not found", observability.F("err", err.Error()))
		span.SetError(err)
		e.failTask(spanCtx, task, err.Error())
		e.publishEvent(orchestrator.EventTaskFailed, task.JobID, task.ID)
		return
	}

	// ── 6. Execute with retries ──────────────────────────────────────────────
	start := time.Now()
	result, execErr := e.executeWithRetry(spanCtx, task, job, tool, memContext)
	elapsed := time.Since(start)

	if execErr != nil {
		e.obs.Log.Error(spanCtx, "executor: task exhausted retries",
			observability.F("err", execErr.Error()),
			observability.F("duration_ms", elapsed.Milliseconds()),
		)
		span.SetError(execErr)
		e.obs.Metrics.RecordTaskDuration(spanCtx, task.Type, elapsed, false)
		e.failTask(spanCtx, task, execErr.Error())
		e.publishEvent(orchestrator.EventTaskFailed, task.JobID, task.ID)
		return
	}

	e.obs.Metrics.RecordTaskDuration(spanCtx, task.Type, elapsed, true)

	// ── 7. Persist result ────────────────────────────────────────────────────
	if err := e.store.SetTaskResult(task.JobID, task.ID, result); err != nil {
		e.obs.Log.Error(spanCtx, "executor: cannot store result", observability.F("err", err.Error()))
		span.SetError(err)
		e.failTask(spanCtx, task, fmt.Sprintf("store error: %v", err))
		e.publishEvent(orchestrator.EventTaskFailed, task.JobID, task.ID)
		return
	}

	// ── 8. Store result in memory ────────────────────────────────────────────
	if err := e.retriever.StoreTaskResult(spanCtx, task.JobID, task.ID, task.Type, result); err != nil {
		e.obs.Log.Warn(spanCtx, "executor: could not store result in memory",
			observability.F("err", err.Error()))
	}

	e.obs.Log.Info(spanCtx, "executor: task completed",
		observability.F("duration_ms", elapsed.Milliseconds()),
	)
	e.publishEvent(orchestrator.EventTaskCompleted, task.JobID, task.ID)
}

type depState int

const (
	depStateReady depState = iota
	depStateWaiting
	depStateFailed
)

func (e *Executor) checkDeps(jobID, taskID string) (depState, error) {
	task, err := e.store.GetTask(jobID, taskID)
	if err != nil {
		return depStateWaiting, err
	}
	for _, depID := range task.DependsOn {
		dep, err := e.store.GetTask(jobID, depID)
		if err != nil {
			return depStateWaiting, fmt.Errorf("dependency %q not found: %w", depID, err)
		}
		switch dep.Status {
		case models.TaskStatusFailed:
			return depStateFailed, nil
		case models.TaskStatusCompleted:
		default:
			return depStateWaiting, nil
		}
	}
	return depStateReady, nil
}

func (e *Executor) executeWithRetry(
	ctx context.Context,
	task models.Task,
	job *models.Job,
	tool tools.Tool,
	memContext string,
) (map[string]any, error) {

	input := buildInput(task, job, memContext)
	var lastErr error

	for attempt := 1; attempt <= maxTaskRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		toolCtx, cancel := context.WithTimeout(ctx, toolTimeout)
		result, err := tool.Execute(toolCtx, input)
		cancel()

		if err == nil {
			return result, nil
		}

		lastErr = err
		e.obs.Log.Warn(ctx, "executor: tool attempt failed",
			observability.F("attempt", attempt),
			observability.F("max_attempts", maxTaskRetries),
			observability.F("err", err.Error()),
		)
		e.obs.Metrics.IncTaskRetry(ctx, task.Type)

		if incrErr := e.store.IncrTaskRetries(task.JobID, task.ID); incrErr != nil {
			e.obs.Log.Warn(ctx, "executor: cannot increment retries",
				observability.F("err", incrErr.Error()))
		}

		if attempt < maxTaskRetries {
			backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}
	}

	return nil, fmt.Errorf("all %d attempts failed, last: %w", maxTaskRetries, lastErr)
}

func (e *Executor) resolveTool(taskType string) (tools.Tool, error) {
	if t, ok := e.tools[taskType]; ok {
		return t, nil
	}
	aliases := map[string]string{
		"research":  "search",
		"generate":  "report",
		"validate":  "report",
		"fetch_url": "fetch",
		"summary":   "summarize",
	}
	if canonical, ok := aliases[taskType]; ok {
		if t, ok := e.tools[canonical]; ok {
			return t, nil
		}
	}
	return nil, fmt.Errorf("no tool registered for task type %q", taskType)
}

func (e *Executor) failTask(ctx context.Context, task models.Task, reason string) {
	if err := e.store.UpdateTaskStatus(task.JobID, task.ID, models.TaskStatusFailed); err != nil {
		e.obs.Log.Error(ctx, "executor: cannot mark task failed", observability.F("err", err.Error()))
	}
	e.obs.Log.Warn(ctx, "executor: task marked failed", observability.F("reason", reason))
}

func (e *Executor) publishEvent(evtType orchestrator.EventType, jobID, taskID string) {
	if e.orch == nil {
		return
	}
	e.orch.Publish(orchestrator.Event{
		Type:   evtType,
		JobID:  jobID,
		TaskID: taskID,
	})
}

func (e *Executor) requeueWithBackoff(task models.Task, delay time.Duration) {
	if delay > requeueBackoffMax {
		delay = requeueBackoffMax
	}
	go func() {
		time.Sleep(delay)
		if err := e.queue.Requeue(task); err != nil {
			e.obs.Log.Warn(context.Background(), "executor: requeue failed",
				observability.F("task_id", task.ID),
				observability.F("err", err.Error()),
			)
		}
	}()
}

func buildInput(task models.Task, job *models.Job, memContext string) map[string]any {
	var input map[string]any
	if m, ok := task.Payload.(map[string]any); ok {
		input = make(map[string]any, len(m)+6)
		for k, v := range m {
			input[k] = v
		}
	} else {
		input = make(map[string]any, 6)
		if task.Payload != nil {
			input["data"] = task.Payload
		}
	}

	input["job_id"] = task.JobID
	input["task_id"] = task.ID
	input["goal"] = job.Goal

	if memContext != "" {
		input["memory_context"] = memContext
	}

	results := make([]any, 0, len(job.Tasks))
	for _, t := range job.Tasks {
		if t.ID != task.ID && t.Result != nil {
			results = append(results, t.Result)
		}
	}
	if len(results) > 0 {
		input["results"] = results
	}

	return input
}
