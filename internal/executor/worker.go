package executor

import (
	"context"
	"fmt"
	"sort"
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

// WorkerRegistry stores the set of task name keyed workers available to the executor
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

// Register adds a worker for the given task name. It returns an error if a
// worker is already registered under that name
func (r *WorkerRegistry) Register(
	taskName string,
	worker Worker,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.workers[taskName]; exists {
		return fmt.Errorf("worker already registered: %s", taskName)
	}
	r.workers[taskName] = worker
	return nil
}

// Get returns the worker registered for the given task name, if any.
func (r *WorkerRegistry) Get(
	taskName string,
) (Worker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, exists := r.workers[taskName]
	return w, exists
}

// Unregister removes the worker for the given task name.
func (r *WorkerRegistry) Unregister(taskName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workers, taskName)
}

func (r *WorkerRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.workers))
	for name := range r.workers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
