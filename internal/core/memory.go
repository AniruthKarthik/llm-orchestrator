package core

import (
	"context"
	"sync"
)

// MemoryMessage represents a single piece of information in memory.
type MemoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Memory defines the interface for agent memory systems.
type Memory interface {
	Add(ctx context.Context, message MemoryMessage) error
	List(ctx context.Context) ([]MemoryMessage, error)
	Clear(ctx context.Context) error
}

// ShortTermMemory implements in-memory transient memory.
type ShortTermMemory struct {
	messages []MemoryMessage
	mu       sync.RWMutex
}

func NewShortTermMemory() *ShortTermMemory {
	return &ShortTermMemory{
		messages: make([]MemoryMessage, 0),
	}
}

func (m *ShortTermMemory) Add(ctx context.Context, msg MemoryMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *ShortTermMemory) List(ctx context.Context) ([]MemoryMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]MemoryMessage, len(m.messages))
	copy(cp, m.messages)
	return cp, nil
}

func (m *ShortTermMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]MemoryMessage, 0)
	return nil
}

// MemoryRegistry manages memory instances for different agents or workflows.
type MemoryRegistry struct {
	memories map[string]Memory
	mu       sync.RWMutex
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		memories: make(map[string]Memory),
	}
}

func (r *MemoryRegistry) Get(id string) (Memory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, exists := r.memories[id]
	return m, exists
}

func (r *MemoryRegistry) Register(id string, m Memory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memories[id] = m
}
