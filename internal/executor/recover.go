package executor

import (
	"context"
	"fmt"
	"log"
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

		// Check workflow-level timeout
		if wfRecord.Timeout > 0 && wfRecord.StartedAt != nil {
			if time.Since(*wfRecord.StartedAt) > wfRecord.Timeout {
				log.Printf("Workflow %s timed out, attempting to fail it", wfRecord.ID)
				// Here we should ideally trigger a workflow failure
				// For now, we'll focus on tasks.
			}
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

			// Check task-level timeout
			// Task timeouts are usually handled by context, but if a worker is misbehaving
			// or the system crashed, the status might stay 'RUNNING'.
			// In a local-only system, this is less likely unless the app crashes and restarts.
			
			if taskRecord.Timeout > 0 && taskRecord.StartedAt != nil {
				if time.Since(*taskRecord.StartedAt) > taskRecord.Timeout + (5 * time.Second) { // 5s grace period
					log.Printf("Task %s in workflow %s is stuck (timed out), marking as failed", taskRecord.ID, wfRecord.ID)
					
					taskRecord.Status = string(core.TaskFailed)
					taskRecord.Error = "task stuck: timeout exceeded"
					now := time.Now()
					taskRecord.FinishedAt = &now
					
					if err := s.store.UpdateTask(taskRecord); err != nil {
						log.Printf("Failed to update stuck task %s: %v", taskRecord.ID, err)
					}
				}
			}
		}
	}

	return nil
}
