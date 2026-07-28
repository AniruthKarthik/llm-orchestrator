package executor

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

// Supervisor monitors running workflows and tasks for stuck execution paths.
type Supervisor struct {
	store    store.Store
	executor *Executor
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	running  bool
}

// NewSupervisor creates a new supervisor instance.
func NewSupervisor(s store.Store, e *Executor, interval time.Duration) *Supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Supervisor{
		store:    s,
		executor: e,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the supervision loop.
func (s *Supervisor) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	go s.loop()
}

// Stop halts the supervisor.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.cancel()
	s.running = false
}

func (s *Supervisor) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.checkStuckTasks(); err != nil {
				log.Printf("Supervisor error checking stuck tasks: %v", err)
			}
		}
	}
}

func (s *Supervisor) checkStuckTasks() error {
	workflows, err := s.store.ListWorkflows()
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	for _, wfRecord := range workflows {
		if wfRecord.Status != string(core.WorkflowRunning) {
			continue
		}

		tasks, err := s.store.GetWorkflowTasks(wfRecord.ID)
		if err != nil {
			log.Printf("Failed to get tasks for workflow %s: %v", wfRecord.ID, err)
			continue
		}

		for _, taskRecord := range tasks {
			if taskRecord.Status != string(core.TaskRunning) {
				continue
			}

			if taskRecord.Timeout > 0 && taskRecord.StartedAt != nil {
				if time.Since(*taskRecord.StartedAt) > taskRecord.Timeout+(5*time.Second) { // 5s grace period
					log.Printf("Task %s in workflow %s is stuck (timed out), cancelling workflow", taskRecord.ID, wfRecord.ID)

					// Cancel the workflow context rather than writing FAILED directly
					// to the store. The executor goroutine will detect the cancellation
					// via ctx.Done() and fail the task through the normal error path,
					// which guarantees a single writer for terminal-status transitions.
					s.executor.CancelWorkflow(wfRecord.ID)
					break // one stuck task is enough to cancel the entire workflow
				}
			}
		}
	}

	return nil
}

// PanicRecoveryMiddleware catches panics in task handlers and converts them to errors.
func PanicRecoveryMiddleware(next TaskHandler) TaskHandler {
	return func(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (output map[string]any, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				err = fmt.Errorf("task panicked: %v\nstack trace:\n%s", r, string(stack))
			}
		}()

		return next(ctx, execCtx, task)
	}
}
