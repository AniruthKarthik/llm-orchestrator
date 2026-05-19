package executor

import (
	"context"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

// Router determines which agent or model should handle a task.
type Router interface {
	Route(ctx context.Context, task *core.Task, availableAgents []*agents.Agent) ([]string, error)
}

// CapabilityAwareRouter selects an agent based on task requirements and agent capabilities.
type CapabilityAwareRouter struct {
	// We might need an LLM here to analyze the task if requirements are not explicit.
}

func NewCapabilityAwareRouter() *CapabilityAwareRouter {
	return &CapabilityAwareRouter{}
}

func (r *CapabilityAwareRouter) Route(ctx context.Context, task *core.Task, availableAgents []*agents.Agent) ([]string, error) {
	// Minimal implementation:
	// 1. If task already has AgentID, use it.
	if task.AgentID != "" {
		return []string{task.AgentID}, nil
	}

	// 2. Look for an agent that matches the task name or description (simple heuristic)
	// In a real implementation, we would use an LLM or a more complex matcher.
	ids := make([]string, 0)
	for _, agent := range availableAgents {
		if agent.Role == agents.RoleExecutor {
			ids = append(ids, agent.ID)
		}
	}

	if len(ids) > 0 {
		return ids, nil
	}

	if len(availableAgents) > 0 {
		return []string{availableAgents[0].ID}, nil
	}

	return nil, nil
}

// CostAwareRouter optimizes for budget by selecting cheaper models.
type CostAwareRouter struct {
	costRegistry *ModelCostRegistry
}

func NewCostAwareRouter(cr *ModelCostRegistry) *CostAwareRouter {
	return &CostAwareRouter{
		costRegistry: cr,
	}
}

func (r *CostAwareRouter) Route(ctx context.Context, task *core.Task, availableAgents []*agents.Agent) ([]string, error) {
	if task.AgentID != "" {
		return []string{task.AgentID}, nil
	}

	type agentCost struct {
		id   string
		cost float64
	}
	var agentsWithCosts []agentCost

	for _, agent := range availableAgents {
		if agent.Role != agents.RoleExecutor {
			continue
		}

		cost, exists := r.costRegistry.Get(agent.Model)
		var totalCost float64
		if !exists {
			totalCost = 1000.0 // Assume expensive if unknown
		} else {
			totalCost = cost.PromptCost + cost.CompletionCost
		}
		agentsWithCosts = append(agentsWithCosts, agentCost{agent.ID, totalCost})
	}

	// Simple sort (manual since it's likely few agents)
	for i := 0; i < len(agentsWithCosts); i++ {
		for j := i + 1; j < len(agentsWithCosts); j++ {
			if agentsWithCosts[i].cost > agentsWithCosts[j].cost {
				agentsWithCosts[i], agentsWithCosts[j] = agentsWithCosts[j], agentsWithCosts[i]
			}
		}
	}

	ids := make([]string, len(agentsWithCosts))
	for i, ac := range agentsWithCosts {
		ids[i] = ac.id
	}

	return ids, nil
}

