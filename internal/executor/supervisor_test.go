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
	e := NewExecutor(NewWorkerRegistry(), ar, art, nil, s)
	supervisor := NewSupervisor(s, e, 100*time.Millisecond)

	// 1. Setup a stuck task
	workflowID := "wf-1"
	taskID := "task-1"
	
	startedAt := time.Now().Add(-10 * time.Minute)
	
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

	// 2. Run supervisor check
	err = supervisor.checkStuckTasks()
	if err != nil {
		t.Fatalf("checkStuckTasks failed: %v", err)
	}

	// 3. Verify task was failed
	taskRecord, err := s.GetTask(workflowID, taskID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if taskRecord.Status != string(core.TaskFailed) {
		t.Errorf("expected task status to be FAILED, got %s", taskRecord.Status)
	}

	if taskRecord.Error != "task stuck: timeout exceeded" {
		t.Errorf("expected specific error message, got %s", taskRecord.Error)
	}
}
