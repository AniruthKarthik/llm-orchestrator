package worker

import (
	"context"
	"log"

	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
)

type Worker struct {
	id       int
	queue    queue.Queue
	executor *executor.Executor
}

// newWorker creates a single worker instance for the pool.
func newWorker(id int, q queue.Queue, exec *executor.Executor) *Worker {
	return &Worker{id: id, queue: q, executor: exec}
}

// run is the internal loop where the worker waits for and executes tasks.
func (w *Worker) run(ctx context.Context) {
	log.Printf("[worker-%d] started", w.id)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[worker-%d] context cancelled — exiting", w.id)
			return
		default:
		}

		task, ok := w.queue.Dequeue()
		if !ok {
			log.Printf("[worker-%d] queue drained — exiting", w.id)
			return
		}

		log.Printf("[worker-%d] dequeued task %s (job %s)", w.id, task.ID, task.JobID)
		w.executor.Run(ctx, task)
	}
}
