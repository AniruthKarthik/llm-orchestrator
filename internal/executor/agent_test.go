package executor

import (
	"context"
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

// mockProvider implements providers.Provider for testing.
type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Generate(_ context.Context, _ providers.GenerateRequest) (*providers.GenerateResponse, error) {
	return &providers.GenerateResponse{Content: `{"status": "success", "agent": "Agent One"}`}, nil
}
func (m *mockProvider) Stream(_ context.Context, _ providers.GenerateRequest) (<-chan providers.StreamChunk, <-chan error) {
	return nil, nil
}
func (m *mockProvider) Capabilities() providers.Capabilities { return providers.Capabilities{} }
func (m *mockProvider) ListModels(_ context.Context) ([]string, error) { return []string{}, nil }

func TestExecutor_AgentExecution(t *testing.T) {
	// Register the mock LLM provider and clean up after the test.
	providers.Register(&mockProvider{})
	defer providers.Clear()

	// 1. Setup
	s := store.NewMemoryStore()
	ar := agents.NewAgentRegistry()
	agent := &agents.Agent{
		ID:       "agent-1",
		Name:     "Agent One",
		Role:     agents.RoleExecutor,
		Provider: "mock",
	}
	ar.Register(agent)

	wr := NewWorkerRegistry()
	eb := events.NewEventBus(10)
	art := core.NewArtifactRegistry()
	mem := core.NewMemoryRegistry()
	tr := core.NewToolRegistry()
	tp := core.NewToolPolicy()
	e := NewExecutor(wr, ar, art, mem, tr, tp, eb, s)

	// 2. Create a task assigned to the agent
	task := core.NewTask("t1", "wf-1", "agent-task", "desc", nil, nil).WithAgentID("agent-1")
	workflow := &core.Workflow{
		ID:     "wf-1",
		Tasks:  map[string]*core.Task{"t1": task},
		Status: core.WorkflowRunning,
	}

	// Save to store to avoid "workflow tasks not found" error
	s.SaveWorkflow(store.WorkflowToRecord(workflow))
	s.SaveTask(store.TaskToRecord(workflow.ID, task))

	// 3. Execute
	execCtx := NewExecutionContext("wf-1", art, mem, tr, tp)
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
