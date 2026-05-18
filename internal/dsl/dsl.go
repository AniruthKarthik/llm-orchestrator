package dsl

import (
	"encoding/json"
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"gopkg.in/yaml.v3"
)

// WorkflowDefinition represents the declarative structure of a workflow.
type WorkflowDefinition struct {
	ID          string           `json:"id" yaml:"id"`
	Name        string           `json:"name" yaml:"name"`
	Description string           `json:"description" yaml:"description"`
	Tasks       []TaskDefinition `json:"tasks" yaml:"tasks"`
}

// TaskDefinition represents the declarative structure of a task.
type TaskDefinition struct {
	ID           string            `json:"id" yaml:"id"`
	Name         string            `json:"name" yaml:"name"`
	Description  string            `json:"description" yaml:"description"`
	Input        map[string]any    `json:"input" yaml:"input"`
	Dependencies []string          `json:"dependencies" yaml:"dependencies"`
	AgentID          string            `json:"agent_id" yaml:"agent_id"`
	OutputSchema     map[string]string `json:"output_schema" yaml:"output_schema"`
	RequiresApproval bool              `json:"requires_approval" yaml:"requires_approval"`
}

// Parser defines the interface for parsing workflow definitions.
type Parser interface {
	Parse(data []byte) (*WorkflowDefinition, error)
}

// JSONParser implements the Parser interface for JSON.
type JSONParser struct{}

func (p *JSONParser) Parse(data []byte) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

// YAMLParser implements the Parser interface for YAML.
type YAMLParser struct{}

func (p *YAMLParser) Parse(data []byte) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

// Compiler converts a WorkflowDefinition into a runtime Workflow.
type Compiler struct{}

func (c *Compiler) Validate(def *WorkflowDefinition) error {
	// 1. Check for cycles
	adj := make(map[string][]string)
	for _, td := range def.Tasks {
		adj[td.ID] = td.Dependencies
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for _, td := range def.Tasks {
		if !visited[td.ID] {
			if hasCycle(td.ID) {
				return fmt.Errorf("workflow contains a cycle involving task %s", td.ID)
			}
		}
	}

	// 2. Check for missing dependencies
	taskIDs := make(map[string]bool)
	for _, td := range def.Tasks {
		taskIDs[td.ID] = true
	}

	for _, td := range def.Tasks {
		for _, dep := range td.Dependencies {
			if !taskIDs[dep] {
				return fmt.Errorf("task %s has unknown dependency: %s", td.ID, dep)
			}
		}
	}

	return nil
}

func (c *Compiler) Compile(def *WorkflowDefinition) (*core.Workflow, error) {
	if err := c.Validate(def); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	workflow := core.NewWorkflow(def.ID, def.Name, def.Description)

	for _, td := range def.Tasks {
		task := core.NewTask(td.ID, workflow.ID, td.Name, td.Description, td.Input, td.Dependencies)
		if td.AgentID != "" {
			task.WithAgentID(td.AgentID)
		}
		if len(td.OutputSchema) > 0 {
			task.WithOutputSchema(td.OutputSchema)
		}
		if td.RequiresApproval {
			task.WithApproval(true)
		}
		if err := workflow.AddTask(task); err != nil {
			return nil, fmt.Errorf("failed to add task %s: %w", td.ID, err)
		}
	}

	return workflow, nil
}
