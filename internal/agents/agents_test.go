package agents

import (
	"context"
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

func TestAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &Agent{
		ID:   "agent-1",
		Name: "Test Agent",
		Role: RolePlanner,
	}

	err := registry.Register(agent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	retrieved, ok := registry.Get("agent-1")
	if !ok {
		t.Fatal("Failed to retrieve agent")
	}

	if retrieved.Name != "Test Agent" {
		t.Errorf("Expected Test Agent, got %s", retrieved.Name)
	}

	// Test duplicate registration
	err = registry.Register(agent)
	if err == nil {
		t.Error("Expected error for duplicate registration, got nil")
	}
}

func TestAgentExecutor(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &Agent{
		ID:   "agent-1",
		Name: "Test Agent",
		Role: RoleExecutor,
	}
	registry.Register(agent)

	executor := NewAgentExecutor(registry)
	task := core.NewTask("t1", "test-task", "desc", nil, nil)

	output, err := executor.Execute(context.Background(), "agent-1", task)
	if err != nil {
		t.Fatalf("Executor.Execute failed: %v", err)
	}

	if output["agent"] != "Test Agent" {
		t.Errorf("Expected Test Agent in output, got %v", output["agent"])
	}
}
