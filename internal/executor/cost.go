package executor

import (
	"sync"
)

// ModelCost represents the cost per 1k tokens.
type ModelCost struct {
	PromptCost     float64
	CompletionCost float64
}

// ModelCostRegistry stores cost information for different models.
type ModelCostRegistry struct {
	costs map[string]ModelCost
	mu    sync.RWMutex
}

func NewModelCostRegistry() *ModelCostRegistry {
	r := &ModelCostRegistry{
		costs: make(map[string]ModelCost),
	}
	// Add some defaults
	r.Register("gpt-4o", ModelCost{PromptCost: 0.005, CompletionCost: 0.015})
	r.Register("gpt-3.5-turbo", ModelCost{PromptCost: 0.0005, CompletionCost: 0.0015})
	r.Register("llama-3.1-8b-instant", ModelCost{PromptCost: 0.00005, CompletionCost: 0.00005})
	return r
}

func (r *ModelCostRegistry) Register(model string, cost ModelCost) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.costs[model] = cost
}

func (r *ModelCostRegistry) Get(model string) (ModelCost, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cost, exists := r.costs[model]
	return cost, exists
}
