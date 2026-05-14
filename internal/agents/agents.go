package agents

import (
	"context"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

// AgentRole defines the role of an agent in a multi-agent system.
type AgentRole string

const (
	RolePlanner  AgentRole = "PLANNER"
	RoleExecutor AgentRole = "EXECUTOR"
	RoleReviewer AgentRole = "REVIEWER"
)

// Agent represents an autonomous entity capable of performing tasks.
type Agent struct {
	ID   string
	Role AgentRole
}

// Message represents communication between agents.
type Message struct {
	FromID  string
	ToID    string
	Content string
}

// Coordinator manages agent interactions and task delegation.
type Coordinator struct {
	agents map[string]*Agent
}

func NewCoordinator() *Coordinator {
	return &Coordinator{
		agents: make(map[string]*Agent),
	}
}

func (c *Coordinator) RegisterAgent(a *Agent) {
	c.agents[a.ID] = a
}

// DelegateTask assigns a task to the most appropriate agent.
func (c *Coordinator) DelegateTask(ctx context.Context, task *core.Task) (string, error) {
	// Simple delegation logic for now
	for id := range c.agents {
		return id, nil
	}
	return "", nil
}
