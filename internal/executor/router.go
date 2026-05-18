package executor

import (
	"context"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

// Router determines which agent or model should handle a task.
type Router interface {
	Route(ctx context.Context, task *core.Task, availableAgents []*agents.Agent) (string, error)
}

// CapabilityAwareRouter selects an agent based on task requirements and agent capabilities.
type CapabilityAwareRouter struct {
	// We might need an LLM here to analyze the task if requirements are not explicit.
}

func NewCapabilityAwareRouter() *CapabilityAwareRouter {
	return &CapabilityAwareRouter{}
}

func (r *CapabilityAwareRouter) Route(ctx context.Context, task *core.Task, availableAgents []*agents.Agent) (string, error) {
	// Minimal implementation:
	// 1. If task already has AgentID, use it.
	if task.AgentID != "" {
		return task.AgentID, nil
	}

	// 2. Look for an agent that matches the task name or description (simple heuristic)
	// In a real implementation, we would use an LLM or a more complex matcher.
	for _, agent := range availableAgents {
		if agent.Role == agents.RoleExecutor {
			return agent.ID, nil
		}
	}

	if len(availableAgents) > 0 {
		return availableAgents[0].ID, nil
	}

	return "", nil
}
