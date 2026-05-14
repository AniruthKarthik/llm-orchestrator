package plugin

import (
	"fmt"
	"sync"
)

type DefaultRegistry struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		plugins: make(map[string]Plugin),
	}
}

func (r *DefaultRegistry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[p.ID()]; exists {
		return fmt.Errorf("plugin already registered: %s", p.ID())
	}

	r.plugins[p.ID()] = p
	return nil
}

func (r *DefaultRegistry) Get(id string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.plugins[id]
	return p, exists
}

func (r *DefaultRegistry) List(t PluginType) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Plugin
	for _, p := range r.plugins {
		if p.Type() == t {
			result = append(result, p)
		}
	}
	return result
}
