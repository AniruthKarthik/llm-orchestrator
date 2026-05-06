package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
)

type Worker struct {
	id       int
	queue    queue.Queue
	executor *executor.Executor
	obs      *observability.Obs
}

func newWorker(id int, q queue.Queue, exec *executor.Executor, obs *observability.Obs) *Worker {
	return &Worker{id: id, queue: q, executor: exec, obs: obs}
}

func (w *Worker) run(ctx context.Context) {
	workerID := fmt.Sprintf("worker-%d", w.id)
	ctx = observability.WithWorkerID(ctx, workerID)
	w.obs.Log.Info(ctx, "worker started")

	for {
		select {
		case <-ctx.Done():
			w.obs.Log.Info(ctx, "worker context cancelled — exiting")
			return
		default:
		}

		task, ok := w.queue.Dequeue()
		if !ok {
			w.obs.Log.Info(ctx, "worker queue drained — exiting")
			return
		}

		taskCtx := observability.WithJobID(ctx, task.JobID)
		taskCtx = observability.WithTaskID(taskCtx, task.ID)

		w.obs.Log.Info(taskCtx, "worker dequeued task",
			observability.F("task_type", task.Type),
		)
		w.obs.Metrics.IncWorkerActive(taskCtx)
		w.obs.Metrics.SetQueueSize(taskCtx, w.queue.Len())

		start := time.Now()
		w.executor.Run(taskCtx, task)
		elapsed := time.Since(start)

		w.obs.Metrics.DecWorkerActive(taskCtx)
		w.obs.Metrics.RecordTaskDuration(taskCtx, task.Type, elapsed, true)
		w.obs.Log.Info(taskCtx, "worker task dispatch complete",
			observability.F("duration_ms", elapsed.Milliseconds()),
		)
	}
}
