package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

// AgentRole defines the role of an agent.
type AgentRole string

const (
	RolePlanner   AgentRole = "PLANNER"
	RoleResearcher AgentRole = "RESEARCHER"
	RoleExecutor  AgentRole = "EXECUTOR"
	RoleReviewer  AgentRole = "REVIEWER"
	RoleEvaluator AgentRole = "EVALUATOR"
)

// Agent represents a runtime entity with specific configuration.
type Agent struct {
	ID           string
	Name         string
	Description  string
	Role         AgentRole
	SystemPrompt string
	Model        string
	Provider     string
	Tools        []string // List of tool names this agent can use
	Config       map[string]any

	mu sync.RWMutex
}

// AgentRegistry manages agent definitions.
type AgentRegistry struct {
	agents map[string]*Agent
	mu     sync.RWMutex
}

func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*Agent),
	}
}

func (r *AgentRegistry) Register(a *Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agents[a.ID]; exists {
		return fmt.Errorf("agent already exists: %s", a.ID)
	}
	r.agents[a.ID] = a
	return nil
}

func (r *AgentRegistry) Get(id string) (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, exists := r.agents[id]
	return a, exists
}

func (r *AgentRegistry) List() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Agent, 0, len(r.agents))
	for _, a := range r.agents {
		list = append(list, a)
	}
	return list
}

// AgentExecutor is responsible for executing tasks using an agent.
type AgentExecutor struct {
	registry *AgentRegistry
	// In the future, this will link to LLM workers and Tool runtime
}

func NewAgentExecutor(r *AgentRegistry) *AgentExecutor {
	return &AgentExecutor{
		registry: r,
	}
}

func (e *AgentExecutor) Execute(ctx context.Context, agentID string, task *core.Task) (map[string]any, error) {
	agent, exists := e.registry.Get(agentID)
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// This is where the actual LLM interaction will happen in Step 2/3
	fmt.Printf("Agent %s (%s) is executing task: %s\n", agent.Name, agent.Role, task.Name)
	
	return map[string]any{
		"status": "simulated_success",
		"agent":  agent.Name,
		"role":   string(agent.Role),
	}, nil
}
