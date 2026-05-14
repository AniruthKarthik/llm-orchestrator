package plugin

import (
	"context"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

// PluginType defines the category of a plugin.
type PluginType string

const (
	PluginTypeWorker      PluginType = "WORKER"
	PluginTypeObserver    PluginType = "OBSERVER"
	PluginTypePersistence PluginType = "PERSISTENCE"
)

// Plugin represents the base interface for all runtime extensions.
type Plugin interface {
	ID() string
	Type() PluginType
	Init(ctx context.Context) error
}

// ExecutionInterceptor allows hooking into the task execution lifecycle.
type ExecutionInterceptor interface {
	BeforeTask(ctx context.Context, task *core.Task) error
	AfterTask(ctx context.Context, task *core.Task, output map[string]any, err error) error
}

// Registry defines the interface for managing and discovering plugins.
type Registry interface {
	Register(p Plugin) error
	Get(id string) (Plugin, bool)
	List(t PluginType) []Plugin
}
