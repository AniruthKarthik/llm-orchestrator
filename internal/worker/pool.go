package worker

import (
	"context"
	"log"
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/queue"
)

const defaultPoolSize = 4

type Pool struct {
	size     int
	queue    queue.Queue
	executor *executor.Executor
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

// NewPool initializes a worker pool with a specific size and task source.
func NewPool(size int, q queue.Queue, exec *executor.Executor) *Pool {
	if size <= 0 {
		size = defaultPoolSize
	}
	return &Pool{
		size:     size,
		queue:    q,
		executor: exec,
	}
}

// Start spawns the configured number of workers and begins processing tasks.
func (p *Pool) Start(ctx context.Context) {
	poolCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	log.Printf("[pool] starting %d workers", p.size)

	for i := 1; i <= p.size; i++ {
		w := newWorker(i, p.queue, p.executor)
		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()
			w.run(poolCtx)
		}(w)
	}
}

// Stop signals all workers to exit and waits for them to finish their current task.
func (p *Pool) Stop() {
	log.Println("[pool] stop requested")
	if p.cancel != nil {
		p.cancel()
	}
	p.queue.Close()
	p.wg.Wait()
	log.Println("[pool] all workers stopped")
}
