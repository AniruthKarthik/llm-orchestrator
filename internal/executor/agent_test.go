package executor

import (
	"context"
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

func TestExecutor_AgentExecution(t *testing.T) {
	// 1. Setup
	s := store.NewMemoryStore()
	ar := agents.NewAgentRegistry()
	agent := &agents.Agent{
		ID:   "agent-1",
		Name: "Agent One",
		Role: agents.RoleExecutor,
	}
	ar.Register(agent)

	wr := NewWorkerRegistry()
	eb := events.NewEventBus(10)
	art := core.NewArtifactRegistry()
	mem := core.NewMemoryRegistry()
	e := NewExecutor(wr, ar, art, mem, eb, s)

	// 2. Create a task assigned to the agent
	task := core.NewTask("t1", "agent-task", "desc", nil, nil).WithAgentID("agent-1")
	workflow := &core.Workflow{
		ID:    "wf-1",
		Tasks: map[string]*core.Task{"t1": task},
		Status: core.WorkflowRunning,
	}

	// Save to store to avoid "workflow tasks not found" error
	s.SaveWorkflow(store.WorkflowToRecord(workflow))
	s.SaveTask(store.TaskToRecord(workflow.ID, task))

	// 3. Execute
	execCtx := NewExecutionContext("wf-1", art, mem)
	err := e.executeTask(context.Background(), execCtx, workflow, task)
	if err != nil {
		t.Fatalf("executeTask failed: %v", err)
	}

	// 4. Verify
	if task.Status != core.TaskCompleted {
		t.Errorf("Expected task status COMPLETED, got %s", task.Status)
	}

	output := task.GetOutput()
	if output["agent"] != "Agent One" {
		t.Errorf("Expected output from Agent One, got %v", output["agent"])
	}
}
