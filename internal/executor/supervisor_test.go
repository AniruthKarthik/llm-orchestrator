package executor

import (
	"testing"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
)

func TestSupervisor_CheckStuckTasks(t *testing.T) {
	s := store.NewMemoryStore()
	ar := agents.NewAgentRegistry()
	art := core.NewArtifactRegistry()
	mem := core.NewMemoryRegistry()
	tr := core.NewToolRegistry()
	tp := core.NewToolPolicy()
	e := NewExecutor(NewWorkerRegistry(), ar, art, mem, tr, tp, nil, s)
	supervisor := NewSupervisor(s, e, 100*time.Millisecond)

	// 1. Setup a "stuck" workflow with a task past its timeout.
	workflowID := "wf-1"
	taskID := "task-1"
	startedAt := time.Now().Add(-10 * time.Minute)

	// Register a cancel func so we can observe when the supervisor calls it.
	cancelled := make(chan struct{})
	e.mu.Lock()
	e.workflowCancels[workflowID] = func() {
		close(cancelled)
	}
	e.mu.Unlock()

	err := s.SaveWorkflow(store.WorkflowRecord{
		ID:        workflowID,
		Status:    string(core.WorkflowRunning),
		StartedAt: &startedAt,
		Timeout:   5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("failed to save workflow: %v", err)
	}

	err = s.SaveTask(store.TaskRecord{
		ID:         taskID,
		WorkflowID: workflowID,
		Status:     string(core.TaskRunning),
		StartedAt:  &startedAt,
		Timeout:    1 * time.Minute,
	})
	if err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	// 2. Run supervisor check.
	err = supervisor.checkStuckTasks()
	if err != nil {
		t.Fatalf("checkStuckTasks failed: %v", err)
	}

	// 3. Verify the supervisor called CancelWorkflow (the cancel func was invoked).
	select {
	case <-cancelled:
		// Success — the executor's context was cancelled.
	case <-time.After(time.Second):
		t.Fatal("supervisor did not cancel the workflow context for stuck task")
	}
}
