package worker

import (
	"context"
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
)

const defaultPoolSize = 4

type Pool struct {
	size     int
	queue    queue.Queue
	executor *executor.Executor
	obs      *observability.Obs
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

// NewPool constructs a Pool.  Pass size ≤ 0 to use defaultPoolSize.
func NewPool(size int, q queue.Queue, exec *executor.Executor, obs *observability.Obs) *Pool {
	if size <= 0 {
		size = defaultPoolSize
	}
	return &Pool{
		size:     size,
		queue:    q,
		executor: exec,
		obs:      obs,
	}
}

// Start launches all worker goroutines and begins processing tasks.
func (p *Pool) Start(ctx context.Context) {
	poolCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	p.obs.Log.Info(ctx, "worker pool starting",
		observability.F("worker_count", p.size),
	)

	for i := 1; i <= p.size; i++ {
		w := newWorker(i, p.queue, p.executor, p.obs)
		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()
			w.run(poolCtx)
		}(w)
	}
}

// Stop signals all workers to stop and waits for them to finish their current tasks.
func (p *Pool) Stop() {
	p.obs.Log.Info(context.Background(), "worker pool stop requested")
	if p.cancel != nil {
		p.cancel()
	}
	p.queue.Close()
	p.wg.Wait()
	p.obs.Log.Info(context.Background(), "worker pool stopped")
}
