package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

// Scheduler manages the assignment of tasks to workers via a queue.
type Scheduler struct {
	queue    TaskQueue
	executor *Executor
	store    store.Store
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	running  bool
}

func NewScheduler(q TaskQueue, e *Executor, s store.Store) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		queue:    q,
		executor: e,
		store:    s,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// ScheduleTask adds a task to the queue for future execution.
func (s *Scheduler) ScheduleTask(workflowID, taskID string, priority int) error {
	return s.queue.Push(&QueuedTask{
		WorkflowID: workflowID,
		TaskID:     taskID,
		Priority:   priority,
	})
}

// Start begins the scheduling loop, pulling tasks from the queue and executing them.
func (s *Scheduler) Start(workerCount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	for i := 0; i < workerCount; i++ {
		go s.workerLoop()
	}
}

// Stop halts the scheduler and its workers.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.cancel()
	s.running = false
}

func (s *Scheduler) workerLoop() {
	for {
		queuedTask, err := s.queue.Pop(s.ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if err := s.executeQueuedTask(queuedTask); err != nil {
			// Handle execution failure (e.g., retry or log)
			fmt.Printf("failed to execute queued task %s: %v\n", queuedTask.TaskID, err)
		}
	}
}


func (s *Scheduler) executeQueuedTask(qt *QueuedTask) error {
	wfRecord, err := s.store.GetWorkflow(qt.WorkflowID)
	if err != nil {
		return err
	}

	taskRecords, err := s.store.GetWorkflowTasks(qt.WorkflowID)
	if err != nil {
		return err
	}

	workflow := store.RecordToWorkflow(wfRecord, taskRecords)
	task, err := workflow.GetTask(qt.TaskID)
	if err != nil {
		return err
	}

	execCtx := NewExecutionContext(workflow.ID)
	// In a real scenario, we might want to recover shared memory from previous tasks.

	return s.executor.executeTask(s.ctx, execCtx, workflow, task)
}
