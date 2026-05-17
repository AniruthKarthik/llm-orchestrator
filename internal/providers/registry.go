package providers

import (
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

// List returns a list of all registered provider names.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Clear removes all providers from the registry (primarily for testing).
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Provider)
}
