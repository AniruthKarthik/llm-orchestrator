package providers

import (
	"context"
	"fmt"
	"sync"
)

var (
	registry = make(map[string]Provider)
	mu       sync.RWMutex
)

// Register adds a provider to the global registry.
func Register(provider Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[provider.Name()] = provider
}

// Get retrieves a provider by name from the global registry.
func Get(name string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return p, nil
}

type ProviderInfo struct {
	Name   string   `json:"name"`
	Models []string `json:"models"`
}

// List returns a list of all registered provider names and their supported models.
func List(ctx context.Context) []ProviderInfo {
	mu.RLock()
	defer mu.RUnlock()
	infos := make([]ProviderInfo, 0, len(registry))
	for name, p := range registry {
		models, err := p.ListModels(ctx)
		if err != nil {
			// If error, just return empty models or log it
			models = []string{}
		}
		infos = append(infos, ProviderInfo{
			Name:   name,
			Models: models,
		})
	}
	return infos
}

// Clear removes all providers from the registry (primarily for testing).
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Provider)
}
