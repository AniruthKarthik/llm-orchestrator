package core

import (
	"context"
	"fmt"
	"sync"
)

// Tool defines the interface for external capabilities an agent can use.
type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]string
	Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}

// ToolRegistry manages the registration and retrieval of tools.
type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[t.Name()]; exists {
		return fmt.Errorf("tool already exists: %s", t.Name())
	}
	r.tools[t.Name()] = t
	return nil
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, exists := r.tools[name]
	return t, exists
}

func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// ToolPermission defines whether an agent can use a specific tool.
type ToolPermission string

const (
	PermissionAllow ToolPermission = "ALLOW"
	PermissionDeny  ToolPermission = "DENY"
)

// ToolPolicy manages tool usage permissions.
type ToolPolicy struct {
	permissions map[string]map[string]ToolPermission // AgentID -> ToolName -> Permission
	mu          sync.RWMutex
}

func NewToolPolicy() *ToolPolicy {
	return &ToolPolicy{
		permissions: make(map[string]map[string]ToolPermission),
	}
}

func (p *ToolPolicy) Grant(agentID, toolName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.permissions[agentID]; !exists {
		p.permissions[agentID] = make(map[string]ToolPermission)
	}
	p.permissions[agentID][toolName] = PermissionAllow
}

func (p *ToolPolicy) Deny(agentID, toolName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.permissions[agentID]; !exists {
		p.permissions[agentID] = make(map[string]ToolPermission)
	}
	p.permissions[agentID][toolName] = PermissionDeny
}

func (p *ToolPolicy) IsAllowed(agentID, toolName string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if agentPerms, exists := p.permissions[agentID]; exists {
		if perm, exists := agentPerms[toolName]; exists {
			return perm == PermissionAllow
		}
	}
	return false // Default to deny
}
