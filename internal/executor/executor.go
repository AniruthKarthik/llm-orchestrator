package executor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
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
	store store.Store
	queue queue.Queue
	tools map[string]tools.Tool
}

// New creates a new Executor with the necessary store, queue, and tool registry.
func New(s store.Store, q queue.Queue, registry []tools.Tool) *Executor {
	toolMap := make(map[string]tools.Tool, len(registry))
	for _, t := range registry {
		toolMap[t.Name()] = t
	}
	return &Executor{store: s, queue: q, tools: toolMap}
}

// Start runs a continuous loop that dequeues and processes tasks from the queue.
func (e *Executor) Start(ctx context.Context) {
	log.Println("[executor] starting worker loop")
	for {
		task, ok := e.queue.Dequeue()
		if !ok {
			log.Println("[executor] queue closed, stopping worker loop")
			return
		}

		// Check context before starting task
		select {
		case <-ctx.Done():
			log.Println("[executor] context cancelled, stopping worker loop")
			return
		default:
		}

		go e.Run(ctx, task)
	}
}

// Run handles the complete lifecycle of a single task, from dependency checks to execution.
func (e *Executor) Run(ctx context.Context, task models.Task) {
	log.Printf("[executor] received task %s (job %s, type %s)", task.ID, task.JobID, task.Type)

	current, err := e.store.GetTask(task.JobID, task.ID)
	if err != nil {
		log.Printf("[executor] cannot fetch task %s: %v — skipping", task.ID, err)
		return
	}
	switch current.Status {
	case models.TaskStatusCompleted:
		log.Printf("[executor] task %s already completed — skipping (idempotent)", task.ID)
		return
	case models.TaskStatusFailed:
		log.Printf("[executor] task %s already failed — skipping", task.ID)
		return
	case models.TaskStatusRunning:
		log.Printf("[executor] task %s already running — dropping duplicate", task.ID)
		return
	}

	depState, err := e.checkDeps(task.JobID, task.ID)
	if err != nil {
		log.Printf("[executor] dep check failed for task %s: %v — requeuing", task.ID, err)
		e.requeueWithBackoff(task, 200*time.Millisecond)
		return
	}
	switch depState {
	case depStateFailed:
		log.Printf("[executor] task %s has a failed dependency — marking failed", task.ID)
		e.failTask(task, "upstream dependency failed")
		e.checkJobCompletion(task.JobID)
		return
	case depStateWaiting:
		log.Printf("[executor] task %s waiting for dependencies — requeuing", task.ID)
		if err := e.store.UpdateTaskStatus(task.JobID, task.ID, models.TaskStatusWaiting); err != nil {
			log.Printf("[executor] cannot set waiting status for task %s: %v", task.ID, err)
		}
		e.requeueWithBackoff(task, 300*time.Millisecond)
		return
	}

	if err := e.store.UpdateTaskStatus(task.JobID, task.ID, models.TaskStatusRunning); err != nil {
		log.Printf("[executor] cannot mark task %s running: %v", task.ID, err)
		return
	}

	job, err := e.store.GetJob(task.JobID)
	if err != nil {
		log.Printf("[executor] cannot fetch job %s: %v", task.JobID, err)
		e.failTask(task, fmt.Sprintf("job lookup failed: %v", err))
		e.checkJobCompletion(task.JobID)
		return
	}

	tool, err := e.resolveTool(task.Type)
	if err != nil {
		log.Printf("[executor] %v", err)
		e.failTask(task, err.Error())
		e.checkJobCompletion(task.JobID)
		return
	}

	result, execErr := e.executeWithRetry(ctx, task, job, tool)
	if execErr != nil {
		log.Printf("[executor] task %s exhausted retries: %v", task.ID, execErr)
		e.failTask(task, execErr.Error())
		e.checkJobCompletion(task.JobID)
		return
	}

	if err := e.store.SetTaskResult(task.JobID, task.ID, result); err != nil {
		log.Printf("[executor] cannot store result for task %s: %v", task.ID, err)
		e.failTask(task, fmt.Sprintf("store error: %v", err))
		e.checkJobCompletion(task.JobID)
		return
	}

	log.Printf("[executor] task %s completed successfully", task.ID)
	e.checkJobCompletion(task.JobID)
}

type depState int

const (
	depStateReady depState = iota
	depStateWaiting
	depStateFailed
)

// checkDeps determines if all upstream tasks have been completed successfully.
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

// executeWithRetry runs a tool with a specific input, retrying on failure up to maxTaskRetries.
func (e *Executor) executeWithRetry(
	ctx context.Context,
	task models.Task,
	job *models.Job,
	tool tools.Tool,
) (map[string]any, error) {

	input := buildInput(task, job)

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
		log.Printf("[executor] task %s attempt %d/%d failed: %v",
			task.ID, attempt, maxTaskRetries, err)

		if incrErr := e.store.IncrTaskRetries(task.JobID, task.ID); incrErr != nil {
			log.Printf("[executor] cannot increment retries for task %s: %v", task.ID, incrErr)
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

// resolveTool maps a task type string to its corresponding tool implementation.
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

// failTask marks a task as failed in the store and logs the reason.
func (e *Executor) failTask(task models.Task, reason string) {
	if err := e.store.UpdateTaskStatus(task.JobID, task.ID, models.TaskStatusFailed); err != nil {
		log.Printf("[executor] cannot mark task %s failed: %v", task.ID, err)
	}
	log.Printf("[executor] task %s marked failed: %s", task.ID, reason)
}

// checkJobCompletion examines all tasks in a job and updates the job's final status if finished.
func (e *Executor) checkJobCompletion(jobID string) {
	done, err := e.store.AllTasksDone(jobID)
	if err != nil {
		log.Printf("[executor] AllTasksDone check failed for job %s: %v", jobID, err)
		return
	}
	if !done {
		return
	}

	anyFailed, err := e.store.AnyTaskFailed(jobID)
	if err != nil {
		log.Printf("[executor] AnyTaskFailed check failed for job %s: %v", jobID, err)
		return
	}
	if anyFailed {
		if err := e.store.SetJobError(jobID, "one or more tasks failed"); err != nil {
			log.Printf("[executor] cannot set job error for %s: %v", jobID, err)
		}
		log.Printf("[executor] job %s finalised — FAILED", jobID)
		return
	}

	job, err := e.store.GetJob(jobID)
	if err != nil {
		log.Printf("[executor] cannot fetch job %s for finalisation: %v", jobID, err)
		return
	}
	aggregated := buildAggregatedResult(job)
	if err := e.store.SetJobResult(jobID, aggregated); err != nil {
		log.Printf("[executor] cannot set job result for %s: %v", jobID, err)
		return
	}
	log.Printf("[executor] job %s finalised — COMPLETED", jobID)
}

// requeueWithBackoff delays and then re-submits a task to the queue for retry.
func (e *Executor) requeueWithBackoff(task models.Task, delay time.Duration) {
	if delay > requeueBackoffMax {
		delay = requeueBackoffMax
	}
	go func() {
		time.Sleep(delay)
		if err := e.queue.Requeue(task); err != nil {
			log.Printf("[executor] requeue failed for task %s: %v", task.ID, err)
		}
	}()
}

// buildInput prepares the data needed by a tool by merging task payload and job context.
func buildInput(task models.Task, job *models.Job) map[string]any {
	var input map[string]any

	if m, ok := task.Payload.(map[string]any); ok {
		input = make(map[string]any, len(m)+4)
		for k, v := range m {
			input[k] = v
		}
	} else {
		input = make(map[string]any, 4)
		if task.Payload != nil {
			input["data"] = task.Payload
		}
	}

	input["job_id"] = task.JobID
	input["task_id"] = task.ID
	input["goal"] = job.Goal

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

// buildAggregatedResult collects and structures the final results of all tasks in a job.
func buildAggregatedResult(job *models.Job) map[string]any {
	taskResults := make([]map[string]any, 0, len(job.Tasks))
	for _, t := range job.Tasks {
		taskResults = append(taskResults, map[string]any{
			"id":     t.ID,
			"type":   t.Type,
			"result": t.Result,
		})
	}
	return map[string]any{
		"job_id":       job.ID,
		"goal":         job.Goal,
		"task_results": taskResults,
		"task_count":   len(job.Tasks),
	}
}
