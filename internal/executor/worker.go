package executor

import (
	"context"
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

type Worker interface {
	Execute(
		ctx context.Context,
		execCtx *ExecutionContext,
		task *core.Task,
	) (map[string]any, error)
}

type WorkerRegistry struct {
	workers map[string]Worker
	mu      sync.RWMutex
}

func NewWorkerRegistry() *WorkerRegistry {
	newWorkerReg := &WorkerRegistry{
		workers: make(map[string]Worker),
	}
	return newWorkerReg
}

func (r *WorkerRegistry) Register(
	taskName string,
	worker Worker,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workers[taskName] = worker
}

func (r *WorkerRegistry) Get(
	taskName string,
) (Worker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, exists := r.workers[taskName]
	return w, exists
}
