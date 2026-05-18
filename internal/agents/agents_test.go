package agents

import (
	"context"
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Generate(ctx context.Context, req providers.GenerateRequest) (*providers.GenerateResponse, error) {
	return &providers.GenerateResponse{Content: `{"status": "success", "agent": "Test Agent"}`}, nil
}
func (m *mockProvider) Stream(ctx context.Context, req providers.GenerateRequest) (<-chan providers.StreamChunk, <-chan error) {
	return nil, nil
}
func (m *mockProvider) Capabilities() providers.Capabilities { return providers.Capabilities{} }
func (m *mockProvider) ListModels(ctx context.Context) ([]string, error) { return []string{}, nil }

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
	// Register the mock provider in the global registry so the executor can find it.
	providers.Register(&mockProvider{})
	defer providers.Clear() // Cleanup after test

	registry := NewAgentRegistry()
	agent := &Agent{
		ID:       "agent-1",
		Name:     "Test Agent",
		Role:     RoleExecutor,
		Provider: "mock", // must match mockProvider.Name()
	}
	registry.Register(agent)

	executor := NewAgentExecutor(registry, nil, nil)
	task := core.NewTask("t1", "test-wf", "test-task", "desc", nil, nil)

	output, err := executor.Execute(context.Background(), "agent-1", task)
	if err != nil {
		t.Fatalf("Executor.Execute failed: %v", err)
	}

	if output["agent"] != "Test Agent" {
		t.Errorf("Expected Test Agent in output, got %v", output["agent"])
	}
}

