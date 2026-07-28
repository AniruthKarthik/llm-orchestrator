package executor

import (
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

type ExecutionContext struct {
	WorkflowID string

	sharedMemory map[string]any
	Artifacts    *core.ArtifactRegistry
	Memories     *core.MemoryRegistry
	Tools        *core.ToolRegistry
	ToolPolicy   *core.ToolPolicy

	mu sync.RWMutex
}

func NewExecutionContext(
	workflowID string,
	artifacts *core.ArtifactRegistry,
	memories *core.MemoryRegistry,
	tools *core.ToolRegistry,
	toolPolicy *core.ToolPolicy,
) *ExecutionContext {
	return &ExecutionContext{
		WorkflowID:   workflowID,
		sharedMemory: make(map[string]any),
		Artifacts:    artifacts,
		Memories:     memories,
		Tools:        tools,
		ToolPolicy:   toolPolicy,
	}
}

func (c *ExecutionContext) Set(
	key string,
	value any,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sharedMemory[key] = value
}

func (c *ExecutionContext) Get(
	key string,
) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, exists := c.sharedMemory[key]

	return val, exists
}
