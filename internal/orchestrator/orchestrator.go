package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/memory"
	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

const (
	eventChanSize = 256
	pollInterval  = 5 * time.Second
	requeueDelay  = 500 * time.Millisecond
)

type Orchestrator struct {
	store      store.Store
	queue      queue.Queue
	retriever  *memory.Retriever
	obs        *observability.Obs
	events     chan Event
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
	activeJobs map[string]struct{}
}

func New(
	s store.Store,
	q queue.Queue,
	r *memory.Retriever,
	obs *observability.Obs,
) *Orchestrator {
	return &Orchestrator{
		store:      s,
		queue:      q,
		retriever:  r,
		obs:        obs,
		events:     make(chan Event, eventChanSize),
		stopCh:     make(chan struct{}),
		activeJobs: make(map[string]struct{}),
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		o.loop(ctx)
	}()
	o.obs.Log.Info(ctx, "orchestrator started")
}

func (o *Orchestrator) Stop() {
	close(o.stopCh)
	o.wg.Wait()
	o.obs.Log.Info(context.Background(), "orchestrator stopped")
}

func (o *Orchestrator) Publish(evt Event) {
	select {
	case o.events <- evt:
	default:
		o.obs.Log.Warn(context.Background(), "orchestrator event channel full — dropping event",
			observability.F("event_type", string(evt.Type)),
			observability.F("job_id", evt.JobID),
		)
	}
}

func (o *Orchestrator) TrackJob(jobID string) {
	o.mu.Lock()
	o.activeJobs[jobID] = struct{}{}
	o.mu.Unlock()
}

func (o *Orchestrator) loop(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopCh:
			return
		case <-ctx.Done():
			return
		case evt := <-o.events:
			o.handleEvent(ctx, evt)
		case <-ticker.C:
			o.sweepAll(ctx)
		}
	}
}

func (o *Orchestrator) handleEvent(ctx context.Context, evt Event) {
	spanCtx, span := o.obs.Tracer.Start(ctx, "orchestrator.handleEvent")
	defer span.End(spanCtx)

	spanCtx = observability.WithJobID(spanCtx, evt.JobID)

	switch evt.Type {
	case EventJobQueued:
		o.TrackJob(evt.JobID)
		o.obs.Log.Info(spanCtx, "orchestrator tracking new job")
		o.coordinateJob(spanCtx, evt.JobID)

	case EventTaskCompleted:
		spanCtx = observability.WithTaskID(spanCtx, evt.TaskID)
		o.obs.Log.Info(spanCtx, "orchestrator reacting to task completion")
		o.coordinateJob(spanCtx, evt.JobID)

	case EventTaskFailed:
		spanCtx = observability.WithTaskID(spanCtx, evt.TaskID)
		o.obs.Log.Warn(spanCtx, "orchestrator reacting to task failure")
		o.coordinateJob(spanCtx, evt.JobID)
	}
}

func (o *Orchestrator) sweepAll(ctx context.Context) {
	o.mu.RLock()
	ids := make([]string, 0, len(o.activeJobs))
	for id := range o.activeJobs {
		ids = append(ids, id)
	}
	o.mu.RUnlock()

	for _, id := range ids {
		jobCtx := observability.WithJobID(ctx, id)
		o.coordinateJob(jobCtx, id)
	}
}

func (o *Orchestrator) coordinateJob(ctx context.Context, jobID string) {
	job, err := o.store.GetJob(jobID)
	if err != nil {
		o.obs.Log.Error(ctx, "orchestrator: cannot load job", observability.F("err", err.Error()))
		return
	}

	if job.Status == models.JobStatusCompleted || job.Status == models.JobStatusFailed {
		o.untrackJob(jobID)
		return
	}

	phase := o.computePhase(job)

	switch phase {
	case phaseCompleted:
		o.obs.Log.Info(ctx, "orchestrator: all tasks done — marking job completed")
		o.finaliseJob(ctx, job, false)
		o.untrackJob(jobID)

	case phaseFailed:
		o.obs.Log.Warn(ctx, "orchestrator: tasks failed — marking job failed")
		o.finaliseJob(ctx, job, true)
		o.untrackJob(jobID)

	case phaseRunning:
		ready := o.collectReadyTasks(ctx, job)
		for _, rt := range ready {
			if err := o.enqueueTask(ctx, rt); err != nil {
				o.obs.Log.Error(ctx, "orchestrator: enqueue failed",
					observability.F("task_id", rt.Task.ID),
					observability.F("err", err.Error()),
				)
			}
		}

		if job.Status == models.JobStatusQueued {
			if err := o.retriever.StoreJobGoal(ctx, job.ID, job.Goal); err != nil {
				o.obs.Log.Warn(ctx, "orchestrator: could not store job goal in memory",
					observability.F("err", err.Error()),
				)
			}
			_ = o.store.UpdateJobStatus(jobID, models.JobStatusRunning)
		}
	}
}

func (o *Orchestrator) computePhase(job *models.Job) jobPhase {
	if len(job.Tasks) == 0 {
		return phaseCompleted
	}

	allTerminal := true
	anyFailed := false

	for _, t := range job.Tasks {
		switch t.Status {
		case models.TaskStatusCompleted:
		case models.TaskStatusFailed:
			anyFailed = true
		default:
			allTerminal = false
		}
	}

	if !allTerminal {
		return phaseRunning
	}
	if anyFailed {
		return phaseFailed
	}
	return phaseCompleted
}

func (o *Orchestrator) collectReadyTasks(ctx context.Context, job *models.Job) []ReadyTask {
	completed := make(map[string]bool, len(job.Tasks))
	for _, t := range job.Tasks {
		if t.Status == models.TaskStatusCompleted {
			completed[t.ID] = true
		}
	}

	var ready []ReadyTask
	for _, t := range job.Tasks {
		if t.Status != models.TaskStatusPending && t.Status != models.TaskStatusWaiting {
			continue
		}
		if depsOK(t, completed) {
			ready = append(ready, ReadyTask{Task: t, JobID: job.ID})
		}
	}
	return ready
}

func depsOK(t models.Task, completed map[string]bool) bool {
	for _, dep := range t.DependsOn {
		if !completed[dep] {
			return false
		}
	}
	return true
}

func (o *Orchestrator) enqueueTask(ctx context.Context, rt ReadyTask) error {
	task := rt.Task
	task.JobID = rt.JobID

	if err := o.queue.Enqueue(task); err != nil {
		return fmt.Errorf("orchestrator: queue enqueue failed: %w", err)
	}
	o.obs.Log.Info(ctx, "orchestrator enqueued ready task",
		observability.F("task_id", task.ID),
		observability.F("task_type", task.Type),
	)
	return nil
}

func (o *Orchestrator) finaliseJob(ctx context.Context, job *models.Job, failed bool) {
	if failed {
		if err := o.store.SetJobError(job.ID, "one or more tasks failed"); err != nil {
			o.obs.Log.Error(ctx, "orchestrator: SetJobError failed", observability.F("err", err.Error()))
		}
		return
	}

	result := buildAggregatedResult(job)
	if err := o.store.SetJobResult(job.ID, result); err != nil {
		o.obs.Log.Error(ctx, "orchestrator: SetJobResult failed", observability.F("err", err.Error()))
		return
	}

	if err := o.retriever.StoreTaskResult(ctx, job.ID, "final", "report", result); err != nil {
		o.obs.Log.Warn(ctx, "orchestrator: could not store final result in memory",
			observability.F("err", err.Error()),
		)
	}
	o.obs.Log.Info(ctx, "orchestrator: job finalised as completed")
}

func (o *Orchestrator) untrackJob(jobID string) {
	o.mu.Lock()
	delete(o.activeJobs, jobID)
	o.mu.Unlock()
}

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
