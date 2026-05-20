package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
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
	registry         *AgentRegistry
	artifactRegistry *core.ArtifactRegistry
	eventBus         *events.EventBus
}

func NewAgentExecutor(r *AgentRegistry, ar *core.ArtifactRegistry, eb *events.EventBus) *AgentExecutor {
	return &AgentExecutor{
		registry:         r,
		artifactRegistry: ar,
		eventBus:         eb,
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
		
		var systemPrompt strings.Builder

		// 1. System Prompt
		if agent.SystemPrompt != "" {
			systemPrompt.WriteString(agent.SystemPrompt)
			systemPrompt.WriteString("\n\n")
		}

		// 2. Retrieval Injection: Inject relevant artifacts
		if e.artifactRegistry != nil {
			artifacts := e.artifactRegistry.ListByWorkflow(task.WorkflowID)
			if len(artifacts) > 0 {
				stitcher := NewContextStitcher(32000) // Increased limit to 32k chars (~8k tokens)

				// Identify dependencies for highlighting
				deps := make(map[string]bool)
				for _, d := range task.Dependencies {
					deps[d] = true
				}

				artifactStrings := make([]string, 0, len(artifacts))
				for _, a := range artifacts {
					dataJSON, _ := json.MarshalIndent(a.Data, "  ", "  ")
					
					isDep := ""
					if deps[a.TaskID] {
						isDep = " [DIRECT DEPENDENCY]"
					}

					artifactStrings = append(artifactStrings, fmt.Sprintf("- Artifact: %s%s (Type: %s, From Task: %s)\n  Data: %s", a.Name, isDep, a.Type, a.TaskID, string(dataJSON)))
				}

				contextMsg := "The following artifacts are available from previous tasks in this workflow. You can reference them by their name or the data provided below:\n"
				contextMsg += stitcher.StitchArtifacts(artifactStrings)

				systemPrompt.WriteString("## Context from previous tasks\n")
				systemPrompt.WriteString(contextMsg)
			}
		}

		if systemPrompt.Len() > 0 {
			messages = append(messages, providers.Message{Role: "system", Content: systemPrompt.String()})
		}

		// Prepare user prompt with interpolation
		userPrompt := ""
		if prompt, ok := task.Input["prompt"].(string); ok && prompt != "" {
			userPrompt = prompt
		} else {
			// fallback generic prompt
			userPrompt = fmt.Sprintf("Execute task: %s. Description: %s", task.Name, task.Description)
		}

		// 3. Template Interpolation: Replace {{Task Name.field}} with actual values
		if e.artifactRegistry != nil {
			artifacts := e.artifactRegistry.ListByWorkflow(task.WorkflowID)
			for _, a := range artifacts {
				// Remove " Output" suffix from artifact name for easier referencing if it exists
				cleanName := strings.TrimSuffix(a.Name, " Output")
				
				// Try to replace {{TaskName}} with the whole data
				dataJSON, _ := json.Marshal(a.Data)
				placeholder := fmt.Sprintf("{{%s}}", cleanName)
				userPrompt = strings.ReplaceAll(userPrompt, placeholder, string(dataJSON))

				// If data is a map, allow referencing specific fields: {{TaskName.field}}
				if dataMap, ok := a.Data.(map[string]any); ok {
					for k, v := range dataMap {
						fieldPlaceholder := fmt.Sprintf("{{%s.%s}}", cleanName, k)
						valStr := fmt.Sprintf("%v", v)
						if vs, ok := v.(string); ok {
							valStr = vs
						} else {
							vj, _ := json.Marshal(v)
							valStr = string(vj)
						}
						userPrompt = strings.ReplaceAll(userPrompt, fieldPlaceholder, valStr)
					}
				}
			}
		}

		messages = append(messages, providers.Message{Role: "user", Content: userPrompt})

		req := providers.GenerateRequest{
			Model:    agent.Model,
			Messages: messages,
		}

		resp, err := provider.Generate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}

		// Publish token usage
		if e.eventBus != nil {
			e.eventBus.Publish(events.Event{
				Type:       events.TaskTokenUsage,
				WorkflowID: task.WorkflowID,
				TaskID:     task.ID,
				Timestamp:  time.Now(),
				Payload: map[string]any{
					"model":             resp.Model,
					"prompt_tokens":     resp.Usage.PromptTokens,
					"completion_tokens": resp.Usage.CompletionTokens,
					"total_tokens":      resp.Usage.TotalTokens,
				},
			})
		}

		result := map[string]any{
			"status": "success",
			"agent":  agent.Name,
			"role":   string(agent.Role),
		}

		// Try to parse content as JSON to map it to task outputs
		var jsonOutput map[string]any
		cleanContent := resp.Content
		if start := strings.Index(cleanContent, "```json"); start != -1 {
			if end := strings.Index(cleanContent[start+7:], "```"); end != -1 {
				cleanContent = cleanContent[start+7 : start+7+end]
			}
		} else if start := strings.Index(cleanContent, "```"); start != -1 {
			if end := strings.Index(cleanContent[start+3:], "```"); end != -1 {
				cleanContent = cleanContent[start+3 : start+3+end]
			}
		}
		cleanContent = strings.TrimSpace(cleanContent)

		if err := json.Unmarshal([]byte(cleanContent), &jsonOutput); err == nil {
			for k, v := range jsonOutput {
				result[k] = v
			}
		} else {
			result["response"] = resp.Content
			result["output"] = resp.Content
		}

		// Intelligent type coercion for the 'response' key based on task schema
		if task.OutputSchema != nil {
			if expectedType, ok := task.OutputSchema["response"]; ok && expectedType == "string" {
				if val, exists := result["response"]; exists {
					if _, isString := val.(string); !isString {
						// Stringify complex objects for string-expected fields
						jsonVal, _ := json.Marshal(val)
						result["response"] = string(jsonVal)
					}
				}
			}
		}

		return result, nil
	}

	return nil, fmt.Errorf("agent %s has no provider configured", agent.ID)
}
