package executor

import (
	"sync"
)

type ExecutionContext struct {
	WorkflowID string

	SharedMemory map[string]any

	Mutex sync.RWMutex
}

func NewExecutionContext(
	workflowID string,
) *ExecutionContext {
	newExecContext := &ExecutionContext{
		WorkflowID:   workflowID,
		SharedMemory: make(map[string]any),
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
