package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
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
	Memory       core.Memory

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

	fmt.Printf("[AgentExecutor] Agent %s (%s) is executing task: %s\n", agent.Name, agent.Role, task.Name)

	if agent.Provider != "" {
		provider, err := providers.Get(agent.Provider)
		if err != nil {
			return nil, fmt.Errorf("failed to get provider for agent %s: %w", agent.Name, err)
		}

		// Prepare messages
		messages := []providers.Message{}
		if agent.SystemPrompt != "" {
			messages = append(messages, providers.Message{Role: "system", Content: agent.SystemPrompt})
		}
		
		// If task has input prompt
		if prompt, ok := task.Input["prompt"].(string); ok {
			messages = append(messages, providers.Message{Role: "user", Content: prompt})
		} else {
			// fallback generic prompt
			messages = append(messages, providers.Message{Role: "user", Content: fmt.Sprintf("Execute task: %s. Description: %s", task.Name, task.Description)})
		}

		req := providers.GenerateRequest{
			Model:    agent.Model,
			Messages: messages,
		}

		resp, err := provider.Generate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}

		result := map[string]any{
			"status": "success",
			"agent":  agent.Name,
			"role":   string(agent.Role),
		}

		// Try to parse content as JSON to map it to task outputs
		var jsonOutput map[string]any
		if err := json.Unmarshal([]byte(resp.Content), &jsonOutput); err == nil {
			for k, v := range jsonOutput {
				result[k] = v
			}
		} else {
			result["output"] = resp.Content
		}

		return result, nil
	}

	// Fallback to simulated response if no provider is configured
	return map[string]any{
		"status": "simulated_success",
		"agent":  agent.Name,
		"role":   string(agent.Role),
	}, nil
}
