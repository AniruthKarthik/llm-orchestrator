package executor

import (
	"sync"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

type ExecutionContext struct {
	WorkflowID string

	SharedMemory map[string]any
	Artifacts    *core.ArtifactRegistry
	Memories     *core.MemoryRegistry

	Mutex sync.RWMutex
}

func NewExecutionContext(
	workflowID string,
	artifacts *core.ArtifactRegistry,
	memories *core.MemoryRegistry,
) *ExecutionContext {
	newExecContext := &ExecutionContext{
		WorkflowID:   workflowID,
		SharedMemory: make(map[string]any),
		Artifacts:    artifacts,
		Memories:     memories,
	}

	return newExecContext
}

func (c *ExecutionContext) Set(
	key string,
	value any,
) {

	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.SharedMemory[key] = value
}

func (c *ExecutionContext) Get(
	key string,
) (any, bool) {
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()

	val, exists := c.SharedMemory[key]

	return val, exists
}
