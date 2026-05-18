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
	AgentID      string            `json:"agent_id" yaml:"agent_id"`
	OutputSchema map[string]string `json:"output_schema" yaml:"output_schema"`
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

func (c *Compiler) Compile(def *WorkflowDefinition) (*core.Workflow, error) {
	workflow := core.NewWorkflow(def.ID, def.Name, def.Description)

	for _, td := range def.Tasks {
		task := core.NewTask(td.ID, workflow.ID, td.Name, td.Description, td.Input, td.Dependencies)
		if td.AgentID != "" {
			task.WithAgentID(td.AgentID)
		}
		if len(td.OutputSchema) > 0 {
			task.WithOutputSchema(td.OutputSchema)
		}
		if err := workflow.AddTask(task); err != nil {
			return nil, fmt.Errorf("failed to add task %s: %w", td.ID, err)
		}
	}

	return workflow, nil
}
